package routes

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/ryangerardwilson/c3po/internal/xdg"
)

const (
	Zone = "ryangerardwilson.com"
	Apex = "@"
)

// Route is one public name forwarded to a port on this laptop.
type Route struct {
	Name string `json:"name"`
	Port int    `json:"port"`
}

// File is the route list the background service watches.
type File struct {
	Zone   string  `json:"zone"`
	Routes []Route `json:"routes"`
}

func Default(zone string) File {
	if zone == "" {
		zone = Zone
	}
	return File{Zone: zone, Routes: []Route{{Name: Apex, Port: 8080}}}
}

func ConfigDir(home string) string {
	base := xdg.ConfigBase(home)
	dest := xdg.App(base, "tunnel")
	_ = xdg.Adopt(dest, filepath.Join(base, "tunnel"))
	return dest
}

func Path(home string) string {
	return filepath.Join(ConfigDir(home), "routes.json")
}

func Load(path string) (File, error) {
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Default(Zone), nil
	}
	if err != nil {
		return File{}, err
	}
	var file File
	if err := json.Unmarshal(raw, &file); err != nil {
		return File{}, fmt.Errorf("read routes: %w", err)
	}
	if file.Zone == "" {
		file.Zone = Zone
	}
	if file.Routes == nil {
		file.Routes = []Route{}
	}
	if err := validate(file); err != nil {
		return File{}, err
	}
	return file, nil
}

func validate(file File) error {
	for _, route := range file.Routes {
		if err := checkRoute(file.Zone, route); err != nil {
			return err
		}
	}
	return nil
}

func checkRoute(zone string, route Route) error {
	label, _, err := Normalize(zone, route.Name)
	if err != nil {
		return err
	}
	if route.Name != Apex && route.Name != "" && label != route.Name {
		return fmt.Errorf("route name %q is not a normalized label", route.Name)
	}
	return CheckPort(route.Port)
}

func Save(path string, file File) error {
	if file.Zone == "" {
		file.Zone = Zone
	}
	if file.Routes == nil {
		file.Routes = []Route{}
	}
	raw, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// EnsureFile returns the saved list, writing the apex route when the file is absent.
func EnsureFile(path string) (File, error) {
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		file := Default(Zone)
		if err := Save(path, file); err != nil {
			return File{}, err
		}
		return file, nil
	}
	return Load(path)
}

var labelRE = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$`)

// Normalize accepts a label, a full name under the zone, or @ for the apex.
func Normalize(zone, input string) (name, hostname string, err error) {
	zone = strings.ToLower(strings.TrimSpace(zone))
	if zone == "" {
		zone = Zone
	}
	raw := strings.ToLower(strings.TrimSpace(input))
	raw = strings.TrimSuffix(raw, ".")
	switch raw {
	case "", Apex, zone:
		return Apex, zone, nil
	}
	suffix := "." + zone
	if strings.HasSuffix(raw, suffix) {
		raw = strings.TrimSuffix(raw, suffix)
	}
	if strings.Contains(raw, ".") || !labelRE.MatchString(raw) {
		return "", "", fmt.Errorf("use one label under %s, such as notes", zone)
	}
	return raw, raw + "." + zone, nil
}

func Hostname(zone, name string) string {
	if name == Apex || name == "" {
		if zone == "" {
			return Zone
		}
		return zone
	}
	return name + "." + zone
}

func CheckPort(port int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("port must be from 1 to 65535")
	}
	return nil
}

func ParsePort(text string) (int, error) {
	port, err := strconv.Atoi(strings.TrimSpace(text))
	if err != nil {
		return 0, fmt.Errorf("port must be from 1 to 65535")
	}
	if err := CheckPort(port); err != nil {
		return 0, err
	}
	return port, nil
}

// Upsert adds a route or replaces the port of an existing name.
func Upsert(file File, name string, port int) (File, string, error) {
	label, hostname, err := Normalize(file.Zone, name)
	if err != nil {
		return File{}, "", err
	}
	if err := CheckPort(port); err != nil {
		return File{}, "", err
	}
	next := file
	next.Routes = append([]Route(nil), file.Routes...)
	for i, route := range next.Routes {
		if route.Name == label {
			next.Routes[i].Port = port
			return next, hostname, nil
		}
	}
	next.Routes = append(next.Routes, Route{Name: label, Port: port})
	return next, hostname, nil
}

func Remove(file File, name string) (File, error) {
	label, _, err := Normalize(file.Zone, name)
	if err != nil {
		return File{}, err
	}
	next := file
	next.Routes = make([]Route, 0, len(file.Routes))
	found := false
	for _, route := range file.Routes {
		if route.Name == label {
			found = true
			continue
		}
		next.Routes = append(next.Routes, route)
	}
	if !found {
		return File{}, fmt.Errorf("no route named %s", label)
	}
	return next, nil
}

func Same(a, b File) bool {
	if a.Zone != b.Zone || len(a.Routes) != len(b.Routes) {
		return false
	}
	for i := range a.Routes {
		if a.Routes[i] != b.Routes[i] {
			return false
		}
	}
	return true
}
