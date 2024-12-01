package user

import "errors"

var ErrLoginNotUnique = errors.New("user with this login already exists")
var ErrUserNotFound = errors.New("user not found")

type Repository interface {
	Add(login string, passwordHash string) error
	GetPasswordHash(login string) (string, error)
}
