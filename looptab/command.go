package looptab

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/ryangerardwilson/c3po/internal/version"
)

// CommandHelp is the help for c3po looptab.
const CommandHelp = `c3po looptab
register jobs files and edit them in vim

  c3po looptab
    open the first registered jobs file in vim, then install it onto looptab
  c3po looptab edit [name]
    open one registered file in vim, then install
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
the sqlx database c3po. c3po tui looptab runs one job now.
`

const jobsTemplate = `# Schedules: now, hourly, hourly at <minute>, daily <time>, weekdays <time>
# A job line is a schedule and a command. ? and : on the next lines notify.
#
# daily 5am,9am,12pm,4pm c3po gmail tidy
# ? notify heading "gmail" body "morning inbox review finished"
# : notify heading "gmail" body "morning inbox review failed"
`

// Command serves c3po looptab. args are the arguments after the app name.
func Command(args []string, stdout, stderr io.Writer) int {
	record, err := recordPath()
	if err != nil {
		fmt.Fprintf(stderr, "looptab: %s\n", err.Error())
		return 1
	}
	schedule, err := schedulePath()
	if err != nil {
		fmt.Fprintf(stderr, "looptab: %s\n", err.Error())
		return 1
	}
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(stderr, "looptab: %s\n", err.Error())
		return 1
	}
	return (&desk{
		stdout:   stdout,
		stderr:   stderr,
		record:   record,
		schedule: schedule,
		jobs:     filepath.Join(home, ".config", "c3po", "looptab", "jobs"),
		edit:     openVim,
	}).run(args)
}

type desk struct {
	stdout, stderr   io.Writer
	record, schedule string
	jobs             string
	edit             func(path string) error
}

func (d *desk) run(args []string) int {
	if len(args) == 0 {
		return d.editFirst()
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
	case "edit":
		if len(args) > 2 {
			return d.usage()
		}
		name := ""
		if len(args) == 2 {
			name = args[1]
		}
		return d.editNamed(name)
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

func (d *desk) editFirst() int {
	cat, err := loadCatalog(d.record)
	if err != nil {
		return d.fail(err)
	}
	if len(cat.Files) == 0 {
		if err := os.MkdirAll(filepath.Dir(d.jobs), 0o755); err != nil {
			return d.fail(err)
		}
		if _, err := os.Stat(d.jobs); os.IsNotExist(err) {
			if err := os.WriteFile(d.jobs, []byte(jobsTemplate), 0o644); err != nil {
				return d.fail(err)
			}
		} else if err != nil {
			return d.fail(err)
		}
		name := fileName(d.jobs)
		cat.Files = []registered{{Name: name, Path: d.jobs}}
		if err := saveCatalog(d.record, cat); err != nil {
			return d.fail(err)
		}
		if code := d.install(cat); code != 0 {
			return code
		}
	}
	return d.open(cat.Files[0].Path, cat)
}

func (d *desk) editNamed(name string) int {
	if name == "" {
		return d.editFirst()
	}
	cat, err := loadCatalog(d.record)
	if err != nil {
		return d.fail(err)
	}
	for _, file := range cat.Files {
		if file.Name == name {
			return d.open(file.Path, cat)
		}
	}
	fmt.Fprintf(d.stderr, "looptab: unknown jobs file %s\n", name)
	return 2
}

func (d *desk) open(path string, cat catalog) int {
	if d.edit == nil {
		fmt.Fprintln(d.stderr, "vim is not on PATH")
		return 1
	}
	if err := d.edit(path); err != nil {
		if err.Error() == "vim is not on PATH" {
			fmt.Fprintln(d.stderr, err.Error())
			return 1
		}
		return d.fail(err)
	}
	return d.install(cat)
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
