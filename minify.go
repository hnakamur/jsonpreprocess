package jsonpreprocess

import (
	"bytes"
	"errors"
	"io"
)

func WriteMinifiedTo(writer io.Writer, reader io.Reader) error {
	l := lex(reader)
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
	reader := bytes.NewBufferString(input)
	err := WriteMinifiedTo(&out, reader)
	if err != nil {
		return "", err
	}
	return out.String(), nil
}
