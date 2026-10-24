package gmail

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDecisionsMapsAnswers(t *testing.T) {
	var bodies [][]byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("auth %q", r.Header.Get("Authorization"))
		}
		body, _ := io.ReadAll(r.Body)
		bodies = append(bodies, body)
		var req struct {
			Model     string `json:"model"`
			Questions []struct {
				Name string `json:"name"`
				Type string `json:"type"`
			} `json:"questions"`
		}
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatal(err)
		}
		if req.Model != decisionsModel {
			t.Errorf("model %s", req.Model)
		}
		n := len(req.Questions) / 2
		promo := strings.Contains(string(body), "promo@bulk.test")
		answers := make([]map[string]any, 0, len(req.Questions))
		for i := 0; i < n; i++ {
			choice := "keep"
			conf := 0.0
			prob := 0.0
			if i == 0 && promo {
				choice, conf, prob = "trash", 1, 1
			}
			answers = append(answers,
				map[string]any{"type": "choice", "name": req.Questions[i*2].Name, "choice": choice, "confidence": conf},
				map[string]any{"type": "predicate", "name": req.Questions[i*2+1].Name, "probability": prob},
			)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"answers": answers})
	}))
	defer srv.Close()

	mail := make([]mailBrief, 151)
	for i := range mail {
		mail[i] = mailBrief{Preset: "1", ID: itoa(i), From: "a@b.test", Subject: "s", Snippet: "n"}
	}
	mail[0].ID = "first"
	mail[0].From = "promo@bulk.test"
	got, err := classifyMail(srv.Client(), srv.URL, "test-key", mail)
	if err != nil {
		t.Fatal(err)
	}
	if len(bodies) != 2 {
		t.Fatalf("requests %d", len(bodies))
	}
	if !strings.Contains(string(bodies[0]), `"name":"a0"`) || !strings.Contains(string(bodies[0]), `"name":"d0"`) {
		t.Fatalf("questions %s", bodies[0])
	}
	if !strings.Contains(string(bodies[0]), "gpt-6-luna") {
		t.Fatalf("model missing %s", bodies[0])
	}
	first := got[mailKey("1", "first")]
	if first.Action != "trash" || first.Confidence != 1 || first.Disposable != 1 {
		t.Fatalf("first %+v", first)
	}
	last := got[mailKey("1", mail[150].ID)]
	if last.Action != "keep" || last.Disposable != 0 {
		t.Fatalf("last %+v", last)
	}
}

func TestDecisionsRejectsABadStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "secret-body", http.StatusBadGateway)
	}))
	defer srv.Close()
	_, err := classifyMail(srv.Client(), srv.URL, "test-key", []mailBrief{{Preset: "1", ID: "m"}})
	if err == nil || err.Error() != "decisions api returned 502" || strings.Contains(err.Error(), "secret-body") {
		t.Fatalf("err %v", err)
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [16]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
