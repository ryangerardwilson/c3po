package sqlxdb

import "testing"

func TestOnceRunsOneTime(t *testing.T) {
	t.Setenv("SQLX_ROOT", t.TempDir())
	if err := Put("app", "kind", "k", []byte("a")); err != nil {
		t.Fatal(err)
	}
	bump := "UPDATE record SET body = body || 'b' WHERE app = 'app'"
	for i := 0; i < 2; i++ {
		if err := Once("test-bump-1", bump); err != nil {
			t.Fatal(err)
		}
	}
	got, _, err := Get("app", "kind", "k")
	if err != nil || string(got) != "ab" {
		t.Fatalf("body = %q, %v", got, err)
	}
}

func TestOnceFailureCommitsNothing(t *testing.T) {
	t.Setenv("SQLX_ROOT", t.TempDir())
	if err := Put("app", "kind", "k", []byte("a")); err != nil {
		t.Fatal(err)
	}
	bad := "UPDATE record SET body = 'x' WHERE app = 'app';\nINSERT INTO missing VALUES (1)"
	if err := Once("test-bad-1", bad); err == nil {
		t.Fatal("a failing migration returned no error")
	}
	got, _, err := Get("app", "kind", "k")
	if err != nil || string(got) != "a" {
		t.Fatalf("after failure body = %q, %v", got, err)
	}
	if err := Once("test-bad-1", "UPDATE record SET body = 'y' WHERE app = 'app'"); err != nil {
		t.Fatal(err)
	}
	got, _, err = Get("app", "kind", "k")
	if err != nil || string(got) != "y" {
		t.Fatalf("the failed id was recorded: body = %q, %v", got, err)
	}
}

func TestOnceForFilePathRunsNothing(t *testing.T) {
	t.Setenv("SQLX_ROOT", "")
	t.Setenv("C3PO_SQLX", "off")
	if err := OnceFor(t.TempDir()+"/state.json", "test-file-1", "SELECT missing FROM nowhere"); err != nil {
		t.Fatal(err)
	}
}
