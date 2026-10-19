package tunnel

import (
	"fmt"
	"net"
	"os/exec"
	"strconv"
	"time"

	"github.com/ryangerardwilson/c3po/tunnel/internal/routes"
)

// unit is the systemd user unit that runs c3po tunnel run.
const unit = "tunnel.service"

func (a *app) start(_ []string) int {
	if err := a.system("start", unit); err != nil {
		return 1
	}
	if err := a.system("--quiet", "is-active", unit); err != nil {
		fmt.Fprintln(a.Err, "tunnel did not start")
		return 1
	}
	fmt.Fprintln(a.Out, "tunnel started")
	return 0
}

func (a *app) stop(_ []string) int {
	if err := a.system("stop", unit); err != nil {
		return 1
	}
	fmt.Fprintln(a.Out, "tunnel stopped")
	return 0
}

func (a *app) status(_ []string) int {
	if err := a.system("status", unit, "--no-pager"); err != nil {
		return 1
	}
	return 0
}

func (a *app) logs(_ []string) int {
	journal := a.Journal
	if journal == nil {
		journal = a.journal
	}
	if err := journal(); err != nil {
		return 1
	}
	return 0
}

func (a *app) journal() error {
	cmd := exec.Command("journalctl", "--user", "-u", unit, "-f")
	cmd.Stdout = a.Out
	cmd.Stderr = a.Err
	return cmd.Run()
}

// system runs systemctl --user with args, streaming its output.
func (a *app) system(args ...string) error {
	if a.Service != nil {
		return a.Service(args...)
	}
	cmd := exec.Command("systemctl", append([]string{"--user"}, args...)...)
	cmd.Stdout = a.Out
	cmd.Stderr = a.Err
	return cmd.Run()
}

func (a *app) list() error {
	file, err := routes.EnsureFile(routes.Path(a.Home))
	if err != nil {
		return err
	}
	if len(file.Routes) == 0 {
		fmt.Fprintln(a.Out, "no routes")
		return nil
	}
	for _, route := range file.Routes {
		state := "down"
		if listening(route.Port) {
			state = "up"
		}
		fmt.Fprintf(a.Out, "%-8s  %-40s  %5d  %s\n", route.Name, routes.Hostname(file.Zone, route.Name), route.Port, state)
	}
	return nil
}

func listening(port int) bool {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)), 200*time.Millisecond)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}
