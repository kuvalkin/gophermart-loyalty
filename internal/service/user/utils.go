package user

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

func GenerateTokenSecret() ([]byte, error) {
	buf := make([]byte, 32)

	_, err := rand.Read(buf)
	if err != nil {
		return nil, fmt.Errorf("failed to read random bytes: %w", err)
	}

	return buf, nil
}

func GeneratePasswordSalt() (string, error) {
	randomBytes, err := GenerateTokenSecret()
	if err != nil {
		return "", fmt.Errorf("failed to generate random string: %w", err)
	}

	buf := bytes.NewBuffer(nil)
	encoder := base64.NewEncoder(base64.StdEncoding, buf)
	_, err = encoder.Write(randomBytes)
	if err != nil {
		return "", fmt.Errorf("failed to encode random bytes: %w", err)
	}

	return buf.String(), nil
}
