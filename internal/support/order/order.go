package order

import (
	"errors"

	"github.com/ShiraazMoollatjie/goluhn"
)

func ValidateNumber(number string) error {
	if number == "" {
		// goluhn doesnt return error on empty string
		return errors.New("empty number")
	}

	return goluhn.Validate(number)
}
