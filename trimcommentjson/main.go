package main

import (
	"os"

	"github.com/hnakamur/jsonpreprocess"
)

func main() {
	err := jsonpreprocess.WriteCommentTrimmedTo(os.Stdout, os.Stdin)
	if err != nil {
		panic(err)
	}
}
