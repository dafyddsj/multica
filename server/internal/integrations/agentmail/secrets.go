package agentmail

import (
	"encoding/base64"
	"errors"
	"strings"

	"github.com/multica-ai/multica/server/internal/util/secretbox"
)

const (
	orgSecretPrefix   = "agentmail:org:v1:"
	inboxSecretPrefix = "agentmail:inbox:v1:"
)

func sealSecret(box *secretbox.Box, prefix, plaintext string) (string, error) {
	sealed, err := box.Seal([]byte(prefix + plaintext))
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(sealed), nil
}

func openSecret(box *secretbox.Box, prefix, encoded string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}
	plain, err := box.Open(raw)
	if err != nil {
		return "", err
	}
	s := string(plain)
	if !strings.HasPrefix(s, prefix) {
		return "", errors.New("agentmail: secret purpose mismatch")
	}
	return strings.TrimPrefix(s, prefix), nil
}

func sealOrgKey(box *secretbox.Box, plaintext string) (string, error) {
	return sealSecret(box, orgSecretPrefix, plaintext)
}

func openOrgKey(box *secretbox.Box, encoded string) (string, error) {
	return openSecret(box, orgSecretPrefix, encoded)
}

func sealInboxKey(box *secretbox.Box, plaintext string) (string, error) {
	return sealSecret(box, inboxSecretPrefix, plaintext)
}

func openInboxKey(box *secretbox.Box, encoded string) (string, error) {
	return openSecret(box, inboxSecretPrefix, encoded)
}
