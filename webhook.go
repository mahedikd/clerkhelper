package clerkhelper

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strings"
)

// VerifySvixSignature verifies a Clerk webhook request using the Svix
// signature scheme. The secret must be in the form "<prefix>_<base64key>"
// (e.g. "whsec_<key>"). Signed content is "<svixID>.<svixTimestamp>.<payload>"
// HMAC-SHA256'd with the decoded key; any "v1,<sig>" entry in the Svix
// signature header that matches is accepted.
func VerifySvixSignature(payload []byte, svixID, svixTimestamp, svixSignature, secret string) error {
	if svixID == "" || svixTimestamp == "" || svixSignature == "" {
		return fmt.Errorf("missing svix headers")
	}

	signedContent := svixID + "." + svixTimestamp + "." + string(payload)

	parts := strings.SplitN(secret, "_", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid webhook secret format")
	}
	keyBytes, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return fmt.Errorf("invalid webhook secret base64: %w", err)
	}

	mac := hmac.New(sha256.New, keyBytes)
	mac.Write([]byte(signedContent))
	expected := mac.Sum(nil)

	for _, sigPart := range strings.Fields(svixSignature) {
		if !strings.HasPrefix(sigPart, "v1,") {
			continue
		}
		sigBytes, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(sigPart, "v1,"))
		if err != nil {
			continue
		}
		if hmac.Equal(sigBytes, expected) {
			return nil
		}
	}
	return fmt.Errorf("invalid webhook signature")
}
