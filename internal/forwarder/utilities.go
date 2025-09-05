package forwarder

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

func ExtractBranchName(ref string) (string, error) {

	if ref == "" {
		return "", errors.New("empty value for 'refs'")
	}

	parts := strings.Split(ref, "refs/heads/")
	if len(parts) != 2 {
		fmt.Println("The value associated with 'refs' does not contain 'refs/heads/ substring!'")
		return "", errors.New("bad value for 'refs")
	}

	branchName := parts[1]
	return branchName, nil

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
