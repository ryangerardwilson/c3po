package tui

import (
	"context"
	"strings"
	"testing"
)

func TestNebulaItemsAndCopyDoNotCarryASecret(t *testing.T) {
	prev := runCrypt
	t.Cleanup(func() { runCrypt = prev })
	var got [][]string
	runCrypt = func(_ context.Context, args ...string) ([]byte, []byte, error) {
		got = append(got, append([]string(nil), args...))
		switch args[0] {
		case "items":
			return []byte("{\"id\":\"abc\",\"vault_id\":\"v1\",\"title\":\"GH_EXAMPLE\",\"category\":\"LOGIN\",\"username\":\"ada\"}\n"), nil, nil
		case "vaults":
			return []byte("{\"id\":\"v1\",\"name\":\"Root\"}\n"), nil, nil
		default:
			return nil, nil, nil
		}
	}
	src := Nebula{}
	ctx := context.Background()
	vaults, err := src.Vaults(ctx)
	if err != nil || len(vaults) != 1 || vaults[0].Name != "Root" {
		t.Fatalf("%v %v", vaults, err)
	}
	items, err := src.Items(ctx, "Root")
	if err != nil || len(items) != 1 || items[0].Username != "ada" || items[0].ID != "abc" {
		t.Fatalf("%v %v", items, err)
	}
	if err := src.Copy(ctx, "Root", "abc"); err != nil {
		t.Fatal(err)
	}
	if err := src.Delete(ctx, "Root", "abc"); err != nil {
		t.Fatal(err)
	}
	want := []string{"vaults", "items Root", "copy Root abc", "delete Root item abc"}
	if len(got) != len(want) {
		t.Fatalf("%v", got)
	}
	for i, args := range got {
		if strings.Join(args, " ") != want[i] {
			t.Fatalf("%v", got)
		}
	}
}
