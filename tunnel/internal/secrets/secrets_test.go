package secrets

import "testing"

func TestParseReadsLabels(t *testing.T) {
	tunnel := []byte(`{"fields":[
		{"label":"credential","value":"{\"AccountTag\":\"a\"}"},
		{"label":"tunnel_id","value":"1bdc9e04-7e31-4589-baa9-f6caf6ae9a74"},
		{"label":"hostname","value":"ryangerardwilson.com"}
	]}`)
	api := []byte(`{"fields":[{"label":"credential","value":"token-value"}]}`)
	got, err := Parse(tunnel, api)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "1bdc9e04-7e31-4589-baa9-f6caf6ae9a74" || got.Zone != "ryangerardwilson.com" {
		t.Fatalf("%+v", got)
	}
	if string(got.Credential) != `{"AccountTag":"a"}` || got.APIToken != "token-value" {
		t.Fatal("credential fields were not kept")
	}
}

func TestParseRejectsMissingID(t *testing.T) {
	tunnel := []byte(`{"fields":[{"label":"credential","value":"{}"}]}`)
	api := []byte(`{"fields":[{"label":"credential","value":"t"}]}`)
	if _, err := Parse(tunnel, api); err == nil {
		t.Fatal("missing tunnel id was accepted")
	}
}
