package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"slices"
	"strings"

	"gh-webhook-forwarder/internal/forwarder"
)

var whitelistedBranches = []string{}
var whitelistedBranchRegex *regexp.Regexp
var webhooks = []string{}
var webhookSecret = ""
var commitPushOnly bool

func branchAllowed(branch string) bool {
	if branch == "" {
		return false
	}
	return slices.Contains(whitelistedBranches, branch) ||
		(whitelistedBranchRegex != nil && whitelistedBranchRegex.MatchString(branch))
}

func eventBranch(event string, body []byte) (string, error) {
	if commitPushOnly {
		if event != "push" {
			return "", nil
		}
		var push struct {
			Ref     string            `json:"ref"`
			Created bool              `json:"created"`
			Deleted bool              `json:"deleted"`
			Forced  bool              `json:"forced"`
			Commits []json.RawMessage `json:"commits"`
		}
		if err := json.Unmarshal(body, &push); err != nil {
			return "", err
		}
		if push.Created || push.Deleted || push.Forced || len(push.Commits) == 0 {
			return "", nil
		}
		branch, _ := forwarder.ExtractBranchName(push.Ref)
		return branch, nil
	}

	var payload struct {
		Ref string `json:"ref"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", err
	}
	branch, err := forwarder.ExtractBranchName(payload.Ref)
	if err != nil {
		fmt.Println(err)
	}
	return branch, nil
}

func webhookHandler(w http.ResponseWriter, r *http.Request) {

	ghHeader := r.Header.Get("X-GitHub-Event") // better then nothing, just use secret
	if ghHeader == "" {
		http.Error(w, "404 page not found", http.StatusNotFound)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Unable to read request body", http.StatusBadRequest)
		return
	}

	if webhookSecret != "" {
		err = forwarder.VerifySignature(body, webhookSecret, r.Header.Get("X-Hub-Signature-256"))
	}
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Unable to verify signature", http.StatusUnauthorized)
		return
	}

	branch, err := eventBranch(ghHeader, body)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Unable to unmarshal JSON", http.StatusBadRequest)
		return
	}

	if !branchAllowed(branch) {
		w.WriteHeader(http.StatusOK)
		return
	}

	for _, webhook := range webhooks {
		req, err := http.NewRequest("POST", webhook, bytes.NewReader(body))
		if err != nil {
			fmt.Println(err)
			continue
		}

		req.Header.Set("Content-Type", "application/json")
		for key, value := range r.Header {
			if strings.HasPrefix(key, "X-") {
				req.Header[key] = value
			}
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			fmt.Println(err)
			continue
		}

		resp.Body.Close()
	}
	w.WriteHeader(http.StatusOK)
}

func main() {
	whitelistEnv := os.Getenv("WHITELISTED_BRANCHES")
	if whitelistEnv != "" {
		whitelistedBranches = strings.Split(whitelistEnv, ",")
	}
	if pattern := os.Getenv("WHITELISTED_BRANCH_REGEX"); pattern != "" {
		compiled, err := regexp.Compile(pattern)
		if err != nil {
			log.Fatalf("invalid WHITELISTED_BRANCH_REGEX: %v", err)
		}
		whitelistedBranchRegex = compiled
	}

	webhooksEnv := os.Getenv("WEBHOOKS")
	if webhooksEnv != "" {
		webhooks = strings.Split(webhooksEnv, ",")
	}

	webhookSecret = os.Getenv("WEBHOOK_SECRET")
	commitPushOnly = os.Getenv("COMMIT_PUSH_ONLY") != "false"

	http.HandleFunc("/forward", webhookHandler)

	fmt.Println("Listening for webhook events on :8080/forward")
	http.ListenAndServe(":8080", nil)
}
