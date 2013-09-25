package main

import (
	"os"

	"github.com/hnakamur/jsonutil"
)

func main() {
	err := jsonutil.WriteCommentTrimmedTo(os.Stdout, os.Stdin)
	if err != nil {
		panic(err)
	}
}
