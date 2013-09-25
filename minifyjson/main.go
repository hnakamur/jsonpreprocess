package main

import (
	"os"

	"github.com/hnakamur/jsonpreprocess"
)

func main() {
	err := jsonpreprocess.WriteMinifiedTo(os.Stdout, os.Stdin)
	if err != nil {
		panic(err)
	}
}
