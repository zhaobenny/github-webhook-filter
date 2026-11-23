package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
)

func TestWebhookHandlerFiltering(t *testing.T) {
	type delivery struct {
		body  string
		event string
	}
	deliveries := make(chan delivery, 1)
	destination := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
		}
		deliveries <- delivery{string(body), r.Header.Get("X-GitHub-Event")}
		w.WriteHeader(http.StatusOK)
	}))
	defer destination.Close()

	previousWebhooks, previousBranches, previousMode := webhooks, whitelistedBranches, commitPushOnly
	previousRegex := webhookPayloadRegex
	t.Cleanup(func() {
		webhooks, whitelistedBranches, commitPushOnly = previousWebhooks, previousBranches, previousMode
		webhookPayloadRegex = previousRegex
	})
	webhooks = []string{destination.URL}
	whitelistedBranches = []string{"main"}

	push := func(ref, fields string) string {
		return fmt.Sprintf(`{"ref":%q,"commits":[{"id":"abc"}]%s}`, ref, fields)
	}
	payload := func(branch, message, fields string) string {
		return fmt.Sprintf(`{"ref":%q,"commits":[{"id":"abc","message": %q}]%s}`, "refs/heads/"+branch, message, fields)
	}
	deployPattern := regexp.MustCompile(`"message"[[:space:]]*:[[:space:]]*"Deploy:`)
	tests := []struct {
		name    string
		event   string
		body    string
		enabled bool
		forward bool
		pattern *regexp.Regexp
	}{
		{"regular push", "push", push("refs/heads/main", ""), true, true, nil},
		{"payload match", "push", payload("main", "Deploy: production", ""), true, true, deployPattern},
		{"payload nonmatch", "push", payload("main", "Update README", ""), true, false, deployPattern},
		{"payload match on another branch", "push", payload("topic", "Deploy: production", ""), true, false, deployPattern},
		{"payload match on force push", "push", payload("main", "Deploy: production", `,"forced":true`), true, false, deployPattern},
		{"force push", "push", push("refs/heads/main", `,"forced":true`), true, false, nil},
		{"branch creation", "push", push("refs/heads/main", `,"created":true`), true, false, nil},
		{"branch deletion", "push", `{"ref":"refs/heads/main","deleted":true,"commits":[]}`, true, false, nil},
		{"tag push", "push", push("refs/tags/main", ""), true, false, nil},
		{"tag with branch prefix inside its name", "push", push("refs/tags/refs/heads/main", ""), true, false, nil},
		{"other branch", "push", push("refs/heads/topic", ""), true, false, nil},
		{"empty push", "push", `{"ref":"refs/heads/main","commits":[]}`, true, false, nil},
		{"non-push with matching ref", "workflow_run", push("refs/heads/main", ""), true, false, nil},
		{"ping", "ping", `{"zen":"Keep it logically awesome."}`, true, false, nil},
		{"force push with commit-only filtering disabled", "push", push("refs/heads/main", `,"forced":true`), false, true, nil},
		{"non-push with commit-only filtering disabled", "workflow_run", `{"ref":"refs/heads/main","commits":42}`, false, true, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			commitPushOnly = tt.enabled
			webhookPayloadRegex = tt.pattern
			request := httptest.NewRequest(http.MethodPost, "/forward", strings.NewReader(tt.body))
			request.Header.Set("X-GitHub-Event", tt.event)
			response := httptest.NewRecorder()

			webhookHandler(response, request)

			if response.Code != http.StatusOK {
				t.Errorf("status = %d, want %d", response.Code, http.StatusOK)
			}
			select {
			case got := <-deliveries:
				if !tt.forward {
					t.Fatal("unexpected delivery")
				}
				if want := (delivery{tt.body, tt.event}); got != want {
					t.Errorf("delivery = %#v, want %#v", got, want)
				}
			default:
				if tt.forward {
					t.Fatal("missing delivery")
				}
			}
		})
	}
}
