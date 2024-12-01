package user

import (
	"database/sql"
)

type dbRepo struct {
	db *sql.DB
}

func NewDatabaseRepository(db *sql.DB) Repository {
	return &dbRepo{db: db}
}

func (d *dbRepo) Add(login string, passwordHash string) error {
	//TODO implement me
	panic("implement me")
}

func (d *dbRepo) GetPasswordHash(login string) (string, error) {
	//TODO implement me
	panic("implement me")
}
