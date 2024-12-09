package database

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func InitDB(ctx context.Context, dsn string) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("could not connect to database: %w", err)
	}

	err = db.PingContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not ping database: %w", err)
	}

	return db, nil
}

func Migrate(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS users (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	login VARCHAR(250) UNIQUE NOT NULL,
	password_hash TEXT NOT NULL,
	created_at TIMESTAMP NOT NULL DEFAULT now()
)`)

	if err != nil {
		return fmt.Errorf("could not create users table: %w", err)
	}

	_, err = db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS balances (
	user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE RESTRICT,
	current INT NOT NULL DEFAULT 0,
	withdrawn INT NOT NULL DEFAULT 0
)`)

	if err != nil {
		return fmt.Errorf("could not create balances table: %w", err)
	}

	_, err = db.ExecContext(ctx, `DO $$ BEGIN
	CREATE TYPE order_status AS ENUM (
		'NEW',
		'PROCESSING',
		'INVALID',
		'PROCESSED'
	);
EXCEPTION
	WHEN duplicate_object THEN null;
END $$;`)

	if err != nil {
		return fmt.Errorf("could not create order_status enum type: %w", err)
	}

	_, err = db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS orders (
	number TEXT PRIMARY KEY,
	user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
	status order_status NOT NULL DEFAULT 'NEW',
	uploaded_at TIMESTAMP NOT NULL DEFAULT now(),
	updated_at TIMESTAMP NOT NULL DEFAULT now(),
	accrual INT DEFAULT NULL
)`)

	if err != nil {
		return fmt.Errorf("could not create orders table: %w", err)
	}

	_, err = db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS orders_history (
	order_number TEXT PRIMARY KEY REFERENCES orders(number) ON DELETE RESTRICT,
	old_status order_status DEFAULT NULL,
	new_status order_status DEFAULT NULL,
	changed_at TIMESTAMP NOT NULL DEFAULT now()
)`)

	if err != nil {
		return fmt.Errorf("could not create orders history table: %w", err)
	}

	_, err = db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS withdrawals (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
	order_number TEXT NOT NULL REFERENCES orders(number) ON DELETE RESTRICT,
	sum INT NOT NULL DEFAULT 0,
	processed_at TIMESTAMP NOT NULL DEFAULT now()
)`)

	if err != nil {
		return fmt.Errorf("could not create withdrawals table: %w", err)
	}

	_, err = db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS secrets (
	name TEXT NOT NULL PRIMARY KEY,
	value TEXT NOT NULL
)`)

	if err != nil {
		return fmt.Errorf("could not create secrets table: %w", err)
	}

	return nil
}
