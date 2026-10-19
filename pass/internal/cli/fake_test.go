package cli

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
)

const fakeToken = "sa-token-value"

type opCall struct {
	args  string
	stdin string
}

// fakeOp stands in for the op binary. It rejects any other token, as
// the old fake script did.
type fakeOp struct {
	list    string
	get     string
	reads   map[string]string
	missing bool
	calls   []opCall
}

func (f *fakeOp) LookPath() error {
	if f.missing {
		return errors.New("op not found")
	}
	return nil
}

func (f *fakeOp) Run(env []string, stdin []byte, stderr io.Writer, args ...string) ([]byte, error) {
	if tokenOf(env) != fakeToken {
		_, _ = fmt.Fprintln(stderr, "token rejected")
		return nil, errors.New("exit status 1")
	}
	joined := strings.Join(args, " ")
	f.calls = append(f.calls, opCall{args: joined, stdin: string(stdin)})
	switch {
	case strings.HasPrefix(joined, "item list"):
		return []byte(or(f.list, "[]")), nil
	case strings.HasPrefix(joined, "item get"):
		return []byte(or(f.get, "{}")), nil
	case strings.HasPrefix(joined, "item create"), strings.HasPrefix(joined, "item edit"), strings.HasPrefix(joined, "item delete"):
		return nil, nil
	case args[0] == "read":
		return f.read(args[len(args)-1], stderr)
	}
	_, _ = fmt.Fprintf(stderr, "unexpected op invocation: %s\n", joined)
	return nil, errors.New("exit status 1")
}

func (f *fakeOp) read(ref string, stderr io.Writer) ([]byte, error) {
	for suffix, secret := range f.reads {
		if strings.HasSuffix(ref, suffix) {
			return []byte(secret), nil
		}
	}
	_, _ = fmt.Fprintln(stderr, "unexpected op read")
	return nil, errors.New("exit status 1")
}

// call returns the last call whose args start with prefix.
func (f *fakeOp) call(prefix string) (opCall, bool) {
	for i := len(f.calls) - 1; i >= 0; i-- {
		if strings.HasPrefix(f.calls[i].args, prefix) {
			return f.calls[i], true
		}
	}
	return opCall{}, false
}

func tokenOf(env []string) string {
	value := ""
	for _, entry := range env {
		if rest, ok := strings.CutPrefix(entry, "OP_SERVICE_ACCOUNT_TOKEN="); ok {
			value = rest
		}
	}
	return value
}

type fakeClip struct {
	text   string
	copied bool
}

func (c *fakeClip) Copy(text string, stderr io.Writer) error {
	c.text = text
	c.copied = true
	return nil
}

// isolate keeps the real token file and op out of every test.
func isolate(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("PATH", t.TempDir())
	t.Setenv("OP_SERVICE_ACCOUNT_TOKEN_FILE", "")
	t.Setenv("OP_SERVICE_ACCOUNT_TOKEN", fakeToken)
}

func newRunner(t *testing.T, fake *fakeOp) (Runner, *bytes.Buffer, *bytes.Buffer, *fakeClip) {
	t.Helper()
	isolate(t)
	var stdout, stderr bytes.Buffer
	clipboard := &fakeClip{}
	devnull, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = devnull.Close() })
	runner := Runner{Out: &stdout, Err: &stderr, In: devnull, Op: fake, Clip: clipboard, Help: "help text\n", Version: "9.9.9"}
	return runner, &stdout, &stderr, clipboard
}

// runPiped runs with stdout as a pipe, as in eval "$(...)".
func runPiped(t *testing.T, runner Runner, args ...string) (string, int) {
	t.Helper()
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	runner.Out = writer
	rc := runner.Run(args)
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	out, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	_ = reader.Close()
	return string(out), rc
}

func assertNoSecret(t *testing.T, stdout, stderr string, secrets ...string) {
	t.Helper()
	for _, secret := range append(secrets, fakeToken) {
		if secret != "" && strings.Contains(stdout+stderr, secret) {
			t.Fatalf("output contained a secret %q\nstdout: %s\nstderr: %s", secret, stdout, stderr)
		}
	}
}
