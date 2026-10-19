package ingress

import (
	"strings"
	"testing"

	"github.com/ryangerardwilson/c3po/tunnel/internal/routes"
)

func TestRenderIngress(t *testing.T) {
	text, err := Render("1bdc9e04-7e31-4589-baa9-f6caf6ae9a74", "/tmp/cred.json", routes.Zone, []routes.Route{
		{Name: routes.Apex, Port: 8080},
		{Name: "notes", Port: 3000},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"protocol: http2\n",
		"hostname: ryangerardwilson.com\n",
		"service: http://127.0.0.1:8080\n",
		"hostname: notes.ryangerardwilson.com\n",
		"service: http://127.0.0.1:3000\n",
		"service: http_status:404\n",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q\n%s", want, text)
		}
	}
	if strings.Contains(text, "Tunnel"+"Secret") {
		t.Fatal("config contained a secret field")
	}
	if _, err := Render("bad\nid", "/tmp/cred.json", routes.Zone, nil); err == nil {
		t.Fatal("unsafe tunnel id was rendered")
	}
}
