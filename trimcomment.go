package jsonpreprocess

import (
	"bytes"
	"errors"
	"io"
	"strings"
)

func WriteCommentTrimmedTo(writer io.Writer, input io.Reader) error {
	l := lex(input)
	for {
		switch item := l.nextItem(); item.typ {
		case itemEOF:
			return nil
		case itemError:
			return errors.New(item.val)
		case itemBlockComment:
			break
		case itemLineComment:
			if strings.HasSuffix(item.val, "\n") {
				item.val = "\n"
			} else {
				item.val = ""
			}
			fallthrough
		default:
			_, err := writer.Write([]byte(item.val))
			if err != nil {
				return err
			}
		}
	}
}

func TrimComment(input string) (string, error) {
	var out bytes.Buffer
	reader := bytes.NewBufferString(input)
	err := WriteCommentTrimmedTo(&out, reader)
	if err != nil {
		return "", err
	}
	return out.String(), nil
}
