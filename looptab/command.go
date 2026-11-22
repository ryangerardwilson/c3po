// Package looptab is c3po looptab. It registers jobs files and installs them
// onto the Looptab scheduler file. The r2d2 looptab screen calls JobsFile and
// Install and does not write the scheduler file itself.
package looptab

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/ryangerardwilson/c3po/internal/version"
)

// CommandHelp is the help for c3po looptab.
const CommandHelp = `c3po looptab
register jobs files and install them onto looptab

  c3po looptab list
    list registered jobs files
  c3po looptab register <path>
    register a jobs file and install it onto looptab
  c3po looptab remove <name>
    remove a registered jobs file and install the rest
  c3po looptab show <name>
    print one registered jobs file
  c3po looptab help
    show this help
  c3po looptab version
    print the c3po version

A jobs file is a crontab-like looptab document. Registrations are stored in
the sqlx database c3po. The jobs file is edited with r2d2 looptab, which
installs it when the editor exits. r2d2 looptab also runs one job now.
`

const jobsTemplate = `# Schedules: now, hourly, hourly at <minute>, daily <time>, weekdays <time>
# A job line is a schedule and a command. ? and : on the next lines notify.
#
# daily 5am,9am,12pm,4pm c3po gmail tidy
# ? c3po notify heading "gmail" body "morning inbox review finished"
# : c3po notify heading "gmail" body "morning inbox review failed"
`

// paths are the catalog record, the installed scheduler file, and the jobs
// file created when nothing is registered.
type paths struct {
	record, schedule string
	jobs             string
}

func defaultPaths() (paths, error) {
	record, err := recordPath()
	if err != nil {
		return paths{}, err
	}
	schedule, err := schedulePath()
	if err != nil {
		return paths{}, err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return paths{}, err
	}
	return paths{
		record:   record,
		schedule: schedule,
		jobs:     filepath.Join(home, ".config", "c3po", "looptab", "jobs"),
	}, nil
}

// JobsFile returns the jobs file r2d2 looptab edits, the first registered
// file. With no registered file, it creates ~/.config/c3po/looptab/jobs from
// the comment template, registers it as jobs, and installs it.
func JobsFile() (string, error) {
	p, err := defaultPaths()
	if err != nil {
		return "", err
	}
	return p.first()
}

// Install writes every registered jobs file onto the Looptab scheduler file
// and returns how many files it installed.
func Install() (int, error) {
	p, err := defaultPaths()
	if err != nil {
		return 0, err
	}
	return p.installAll()
}

func (p paths) first() (string, error) {
	cat, err := loadCatalog(p.record)
	if err != nil {
		return "", err
	}
	if len(cat.Files) > 0 {
		return cat.Files[0].Path, nil
	}
	if err := os.MkdirAll(filepath.Dir(p.jobs), 0o755); err != nil {
		return "", err
	}
	if _, err := os.Stat(p.jobs); errors.Is(err, os.ErrNotExist) {
		if err := os.WriteFile(p.jobs, []byte(jobsTemplate), 0o644); err != nil {
			return "", err
		}
	} else if err != nil {
		return "", err
	}
	cat.Files = []registered{{Name: fileName(p.jobs), Path: p.jobs}}
	if err := saveCatalog(p.record, cat); err != nil {
		return "", err
	}
	if err := installJobs(p.schedule, cat.Files); err != nil {
		return "", err
	}
	return p.jobs, nil
}

func (p paths) installAll() (int, error) {
	cat, err := loadCatalog(p.record)
	if err != nil {
		return 0, err
	}
	if err := installJobs(p.schedule, cat.Files); err != nil {
		return 0, err
	}
	return len(cat.Files), nil
}

// Command serves c3po looptab. args are the arguments after the app name.
func Command(args []string, stdout, stderr io.Writer) int {
	p, err := defaultPaths()
	if err != nil {
		fmt.Fprintf(stderr, "looptab: %s\n", err.Error())
		return 1
	}
	return (&desk{stdout: stdout, stderr: stderr, paths: p}).run(args)
}

type desk struct {
	stdout, stderr io.Writer
	paths
}

