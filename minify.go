package jsonutil

import (
	"bytes"
	"errors"
	"io"
)

func WriteMinifiedTo(writer io.Writer, input string) error {
	l := lex(input)
	for {
		switch item := l.nextItem(); item.typ {
		case itemEOF:
			return nil
		case itemError:
			return errors.New(item.val)
		case itemWhitespace:
			break
		case itemBlockComment:
			break
		case itemLineComment:
			break
		default:
			_, err := writer.Write([]byte(item.val))
			if err != nil {
				return err
			}
		}
	}
}

func Minify(input string) (string, error) {
	var out bytes.Buffer
	err := WriteMinifiedTo(&out, input)
	if err != nil {
		return "", err
	}
	return out.String(), nil
}
