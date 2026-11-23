package forwarder

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
)

func ExtractBranchName(ref string) (string, error) {
	branch, ok := strings.CutPrefix(ref, "refs/heads/")
	if !ok || branch == "" {
		return "", errors.New("ref is not a branch")
	}
	return branch, nil
}

func VerifySignature(payloadBody []byte, secretToken string, signatureHeader string) error {
	if signatureHeader == "" {
		return errors.New("x-hub-signature-256 header is missing")
	}

	mac := hmac.New(sha256.New, []byte(secretToken))
	_, err := mac.Write(payloadBody)
	if err != nil {
		return err
	}

	expectedSignature := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(signatureHeader), []byte(expectedSignature)) {
		return errors.New("request signatures didn't match")
	}
	return nil
}
