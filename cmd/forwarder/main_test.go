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
	previousRegex := whitelistedBranchRegex
	t.Cleanup(func() {
		webhooks, whitelistedBranches, commitPushOnly = previousWebhooks, previousBranches, previousMode
		whitelistedBranchRegex = previousRegex
	})
	webhooks = []string{destination.URL}
	whitelistedBranches = []string{"main"}
	whitelistedBranchRegex = regexp.MustCompile(`^release/.+$`)

	push := func(ref, fields string) string {
		return fmt.Sprintf(`{"ref":%q,"commits":[{"id":"abc"}]%s}`, ref, fields)
	}
	tests := []struct {
		name    string
		event   string
		body    string
		enabled bool
		forward bool
	}{
		{"regular push", "push", push("refs/heads/main", ""), true, true},
		{"regex match", "push", push("refs/heads/release/v1", ""), true, true},
		{"regex nonmatch", "push", push("refs/heads/topic/release/v1", ""), true, false},
		{"force push on regex match", "push", push("refs/heads/release/v1", `,"forced":true`), true, false},
		{"tag with matching embedded branch name", "push", push("refs/tags/refs/heads/release/v1", ""), false, false},
		{"force push", "push", push("refs/heads/main", `,"forced":true`), true, false},
		{"branch creation", "push", push("refs/heads/main", `,"created":true`), true, false},
		{"branch deletion", "push", `{"ref":"refs/heads/main","deleted":true,"commits":[]}`, true, false},
		{"tag push", "push", push("refs/tags/main", ""), true, false},
		{"tag with branch prefix inside its name", "push", push("refs/tags/refs/heads/main", ""), true, false},
		{"other branch", "push", push("refs/heads/topic", ""), true, false},
		{"empty push", "push", `{"ref":"refs/heads/main","commits":[]}`, true, false},
		{"non-push with matching ref", "workflow_run", push("refs/heads/main", ""), true, false},
		{"ping", "ping", `{"zen":"Keep it logically awesome."}`, true, false},
		{"force push with commit-only filtering disabled", "push", push("refs/heads/main", `,"forced":true`), false, true},
		{"non-push with commit-only filtering disabled", "workflow_run", `{"ref":"refs/heads/main","commits":42}`, false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			commitPushOnly = tt.enabled
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
