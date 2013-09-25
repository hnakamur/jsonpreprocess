package jsonpreprocess

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strings"
	"unicode"
	"unicode/utf8"
)

type item struct {
	typ itemType
	pos int
	val string
}

func (i item) String() string {
	switch {
	case i.typ == itemEOF:
		return "EOF"
	case i.typ == itemError:
		return i.val
	}
	return fmt.Sprintf("%q", i.val)
}

type itemType int

const (
	itemError itemType = iota
	itemString
	itemText
	itemBlockComment
	itemLineComment
	itemWhitespace
	itemEOF
)

const eof = -1

type stateFn func(*lexer) stateFn

type lexer struct {
	input  *bufio.Reader
	buffer bytes.Buffer
	state  stateFn
	pos    int
	start  int
	items  chan item
}

func (l *lexer) nextItem() item {
	item := <-l.items
	return item
}

func lex(input io.Reader) *lexer {
	l := &lexer{
		input: bufio.NewReader(input),
		items: make(chan item),
	}
	go l.run()
	return l
}

func (l *lexer) run() {
	for l.state = lexText; l.state != nil; {
		l.state = l.state(l)
	}
}

func (l *lexer) next() rune {
	r, w, err := l.input.ReadRune()
	if err == io.EOF {
		return eof
	}
	l.pos += w
	l.buffer.WriteRune(r)
	return r
}

func (l *lexer) peek() rune {
	lead, err := l.input.Peek(1)
	if err == io.EOF {
		return eof
	} else if err != nil {
		l.errorf("%s", err.Error())
		return 0
	}

	p, err := l.input.Peek(runeLen(lead[0]))
	if err == io.EOF {
		return eof
	} else if err != nil {
		l.errorf("%s", err.Error())
		return 0
	}
	r, _ := utf8.DecodeRune(p)
	return r
}

func runeLen(lead byte) int {
	if lead >= 0xF0 {
		return 4
	} else if lead >= 0xE0 {
		return 3
	} else if lead >= 0xC0 {
		return 2
	} else {
		return 1
	}
}

func (l *lexer) emit(t itemType) {
	l.items <- item{t, l.start, l.buffer.String()}
	l.start = l.pos
	l.buffer.Truncate(0)
}

func (l *lexer) accept(valid string) bool {
	if strings.IndexRune(valid, l.peek()) >= 0 {
		l.next()
		return true
	}
	return false
}

func (l *lexer) errorf(format string, args ...interface{}) stateFn {
	l.items <- item{itemError, l.start, fmt.Sprintf(format, args...)}
	return nil
}

func (l *lexer) hasPrefix(prefix string) bool {
	p, err := l.input.Peek(len(prefix))
	if err == io.EOF {
		return false
	} else if err != nil {
		l.errorf("%s", err.Error())
		return false
	}
	return string(p) == prefix
}

// Accept next count runes. Normally called after hasPrefix().
func (l *lexer) nextRuneCount(count int) {
	for i := 0; i < count; i++ {
		l.next()
	}
}

const (
	doubleQuote  = `"`
	lineComment  = "//"
	leftComment  = "/*"
	rightComment = "*/"
)

func lexText(l *lexer) stateFn {
	for {
		if l.hasPrefix(doubleQuote) {
			if l.pos > l.start {
				l.emit(itemText)
			}
			return lexString
		} else if l.hasPrefix(lineComment) {
			if l.pos > l.start {
				l.emit(itemText)
			}
			return lexLineComment
		} else if l.hasPrefix(leftComment) {
			if l.pos > l.start {
				l.emit(itemText)
			}
			return lexBlockComment
		}

		r := l.peek()
		if unicode.IsSpace(r) {
			if l.pos > l.start {
				l.emit(itemText)
			}
			return lexWhitespace
		} else if r == eof {
			l.next()
			break
		} else {
			l.next()
		}
	}
	if l.pos > l.start {
		l.emit(itemText)
	}
	l.emit(itemEOF)
	return nil
}

func lexString(l *lexer) stateFn {
	l.next()
	for {
		switch r := l.next(); {
		case r == '"':
			l.emit(itemString)
			return lexText
		case r == '\\':
			if l.accept(`"\/bfnrt`) {
				break
			} else if r := l.next(); r == 'u' {
				for i := 0; i < 4; i++ {
					if !l.accept("0123456789ABCDEFabcdef") {
						return l.errorf("expected 4 hexadecimal digits")
					}
				}
			} else {
				return l.errorf("unsupported escape character")
			}
		case unicode.IsControl(r):
			return l.errorf("cannot contain control characters in strings")
		case r == eof:
			return l.errorf("unclosed string")
		}
	}
}

func lexWhitespace(l *lexer) stateFn {
	for unicode.IsSpace(l.peek()) {
		l.next()
	}
	l.emit(itemWhitespace)
	return lexText
}

func lexLineComment(l *lexer) stateFn {
	for {
		r := l.next()
		if r == '\n' || r == eof {
			if l.pos > l.start {
				l.emit(itemLineComment)
			}
			if r == eof {
				l.emit(itemEOF)
				return nil
			}
			return lexText
		}
	}
}

func lexBlockComment(l *lexer) stateFn {
	for {
		if l.hasPrefix(rightComment) {
			l.nextRuneCount(utf8.RuneCountInString(rightComment))
			if l.pos > l.start {
				l.emit(itemBlockComment)
			}
			return lexText
		}
		if l.next() == eof {
			break
		}
	}
	if l.pos > l.start {
		l.emit(itemText)
	}
	l.emit(itemEOF)
	return nil
}
