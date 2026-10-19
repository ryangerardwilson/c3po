package trace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ryangerardwilson/c3po/ast/internal/config"
)

func TestGoCallersCalleesAndMethod(t *testing.T) {
	root := t.TempDir()
	must(t, os.WriteFile(filepath.Join(root, "a.go"), []byte(`package p

func Target() {}

func Start() { Target() }
`), 0o644))
	must(t, os.WriteFile(filepath.Join(root, "b.go"), []byte(`package p

type Screen struct{}

func (s *Screen) Render() {}

func (s *Screen) Draw() { s.Render() }

func (s *Screen) Other() { helper.Render() }
`), 0o644))

	cfg := config.Default()
	got, err := Query(root, cfg, "Target", "callers", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if got.Ambiguous || len(got.Callers) != 1 || got.Callers[0].Symbol != "Start" || got.Callers[0].Confidence != "statically-confirmed" {
		t.Fatalf("%+v", got.Callers)
	}
	cal, err := Query(root, cfg, "Screen.Draw", "callees", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(cal.Callees) != 1 || cal.Callees[0].Confidence != "statically-confirmed" || cal.Callees[0].PeerSymbol != "Screen.Render" {
		t.Fatalf("%+v", cal.Callees)
	}
	rev, err := Query(root, cfg, "Screen.Render", "callers", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	var confirmed, unresolved int
	for _, e := range rev.Callers {
		switch e.Confidence {
		case "statically-confirmed":
			confirmed++
		case "unresolved":
			unresolved++
		}
	}
	if confirmed != 1 || unresolved != 1 {
		t.Fatalf("render callers %+v", rev.Callers)
	}
}

func TestPythonMethodAndAmbiguousGo(t *testing.T) {
	root := t.TempDir()
	must(t, os.WriteFile(filepath.Join(root, "screen.py"), []byte(`
class Screen:
    def render(self):
        self.paint()
    def paint(self):
        return 1

def outside():
    helper()

def helper():
    return 2
`), 0o644))
	cfg := config.Default()
	got, err := Query(root, cfg, "Screen.paint", "callers", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Callers) == 0 || got.Callers[0].Confidence != "statically-confirmed" {
		t.Fatalf("%+v", got.Callers)
	}
	out, err := Query(root, cfg, "helper", "callers", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Callers) != 1 || out.Callers[0].Symbol != "outside" {
		t.Fatalf("%+v", out.Callers)
	}

	must(t, os.MkdirAll(filepath.Join(root, "one"), 0o755))
	must(t, os.MkdirAll(filepath.Join(root, "two"), 0o755))
	must(t, os.WriteFile(filepath.Join(root, "one", "a.go"), []byte("package one\nfunc Work() {}\n"), 0o644))
	must(t, os.WriteFile(filepath.Join(root, "two", "b.go"), []byte("package two\nfunc Work() {}\n"), 0o644))
	amb, err := Query(root, cfg, "Work", "callers", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !amb.Ambiguous || len(amb.Candidates) != 2 {
		t.Fatalf("%+v", amb.Candidates)
	}
}

func TestCallersExceedSearchCapAndSkipNestedRepo(t *testing.T) {
	root := t.TempDir()
	var b strings.Builder
	b.WriteString("package p\nfunc Target() {}\n")
	for i := 0; i < 55; i++ {
		b.WriteString("func c" + itoa(i) + "() { Target() }\n")
	}
	must(t, os.WriteFile(filepath.Join(root, "many.go"), []byte(b.String()), 0o644))
	nested := filepath.Join(root, "nested")
	must(t, os.MkdirAll(nested, 0o755))
	must(t, os.MkdirAll(filepath.Join(nested, ".git"), 0o755))
	must(t, os.WriteFile(filepath.Join(nested, "hidden.go"), []byte("package hidden\nfunc Target() {}\nfunc Secret() { Target() }\n"), 0o644))

	cfg := config.Default()
	got, err := Query(root, cfg, "Target", "callers", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Callers) != 55 {
		t.Fatalf("callers %d", len(got.Callers))
	}
	page, err := Query(root, cfg, "Target", "callers", 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !page.Truncated || len(page.Callers) != 10 || page.NextOffset == nil || *page.NextOffset != 10 {
		t.Fatalf("page %+v n %d", page.NextOffset, len(page.Callers))
	}
	inner, err := Query(nested, cfg, "Secret", "callees", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if inner.Resolved.Name != "Secret" || len(inner.Callees) != 1 {
		t.Fatalf("%+v", inner)
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var d []byte
	for n > 0 {
		d = append([]byte{byte('0' + n%10)}, d...)
		n /= 10
	}
	return string(d)
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func TestJSXAndEnclosingFunction(t *testing.T) {
	root := t.TempDir()
	must(t, os.MkdirAll(filepath.Join(root, "components"), 0o755))
	must(t, os.WriteFile(filepath.Join(root, "components", "avatar.tsx"), []byte(`
export function GoldenCrossAvatar() {
  return null
}
`), 0o644))
	must(t, os.WriteFile(filepath.Join(root, "components", "card.tsx"), []byte(`
function quotedScripture() {
  return "verse"
}

export function OrganisationDirectoryCard() {
  function quotedScripture() {
    return "inner"
  }
  return <GoldenCrossAvatar title={organisationHasStatementOfFaith(name)} />
}

function organisationHasStatementOfFaith(name) {
  return true
}
`), 0o644))
	cfg := config.Default()
	got, err := Query(root, cfg, "GoldenCrossAvatar", "callers", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Callers) != 1 || got.Callers[0].Symbol != "OrganisationDirectoryCard" || got.Callers[0].Kind != "jsx" {
		t.Fatalf("jsx callers %+v", got.Callers)
	}
	call, err := Query(root, cfg, "organisationHasStatementOfFaith", "callers", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(call.Callers) != 1 || call.Callers[0].Symbol != "OrganisationDirectoryCard" {
		t.Fatalf("enclosing %+v", call.Callers)
	}
}

func TestPythonImportAndRoute(t *testing.T) {
	root := t.TempDir()
	must(t, os.WriteFile(filepath.Join(root, "policy.py"), []byte("def current_runtime_policy():\n    return 1\n"), 0o644))
	must(t, os.WriteFile(filepath.Join(root, "api.py"), []byte(`
from policy import current_runtime_policy

@app.get("/churches")
def list_churches():
    return current_runtime_policy()
`), 0o644))
	cfg := config.Default()
	got, err := Query(root, cfg, "current_runtime_policy", "callers", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Callers) != 1 || got.Callers[0].Symbol != "list_churches" || got.Callers[0].Kind != "import" {
		t.Fatalf("import %+v", got.Callers)
	}
	route, err := Query(root, cfg, "list_churches", "callers", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(route.Callers) != 1 || route.Callers[0].Confidence != "framework-inferred" || route.Callers[0].Symbol != "route GET /churches" {
		t.Fatalf("route %+v", route.Callers)
	}
}

func TestGoInterfaceAndStored(t *testing.T) {
	root := t.TempDir()
	must(t, os.WriteFile(filepath.Join(root, "m.go"), []byte(`package p

type Client struct{}

func (c *Client) AggregateSpans() {}

type Metrics struct {
	client    *Client
	collectL1 func()
}

func (m *Metrics) gather() { m.collectL1 = m.AggregateSpans }

func (m *Metrics) AggregateSpans() {}

func (m *Metrics) collect() {
	m.client.AggregateSpans()
	m.collectL1()
}

type Collector interface{ Collect() }

type cpu struct{}

func (c cpu) Collect() {}

func run(c Collector) { c.Collect() }
`), 0o644))
	cfg := config.Default()
	got, err := Query(root, cfg, "Client.AggregateSpans", "callers", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Callers) == 0 || got.Callers[0].Confidence != "statically-confirmed" {
		t.Fatalf("field call %+v", got.Callers)
	}
	iface, err := Query(root, cfg, "cpu.Collect", "callers", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, e := range iface.Callers {
		if e.Symbol == "run" && e.Confidence == "interface-inferred" {
			found = true
		}
	}
	if !found {
		t.Fatalf("interface %+v", iface.Callers)
	}
	stored, err := Query(root, cfg, "Metrics.AggregateSpans", "callers", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	found = false
	for _, e := range stored.Callers {
		if e.Symbol == "Metrics.collect" && e.Kind == "method-value" && e.Confidence == "statically-confirmed" {
			found = true
		}
	}
	if !found {
		t.Fatalf("stored method %+v", stored.Callers)
	}
}

func TestPythonFrameworkEdges(t *testing.T) {
	root := t.TempDir()
	must(t, os.WriteFile(filepath.Join(root, "policy.py"), []byte("def current_runtime_policy():\n    return 1\n"), 0o644))
	must(t, os.WriteFile(filepath.Join(root, "api.py"), []byte(`
from policy import current_runtime_policy

@app.get("/churches")
def list_churches(policy=Depends(current_runtime_policy)):
    return policy

def create_app(app):
    app.include_router(router)
    app.add_event_handler("startup", current_runtime_policy)

def mount():
    @app.get("/nested")
    def inner():
        return current_runtime_policy()
`), 0o644))
	cfg := config.Default()
	got, err := Query(root, cfg, "current_runtime_policy", "callers", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	var dep, cb bool
	for _, e := range got.Callers {
		if e.Symbol == "list_churches" && e.Kind == "dependency" && e.Confidence == "framework-inferred" {
			dep = true
		}
		if e.Symbol == "create_app" && e.Kind == "callback" && e.Confidence == "framework-inferred" {
			cb = true
		}
	}
	if !dep || !cb {
		t.Fatalf("framework callers %+v", got.Callers)
	}
	app, err := Query(root, cfg, "create_app", "callees", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	routed := false
	for _, e := range app.Callees {
		if e.Kind == "router" && e.Confidence == "configuration-inferred" && e.CalleeText == "router" {
			routed = true
		}
	}
	if !routed {
		t.Fatalf("router %+v", app.Callees)
	}
	inner, err := Query(root, cfg, "inner", "callers", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(inner.Callers) != 1 || inner.Callers[0].Symbol != "route GET /nested" {
		t.Fatalf("nested route %+v", inner.Callers)
	}
}

func TestGoImportCall(t *testing.T) {
	root := t.TempDir()
	must(t, os.MkdirAll(filepath.Join(root, "work"), 0o755))
	must(t, os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\n\nimport \"example.com/app/work\"\n\nfunc main() {\n\twork.Start()\n}\n"), 0o644))
	must(t, os.WriteFile(filepath.Join(root, "work", "work.go"), []byte("package work\n\nfunc Start() { Hidden() }\n"), 0o644))
	cal, err := Query(root, config.Default(), "main", "callees", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(cal.Callees) != 1 || cal.Callees[0].PeerSymbol != "Start" || cal.Callees[0].Confidence != "statically-confirmed" {
		t.Fatalf("%+v", cal.Callees)
	}
}

func TestStringConversionIsNotACall(t *testing.T) {
	root := t.TempDir()
	must(t, os.MkdirAll(filepath.Join(root, "parse"), 0o755))
	must(t, os.WriteFile(filepath.Join(root, "show.go"), []byte("package show\n\nfunc Show(buf []byte) string {\n\treturn string(buf)\n}\n"), 0o644))
	must(t, os.WriteFile(filepath.Join(root, "parse", "lex.go"), []byte("package parse\n\ntype lexer struct{}\n\nfunc (l *lexer) string() {}\n"), 0o644))
	cal, err := Query(root, config.Default(), "Show", "callees", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range cal.Callees {
		if e.Confidence == "statically-confirmed" {
			t.Fatalf("%+v", cal.Callees)
		}
	}
}
