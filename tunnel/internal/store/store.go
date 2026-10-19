package store

import (
	"context"
	"fmt"
	"time"

	"github.com/ryangerardwilson/c3po/tunnel/internal/routes"
	"github.com/ryangerardwilson/c3po/tunnel/internal/secrets"
)

type Ensure func(ctx context.Context, zone, hostname, tunnelID string) error

// Add publishes DNS first, then saves the route. A refused name is not saved.
func Add(home, name, portText string, load func() (secrets.Tunnel, error), ensure Ensure) (routes.File, string, error) {
	port, err := routes.ParsePort(portText)
	if err != nil {
		return routes.File{}, "", err
	}
	file, err := routes.EnsureFile(routes.Path(home))
	if err != nil {
		return routes.File{}, "", err
	}
	next, host, err := routes.Upsert(file, name, port)
	if err != nil {
		return routes.File{}, "", err
	}
	if err := publish(next.Zone, host, load, ensure); err != nil {
		return routes.File{}, "", err
	}
	if err := routes.Save(routes.Path(home), next); err != nil {
		return routes.File{}, "", err
	}
	return next, host, nil
}

// publish makes sure the hostname points at the tunnel. With no ensure,
// DNS is left alone.
func publish(zone, host string, load func() (secrets.Tunnel, error), ensure Ensure) error {
	if ensure == nil {
		return nil
	}
	if load == nil {
		return fmt.Errorf("missing credentials")
	}
	tunnel, err := load()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if zone == "" {
		zone = tunnel.Zone
	}
	return ensure(ctx, zone, host, tunnel.ID)
}

func Remove(home, name string) (routes.File, error) {
	file, err := routes.EnsureFile(routes.Path(home))
	if err != nil {
		return routes.File{}, err
	}
	next, err := routes.Remove(file, name)
	if err != nil {
		return routes.File{}, err
	}
	if err := routes.Save(routes.Path(home), next); err != nil {
		return routes.File{}, err
	}
	return next, nil
}
