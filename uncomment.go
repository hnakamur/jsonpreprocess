package jsonutil

import (
	"bytes"
	"errors"
	"io"
)

func WriteUncommentedTo(writer io.Writer, input string) error {
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
			item.val = "\n"
			fallthrough
		default:
			_, err := writer.Write([]byte(item.val))
			if err != nil {
				return err
			}
		}
    }
}

func Uncomment(input string) (string, error) {
	var out bytes.Buffer
	err := WriteUncommentedTo(&out, input)
	if err != nil {
		return "", err
	}
	return out.String(), nil
}
