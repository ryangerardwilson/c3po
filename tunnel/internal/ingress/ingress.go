package ingress

import (
	"fmt"
	"strings"

	"github.com/ryangerardwilson/c3po/tunnel/internal/routes"
)

// Render writes a cloudflared config that forwards each route to localhost.
func Render(tunnelID, credPath, zone string, items []routes.Route) (string, error) {
	if err := plain(tunnelID); err != nil {
		return "", fmt.Errorf("tunnel id: %w", err)
	}
	if err := plain(credPath); err != nil {
		return "", fmt.Errorf("credentials path: %w", err)
	}
	if zone == "" {
		zone = routes.Zone
	}
	var b strings.Builder
	fmt.Fprintf(&b, "tunnel: %s\n", tunnelID)
	fmt.Fprintf(&b, "credentials-file: %s\n", quote(credPath))
	b.WriteString("protocol: http2\n")
	b.WriteString("ingress:\n")
	for _, route := range items {
		if err := routes.CheckPort(route.Port); err != nil {
			return "", err
		}
		_, host, err := routes.Normalize(zone, route.Name)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&b, "  - hostname: %s\n", host)
		fmt.Fprintf(&b, "    service: http://127.0.0.1:%d\n", route.Port)
	}
	b.WriteString("  - service: http_status:404\n")
	return b.String(), nil
}

func plain(value string) error {
	if value == "" || strings.ContainsAny(value, "\r\n\"") {
		return fmt.Errorf("empty or unsafe value")
	}
	return nil
}

func quote(value string) string {
	if strings.ContainsAny(value, " :#") {
		return "\"" + value + "\""
	}
	return value
}
