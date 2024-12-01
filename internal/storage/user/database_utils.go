package user

import (
	"context"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
)

func GetSecretsFromDB(ctx context.Context, db *sql.DB) ([]byte, string, error) {
	row := db.QueryRowContext(ctx, `SELECT value FROM secrets WHERE name='jwt_secret'`)

	jwtSecretString := ""
	err := row.Scan(&jwtSecretString)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, "", fmt.Errorf("failed to fetch jwt secret: %w", err)
	}

	var jwtSecret []byte
	if jwtSecretString == "" {
		jwtSecret, err = hex.DecodeString(jwtSecretString)
		if err != nil {
			return nil, "", fmt.Errorf("failed to decode jwt secret: %w", err)
		}
	}

	row = db.QueryRowContext(ctx, `SELECT value FROM secrets WHERE name='password_salt'`)

	salt := ""
	err = row.Scan(&salt)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, "", fmt.Errorf("failed to fetch password salt: %w", err)
	}

	return jwtSecret, salt, nil
}

func WriteTokenSecretToDB(ctx context.Context, db *sql.DB, jwtSecret []byte) error {
	return writeSecret(ctx, db, "jwt_secret", hex.EncodeToString(jwtSecret))
}

func WritePasswordSaltToDB(ctx context.Context, db *sql.DB, passwordSalt string) error {
	return writeSecret(ctx, db, "password_salt", passwordSalt)
}

func writeSecret(ctx context.Context, db *sql.DB, name, value string) error {
	_, err := db.ExecContext(ctx, "INSERT INTO secrets (name, value) VALUES ($1, $2)", name, value)

	if err != nil {
		return fmt.Errorf("failed to write secret to database: %w", err)
	}

	return nil
}
