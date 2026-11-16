package idealake

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ryangerardwilson/c3po/internal/sqlxdb"
)

func isolate(t *testing.T) {
	t.Helper()
	t.Setenv("SQLX_ROOT", t.TempDir())
	prevEmbed := embedFn
	prevTerm := stdinTerminal
	embedFn = func(inputs []string) ([][]float64, error) {
		return scriptedEmbed(inputs)
	}
	t.Cleanup(func() {
		embedFn = prevEmbed
		stdinTerminal = prevTerm
	})
}

func scriptedEmbed(inputs []string) ([][]float64, error) {
	out := make([][]float64, len(inputs))
	for i, in := range inputs {
		switch {
		case strings.Contains(in, "quiet ending") || strings.Contains(in, "conclusion"):
			out[i] = []float64{1, 0}
		case strings.Contains(in, "loud market") || strings.Contains(in, "market day"):
			out[i] = []float64{0, 1}
		default:
			out[i] = []float64{0.2, 0.2}
		}
	}
	return out, nil
}

func call(t *testing.T, args ...string) (string, string, int) {
	t.Helper()
	return callIn(t, strings.NewReader(""), args...)
}

func callIn(t *testing.T, in io.Reader, args ...string) (string, string, int) {
	t.Helper()
	var stdout, stderr strings.Builder
	code := run(args, &stdout, &stderr, in)
	return stdout.String(), stderr.String(), code
}

func TestAddListAndTopic(t *testing.T) {
	isolate(t)
	out, errOut, code := call(t, "add", "xyz", "a quiet ending", "the loud market")
	if code != 0 || strings.Contains(errOut, "without an embedding") {
		t.Fatalf("add %d %s\n%s", code, errOut, out)
	}
	if !strings.Contains(out, "1  xyz  a quiet ending") || !strings.Contains(out, "2  xyz  the loud market") {
		t.Fatalf("out %s", out)
	}
	out, _, code = call(t, "add", "book", "a quiet ending")
	if code != 0 || !strings.Contains(out, "3  book  a quiet ending") {
		t.Fatalf("second %d %s", code, out)
	}
	listed, _, code := call(t, "list", " xyz ")
	if code != 0 || strings.Contains(listed, "book") || !strings.Contains(listed, "quiet ending") {
		t.Fatalf("list %d %s", code, listed)
	}
}

func TestLongIdeaStoresNothing(t *testing.T) {
	isolate(t)
	called := false
	embedFn = func(inputs []string) ([][]float64, error) {
		called = true
		return scriptedEmbed(inputs)
	}
	long := strings.Repeat("a", 301)
	_, errOut, code := call(t, "add", "xyz", "kept", long)
	if code != 2 || called || !strings.Contains(errOut, "idea 2") || !strings.Contains(errOut, "300") {
		t.Fatalf("code %d called %v\n%s", code, called, errOut)
	}
	out, _, code := call(t, "list")
	if code != 0 || out != "no ideas\n" {
		t.Fatalf("list %d %q", code, out)
	}
}

func TestIdeaOf300Characters(t *testing.T) {
	isolate(t)
	text := strings.Repeat("a", 299) + "é"
	out, _, code := call(t, "add", "xyz", text)
	if code != 0 || !strings.Contains(out, text) {
		t.Fatalf("code %d %s", code, out)
	}
	_, errOut, code := call(t, "add", "xyz", text+"b")
	if code != 2 || !strings.Contains(errOut, "301") {
		t.Fatalf("code %d %s", code, errOut)
	}
}

func TestRejectsNewlineAndLongTopic(t *testing.T) {
	isolate(t)
	_, errOut, code := call(t, "add", "xyz", "one\ntwo")
	if code != 2 || !strings.Contains(errOut, "one line") {
		t.Fatalf("newline %d %s", code, errOut)
	}
	_, errOut, code = call(t, "add", strings.Repeat("t", 81), "idea")
	if code != 2 || !strings.Contains(errOut, "80") {
		t.Fatalf("topic %d %s", code, errOut)
	}
	out, _, code := call(t, "list")
	if code != 0 || out != "no ideas\n" {
		t.Fatalf("list %q", out)
	}
}