func (d *desk) run(args []string) int {
	if len(args) == 0 {
		fmt.Fprint(d.stdout, CommandHelp)
		return 0
	}
	switch args[0] {
	case "help", "-h", "--help":
		if len(args) != 1 {
			return d.usage()
		}
		fmt.Fprint(d.stdout, CommandHelp)
		return 0
	case "version", "--version", "-v":
		if len(args) != 1 {
			return d.usage()
		}
		fmt.Fprintln(d.stdout, "c3po "+version.String())
		return 0
	case "list":
		if len(args) != 1 {
			return d.usage()
		}
		return d.list()
	case "register":
		if len(args) != 2 {
			return d.usage()
		}
		return d.register(args[1])
	case "remove":
		if len(args) != 2 {
			return d.usage()
		}
		return d.remove(args[1])
	case "show":
		if len(args) != 2 {
			return d.usage()
		}
		return d.show(args[1])
	default:
		return d.usage()
	}
}

func (d *desk) usage() int {
	fmt.Fprint(d.stderr, CommandHelp)
	return 2
}

func (d *desk) list() int {
	cat, err := loadCatalog(d.record)
	if err != nil {
		return d.fail(err)
	}
	if len(cat.Files) == 0 {
		fmt.Fprintln(d.stdout, "no jobs files")
		return 0
	}
	for _, file := range cat.Files {
		fmt.Fprintf(d.stdout, "%s\t%s\n", file.Name, file.Path)
	}
	return 0
}

func (d *desk) register(path string) int {
	abs, err := filepath.Abs(path)
	if err != nil {
		return d.fail(err)
	}
	sched, err := filepath.Abs(d.schedule)
	if err != nil {
		return d.fail(err)
	}
	if abs == sched {
		fmt.Fprintln(d.stderr, "looptab: the scheduler file is installed from registered jobs files")
		return 2
	}
	info, err := os.Stat(abs)
	if err != nil || !info.Mode().IsRegular() {
		fmt.Fprintln(d.stderr, "looptab: jobs file must be a regular file")
		return 2
	}
	name := fileName(abs)
	if name == "" {
		fmt.Fprintln(d.stderr, "looptab: jobs file name must be one word")
		return 2
	}
	cat, err := loadCatalog(d.record)
	if err != nil {
		return d.fail(err)
	}
	for _, file := range cat.Files {
		if file.Name == name {
			fmt.Fprintf(d.stderr, "looptab: jobs file %s is already registered\n", name)
			return 2
		}
	}
	cat.Files = append(cat.Files, registered{Name: name, Path: abs})
	if err := saveCatalog(d.record, cat); err != nil {
		return d.fail(err)
	}
	return d.install(cat)
}

func (d *desk) remove(name string) int {
	cat, err := loadCatalog(d.record)
	if err != nil {
		return d.fail(err)
	}
	next := catalog{}
	found := false
	for _, file := range cat.Files {
		if file.Name == name {
			found = true
			continue
		}
		next.Files = append(next.Files, file)
	}
	if !found {
		fmt.Fprintf(d.stderr, "looptab: unknown jobs file %s\n", name)
		return 2
	}
	if next.Files == nil {
		next.Files = []registered{}
	}
	if err := saveCatalog(d.record, next); err != nil {
		return d.fail(err)
	}
	return d.install(next)
}

func (d *desk) show(name string) int {
	cat, err := loadCatalog(d.record)
	if err != nil {
		return d.fail(err)
	}
	for _, file := range cat.Files {
		if file.Name != name {
			continue
		}
		body, err := os.ReadFile(file.Path)
		if err != nil {
			return d.fail(err)
		}
		fmt.Fprintf(d.stdout, "%s\t%s\n", file.Name, file.Path)
		fmt.Fprint(d.stdout, string(body))
		if len(body) == 0 || body[len(body)-1] != '\n' {
			fmt.Fprintln(d.stdout)
		}
		return 0
	}
	fmt.Fprintf(d.stderr, "looptab: unknown jobs file %s\n", name)
	return 2
}

func (d *desk) install(cat catalog) int {
	if err := installJobs(d.schedule, cat.Files); err != nil {
		return d.fail(err)
	}
	fmt.Fprintf(d.stdout, "installed %d jobs files\n", len(cat.Files))
	return 0
}

func (d *desk) fail(err error) int {
	fmt.Fprintf(d.stderr, "looptab: %s\n", err.Error())
	return 1
}

func fileName(path string) string {
	base := strings.ToLower(strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)))
	if base == "" || base == "." || strings.ContainsAny(base, " \t/") {
		return ""
	}
	return base
}
