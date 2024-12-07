package test

import "github.com/ShiraazMoollatjie/goluhn"

func NewOrderNumber() string {
	return goluhn.Generate(10)
}