func TestStdinIdeasAndTerminal(t *testing.T) {
	isolate(t)
	out, _, code := callIn(t, strings.NewReader(" one \n\ntwo\n"), "add", "xyz")
	if code != 0 || !strings.Contains(out, "1  xyz  one") || !strings.Contains(out, "2  xyz  two") {
		t.Fatalf("stdin %d %s", code, out)
	}
	stdinTerminal = func(io.Reader) bool { return true }
	_, errOut, code := call(t, "add", "xyz")
	if code != 2 || !strings.Contains(errOut, "300") {
		t.Fatalf("terminal %d %s", code, errOut)
	}
}

func TestSearchRanksByMeaningInsideTopic(t *testing.T) {
	isolate(t)
	if _, _, code := call(t, "add", "xyz", "a quiet ending", "the loud market"); code != 0 {
		t.Fatal(code)
	}
	if _, _, code := call(t, "add", "book", "a quiet ending"); code != 0 {
		t.Fatal(code)
	}
	out, errOut, code := call(t, "search", "xyz", "conclusion")
	if code != 0 || errOut != "" {
		t.Fatalf("search %d %s\n%s", code, errOut, out)
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 2 || !strings.HasPrefix(lines[0], "1.000") || !strings.Contains(lines[0], "quiet ending") {
		t.Fatalf("rank %q", out)
	}
	if strings.Contains(out, "book") || strings.Contains(lines[0], "conclusion") {
		t.Fatalf("leaked %q", out)
	}
	if !strings.Contains(lines[1], "loud market") {
		t.Fatalf("second %q", out)
	}
}

func TestSearchDoesNotFallBackToText(t *testing.T) {
	isolate(t)
	embedFn = func([]string) ([][]float64, error) {
		return nil, os.ErrClosed
	}
	out, errOut, code := call(t, "add", "xyz", "the word bridge is here")
	if code != 0 || !strings.Contains(errOut, "stored without an embedding") {
		t.Fatalf("add %d %s\n%s", code, errOut, out)
	}
	called := false
	embedFn = func([]string) ([][]float64, error) {
		called = true
		return nil, os.ErrClosed
	}
	out, errOut, code = call(t, "search", "xyz", "bridge")
	if code != 0 || called || !strings.Contains(errOut, "no embedding") || strings.Contains(out, "bridge") {
		t.Fatalf("search %d called %v\n%s\n%s", code, called, errOut, out)
	}
	if out != "no ideas\n" {
		t.Fatalf("stdout %q", out)
	}
}

func TestSearchUnavailableDoesNotListText(t *testing.T) {
	isolate(t)
	if _, _, code := call(t, "add", "xyz", "a quiet ending"); code != 0 {
		t.Fatal(code)
	}
	embedFn = func([]string) ([][]float64, error) {
		return nil, os.ErrClosed
	}
	out, errOut, code := call(t, "search", "xyz", "conclusion")
	if code != 1 || out != "" || !strings.Contains(errOut, "embedding unavailable") || strings.Contains(errOut, "quiet") {
		t.Fatalf("code %d\n%s\n%s", code, errOut, out)
	}
}

func TestContainsIsLowercaseText(t *testing.T) {
	isolate(t)
	embedFn = func([]string) ([][]float64, error) {
		return nil, os.ErrClosed
	}
	if _, _, code := call(t, "add", "xyz", "A Quiet ending"); code != 0 {
		t.Fatal(code)
	}
	if _, _, code := call(t, "add", "book", "a quiet room"); code != 0 {
		t.Fatal(code)
	}
	out, errOut, code := call(t, "contains", "QUIET", "xyz")
	if code != 0 || errOut != "" || !strings.Contains(out, "Quiet ending") || strings.Contains(out, "book") {
		t.Fatalf("contains %d %s\n%s", code, errOut, out)
	}
	out, _, code = call(t, "contains", "conclusion")
	if code != 0 || out != "no ideas\n" {
		t.Fatalf("miss %d %q", code, out)
	}
}

func TestRemoveDoesNotReuseID(t *testing.T) {
	isolate(t)
	if _, _, code := call(t, "add", "xyz", "one", "two"); code != 0 {
		t.Fatal(code)
	}
	out, _, code := call(t, "remove", "2")
	if code != 0 || out != "removed 2\n" {
		t.Fatalf("remove %d %q", code, out)
	}
	_, errOut, code := call(t, "show", "2")
	if code != 1 || !strings.Contains(errOut, "no idea 2") {
		t.Fatalf("show %d %s", code, errOut)
	}
	out, _, code = call(t, "add", "xyz", "three")
	if code != 0 || !strings.Contains(out, "3  xyz  three") {
		t.Fatalf("next %d %s", code, out)
	}
}

func TestJSONOmitsVectors(t *testing.T) {
	isolate(t)
	out, _, code := call(t, "--json", "add", "xyz", "a quiet ending")
	if code != 0 || strings.Contains(out, `"embedding"`) || !strings.Contains(out, `"embedded":true`) {
		t.Fatalf("json %d %s", code, out)
	}
	var doc ideasDoc
	if err := json.Unmarshal([]byte(out), &doc); err != nil || len(doc.Ideas) != 1 || doc.Ideas[0].ID != "1" {
		t.Fatalf("decode %v %s", err, out)
	}
	body, ok, err := sqlxdb.Get(recordApp, recordKind, "1")
	if err != nil || !ok || !strings.Contains(string(body), `"embedding"`) {
		t.Fatalf("record ok=%v err=%v %s", ok, err, body)
	}
	path := filepath.Join(os.Getenv("SQLX_ROOT"), "data", "c3po.sqlite")
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}

func TestSearchJSONScore(t *testing.T) {
	isolate(t)
	if _, _, code := call(t, "add", "xyz", "a quiet ending", "the loud market"); code != 0 {
		t.Fatal(code)
	}
	out, _, code := call(t, "--json", "search", "xyz", "conclusion")
	if code != 0 || strings.Contains(out, `"embedding"`) {
		t.Fatalf("json %d %s", code, out)
	}
	var doc ideasDoc
	if err := json.Unmarshal([]byte(out), &doc); err != nil || len(doc.Ideas) != 2 {
		t.Fatalf("decode %v %s", err, out)
	}
	if doc.Ideas[0].Text != "a quiet ending" || doc.Ideas[0].Score == nil || *doc.Ideas[0].Score < 0.99 {
		t.Fatalf("first %#v", doc.Ideas[0])
	}
	if doc.Ideas[1].Score == nil || *doc.Ideas[0].Score <= *doc.Ideas[1].Score {
		t.Fatalf("order %#v", doc.Ideas)
	}
}

func TestTopicsAndShow(t *testing.T) {
	isolate(t)
	if _, _, code := call(t, "add", "xyz", "one", "two"); code != 0 {
		t.Fatal(code)
	}
	if _, _, code := call(t, "add", "book", "three"); code != 0 {
		t.Fatal(code)
	}
	out, _, code := call(t, "topics")
	if code != 0 || out != "1  book\n2  xyz\n" {
		t.Fatalf("topics %q", out)
	}
	out, _, code = call(t, "show", "1")
	if code != 0 || !strings.HasPrefix(out, "1  xyz  ") || !strings.HasSuffix(out, "one\n") {
		t.Fatalf("show %q", out)
	}
	out, _, code = call(t, "topics")
	if strings.Contains(out, "no topics") {
		t.Fatal(out)
	}
	empty, _, code := call(t, "list", "missing")
	if code != 0 || empty != "no ideas\n" {
		t.Fatalf("missing %q", empty)
	}
}

func TestSearchLimit(t *testing.T) {
	isolate(t)
	args := []string{"add", "xyz"}
	for i := 0; i < 25; i++ {
		args = append(args, "a quiet ending")
	}
	if _, _, code := call(t, args...); code != 0 {
		t.Fatal(code)
	}
	out, _, code := call(t, "search", "xyz", "conclusion")
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if code != 0 || len(lines) != searchLimit {
		t.Fatalf("code %d lines %d", code, len(lines))
	}
}

func TestHelpAndUnknown(t *testing.T) {
	out, _, code := call(t, "help")
	if code != 0 || !strings.Contains(out, "300") || !strings.Contains(out, "XAI_API_KEY") || !strings.Contains(out, "contains") {
		t.Fatalf("help %d %s", code, out)
	}
	same, _, code := call(t)
	if code != 0 || same != out {
		t.Fatal(code)
	}
	_, errOut, code := call(t, "nope")
	if code != 2 || !strings.Contains(errOut, "unknown command") || !strings.Contains(errOut, "contains") {
		t.Fatalf("unknown %d %s", code, errOut)
	}
	ver, _, code := call(t, "version")
	if code != 0 || !strings.HasPrefix(ver, "c3po ") {
		t.Fatalf("version %d %s", code, ver)
	}
}

func TestApostropheAndDuplicate(t *testing.T) {
	isolate(t)
	out, _, code := call(t, "add", "xyz", "a dog's idea", "a dog's idea")
	if code != 0 || strings.Count(out, "a dog's idea") != 2 {
		t.Fatalf("add %d %s", code, out)
	}
}

func TestCosine(t *testing.T) {
	if cosine([]float64{1, 0}, []float64{1, 0}) != 1 {
		t.Fatal("same")
	}
	if cosine([]float64{1, 0}, []float64{0, 1}) != 0 {
		t.Fatal("orthogonal")
	}
	if cosine([]float64{0, 0}, []float64{1, 0}) != 0 {
		t.Fatal("zero")
	}
}

func TestEmbedRemote(t *testing.T) {
	var gotAuth, gotModel string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		var body embedRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
			return
		}
		gotModel = body.Model
		if body.EncodingFormat != "float" || len(body.Input) != 2 {
			t.Errorf("body %#v", body)
		}
		_, _ = io.WriteString(w, `{"data":[{"index":1,"embedding":[0,1]},{"index":0,"embedding":[1,0]}]}`)
	}))
	t.Cleanup(srv.Close)
	t.Setenv("XAI_API_KEY", "test-key")
	prev := embedURL
	embedURL = srv.URL
	t.Cleanup(func() { embedURL = prev })
	vectors, err := embedRemote([]string{"passage: a", "passage: b"})
	if err != nil {
		t.Fatal(err)
	}
	if gotAuth != "Bearer test-key" || gotModel != embedModel {
		t.Fatalf("auth %s model %s", gotAuth, gotModel)
	}
	if len(vectors) != 2 || vectors[0][0] != 1 || vectors[1][1] != 1 {
		t.Fatalf("vectors %#v", vectors)
	}
}

func TestEmbedRemoteHidesTheKey(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = io.WriteString(w, "bad "+r.Header.Get("Authorization"))
	}))
	t.Cleanup(srv.Close)
	t.Setenv("XAI_API_KEY", "test-key")
	prev := embedURL
	embedURL = srv.URL
	t.Cleanup(func() { embedURL = prev })
	_, err := embedRemote([]string{"passage: a"})
	if err == nil || strings.Contains(err.Error(), "test-key") || strings.Contains(err.Error(), "Bearer") {
		t.Fatal(err)
	}
}

func TestEmbedRemoteNeedsKey(t *testing.T) {
	t.Setenv("XAI_API_KEY", "")
	_, err := embedRemote([]string{"passage: a"})
	if err == nil || strings.Contains(err.Error(), "http") {
		t.Fatal(err)
	}
}
