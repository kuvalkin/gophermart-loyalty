package user

import (
	"crypto/rand"
	"encoding/hex"
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

	return hex.EncodeToString(randomBytes), nil
}
