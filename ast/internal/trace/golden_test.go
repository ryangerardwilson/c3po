package trace

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ryangerardwilson/c3po/ast/internal/config"
)

// Golden corpus. Recall tests require edges that must exist.
// Precision tests require edges that must not be invented.

func TestGoldenRecallPythonMultilineAndNested(t *testing.T) {
	root := t.TempDir()
	must(t, os.WriteFile(filepath.Join(root, "policy.py"), []byte("def current_runtime_policy():\n    return 1\n"), 0o644))
	must(t, os.WriteFile(filepath.Join(root, "engine.py"), []byte(`
from policy import current_runtime_policy

class PromiseEngine:
    def other(self) -> str:
        return "no"

    def promise(
        self,
        policy: str,
    ) -> str:
        return current_runtime_policy()

class Card:
    def quotedScripture(self) -> str:
        return "verse"

    def render(self) -> str:
        def inner() -> str:
            return current_runtime_policy()
        return inner()
`), 0o644))
	cfg := config.Default()
	start, end, ok := SymbolSpan(filepath.Join(root, "engine.py"), "engine.py", "PromiseEngine.promise")
	if !ok {
		t.Fatal("missing promise span")
	}
	body, err := os.ReadFile(filepath.Join(root, "engine.py"))
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(string(body), "\n")
	got := strings.Join(lines[start-1:end], "\n")
	if !strings.Contains(got, "return current_runtime_policy()") || strings.Contains(got, "def other") {
		t.Fatalf("span %d-%d does not contain only the promise body:\n%s", start, end, got)
	}
	callees, err := Query(root, cfg, "PromiseEngine.promise", "callees", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !hasEdge(callees.Callees, "PromiseEngine.promise", "current_runtime_policy", "import") {
		t.Fatalf("promise callees %+v", callees.Callees)
	}
	callers, err := Query(root, cfg, "current_runtime_policy", "callers", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !hasEdge(callers.Callers, "Card.render.inner", "current_runtime_policy", "") {
		t.Fatalf("nested caller missing %+v", callers.Callers)
	}
}

func TestGoldenPrecisionNoGuessedContainer(t *testing.T) {
	root := t.TempDir()
	must(t, os.WriteFile(filepath.Join(root, "engine.py"), []byte(`
def current_runtime_policy():
    return 1

class Card:
    def quotedScripture(self) -> str:
        return "verse"

    def render(self) -> str:
        def inner() -> str:
            return current_runtime_policy()
        return inner()
`), 0o644))
	got, err := Query(root, config.Default(), "current_runtime_policy", "callers", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range got.Callers {
		if e.Symbol == "Card" || e.Symbol == "Card.render" || e.Symbol == "Card.quotedScripture" || e.Symbol == "quotedScripture" {
			t.Fatalf("call attributed to the wrong symbol %+v", got.Callers)
		}
	}
	if len(got.Callers) != 1 || got.Callers[0].Symbol != "Card.render.inner" {
		t.Fatalf("precision callers %+v", got.Callers)
	}
}

func TestGoldenRecallJSXMultilineComponent(t *testing.T) {
	root := t.TempDir()
	must(t, os.MkdirAll(filepath.Join(root, "components"), 0o755))
	must(t, os.WriteFile(filepath.Join(root, "components", "avatar.tsx"), []byte(`
export function GoldenCrossAvatar(props: { title: boolean }) {
  return null
}
`), 0o644))
	must(t, os.WriteFile(filepath.Join(root, "components", "card.tsx"), []byte(`
export default function OrganisationDirectoryCard({
  organisation,
}: { organisation: string }) {
  const listingRating = organisationListingRating(organisation);
  const badges = ["a"].map((badge) => badge);
  return (
    <GoldenCrossAvatar title={organisationHasStatementOfFaith(organisation)} />
  )
}

export const PublicUserProfile = (
  name: string,
) => (
  <GoldenCrossAvatar title={false} />
)

export function organisationHasStatementOfFaith(name: string) {
  return true
}
`), 0o644))
	cfg := config.Default()
	got, err := Query(root, cfg, "GoldenCrossAvatar", "callers", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	var card, profile bool
	for _, e := range got.Callers {
		if e.Kind != "jsx" {
			continue
		}
		if e.Symbol == "OrganisationDirectoryCard" && strings.Contains(e.File, "card.tsx") {
			card = true
		}
		if e.Symbol == "PublicUserProfile" {
			profile = true
		}
		if e.Symbol == "quotedScripture" || e.Symbol == "listingRating" {
			t.Fatalf("jsx attributed to %s %+v", e.Symbol, got.Callers)
		}
	}
	if !card || !profile {
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

func TestGoldenJSXNamesEnclosingFunction(t *testing.T) {
	root := t.TempDir()
	must(t, os.WriteFile(filepath.Join(root, "listing.tsx"), []byte(`
export function OrganisationDirectoryCard(props: { business: string }) {
  return null
}

export default function OrganisationsListing() {
  const visibleBusinesses = useMemo(
    () => items.filter((item) => item.ok),
    [items],
  );
  return (
    <section>
      {visibleBusinesses.map((business) => (
        <OrganisationDirectoryCard business={business} />
      ))}
    </section>
  );
}

it("renders the organisation card", () => {
  const html = renderToStaticMarkup(
    <OrganisationDirectoryCard business="1" />,
  );
  return html;
});

it("renders the signed card", () => {
  const html = renderToStaticMarkup(
    <OrganisationDirectoryCard business="2" />,
  );
  return html;
});
`), 0o644))
	got, err := Query(root, config.Default(), "OrganisationDirectoryCard", "callers", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	var listing, first, second bool
	for _, e := range got.Callers {
		switch e.Symbol {
		case "OrganisationsListing":
			listing = true
		case "renders the organisation card":
			first = true
		case "renders the signed card":
			second = true
		case "visibleBusinesses", "html":
			t.Fatalf("nearby binding used as the function %+v", got.Callers)
		}
	}
	if !listing || !first || !second {
		t.Fatalf("callers %+v", got.Callers)
	}
}

func TestGoldenRecallGoMethodValueAndInterface(t *testing.T) {
	root := t.TempDir()
	must(t, os.WriteFile(filepath.Join(root, "m.go"), []byte(`package p

type Collector interface{ Collect() }

type cpu struct{}

func (c cpu) Collect() {}

var collector Collector = cpu{}

func tick() { collector.Collect() }

type work struct{ fn func() }

type Metrics struct {
	collectL1 func()
}

func (m *Metrics) AggregateSpans() {}

func (m *Metrics) gather() {
	m.collectL1 = m.AggregateSpans
	item := work{fn: m.collectL1}
	item.fn()
}
`), 0o644))
	cfg := config.Default()
	iface, err := Query(root, cfg, "cpu.Collect", "callers", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, e := range iface.Callers {
		if e.Symbol == "tick" && e.Kind == "interface" && e.Confidence == "interface-inferred" && strings.Contains(e.Reason, "Collector") {
			found = true
		}
		if e.Symbol == "tick" && e.Kind == "direct" {
			t.Fatalf("interface call labeled direct %+v", iface.Callers)
		}
	}
	if !found {
		t.Fatalf("interface callers %+v", iface.Callers)
	}
	stored, err := Query(root, cfg, "Metrics.AggregateSpans", "callers", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	found = false
	for _, e := range stored.Callers {
		if e.Symbol == "Metrics.gather" && e.Kind == "method-value" && e.Confidence == "statically-confirmed" && strings.Contains(e.Reason, "stored") {
			found = true
		}
		if e.Kind == "direct" {
			t.Fatalf("method value labeled direct %+v", stored.Callers)
		}
	}
	if !found {
		t.Fatalf("stored callers %+v", stored.Callers)
	}
}

func TestGoldenImpactConclusion(t *testing.T) {
	root := t.TempDir()
	must(t, os.WriteFile(filepath.Join(root, "a.py"), []byte("def leaf():\n    return 1\n\ndef root():\n    return leaf()\n"), 0o644))
	got, err := Query(root, config.Default(), "leaf", "impact", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if got.Assessment == nil || got.Assessment.Conclusion == "" || got.Assessment.Direction != "both" {
		t.Fatalf("assessment %+v", got.Assessment)
	}
	if got.Assessment.MaxDepth != 6 || got.Assessment.PathLimit != 8 {
		t.Fatalf("bounds %+v", got.Assessment)
	}
	if len(got.Upstream) == 0 {
		t.Fatalf("missing upstream %+v", got.Upstream)
	}
	raw, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"callees":[]`) && !strings.Contains(string(raw), `"callees": []`) {
		t.Fatalf("empty callees must be present: %s", raw)
	}
	if got.Assessment.UnresolvedEdges > 0 && got.Assessment.Conclusion != "partial" {
		t.Fatal("unresolved result must be partial")
	}
}

func hasEdge(es []Edge, symbol, peer, kind string) bool {
	for _, e := range es {
		if e.Symbol != symbol {
			continue
		}
		if peer != "" && e.PeerSymbol != peer && e.CalleeText != peer {
			continue
		}
		if kind != "" && e.Kind != kind {
			continue
		}
		return true
	}
	return false
}
