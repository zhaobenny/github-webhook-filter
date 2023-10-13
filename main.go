package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

type Ref struct {
	RefText string `json:"ref,omitempty"`
}

var whitelistedBranches = []string{}
var webhooks = []string{}
var webhookSecret = ""

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
		err = VerifySignature(body, webhookSecret, r.Header.Get("X-Hub-Signature-256"))
	}
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Unable to verify signature", http.StatusUnauthorized)
		return
	}

	var ref Ref

	err = json.Unmarshal(body, &ref) // finding branch name
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Unable to unmarshal JSON", http.StatusBadRequest)
		return
	}

	branchName, err := ExtractBranchName(ref.RefText)
	isWhitelisted := false

	if err == nil {
		for _, allowedBranch := range whitelistedBranches {
			if branchName == allowedBranch {
				isWhitelisted = true
				break
			}
		}
	}

	if err != nil || isWhitelisted {
		for _, webhook := range webhooks {
			var resp *http.Response

			requestbody := bytes.NewReader(body)

			if err != nil {
				fmt.Println(err)
				continue
			}

			req, err := http.NewRequest("POST", webhook, requestbody)
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

			resp, err = http.DefaultClient.Do(req)
			if err != nil {
				fmt.Println(err)
				continue
			}

			resp.Body.Close()
		}
	}
	w.WriteHeader(http.StatusOK)
}

func main() {
	whitelistEnv := os.Getenv("WHITELISTED_BRANCHES")
	if whitelistEnv != "" {
		whitelistedBranches = strings.Split(whitelistEnv, ",")
	}

	webhooksEnv := os.Getenv("WEBHOOKS")
	if webhooksEnv != "" {
		webhooks = strings.Split(webhooksEnv, ",")
	}

	webhookSecret = os.Getenv("WEBHOOK_SECRET")

	http.HandleFunc("/forward", webhookHandler)

	fmt.Println("Listening for webhook events on :8080/forward")
	http.ListenAndServe(":8080", nil)
}
