package jsonutil

import (
	"fmt"
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
	itemEOF
)

const eof = -1

type stateFn func(*lexer) stateFn

type lexer struct {
	input   string
	state   stateFn
	pos     int
	start   int
	width   int
	lastPos int
	items   chan item
}

func (l *lexer) nextItem() item {
	item := <-l.items
	l.lastPos = item.pos
	return item
}

func lex(input string) *lexer {
	l := &lexer{
		input: input,
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
	if int(l.pos) >= len(l.input) {
		l.width = 0
		return eof
	}
	r, w := utf8.DecodeRuneInString(l.input[l.pos:])
	l.width = w
	l.pos += w
	return r
}

func (l *lexer) peek() rune {
	r := l.next()
	l.backup()
	return r
}

func (l *lexer) backup() {
	l.pos -= l.width
}

func (l *lexer) emit(t itemType) {
	l.items <- item{t, l.start, l.input[l.start:l.pos]}
	l.start = l.pos
}

func (l *lexer) accept(valid string) bool {
	if strings.IndexRune(valid, l.next()) >= 0 {
		return true
	}
	l.backup()
	return false
}

func (l *lexer) errorf(format string, args ...interface{}) stateFn {
	l.items <- item{itemError, l.start, fmt.Sprintf(format, args...)}
	return nil
}

const (
	doubleQuote  = `"`
	lineComment  = "//"
	leftComment  = "/*"
	rightComment = "*/"
)

func lexText(l *lexer) stateFn {
	for {
		if strings.HasPrefix(l.input[l.pos:], doubleQuote) {
			if l.pos > l.start {
				l.emit(itemText)
			}
			return lexString
		} else if strings.HasPrefix(l.input[l.pos:], lineComment) {
			if l.pos > l.start {
				l.emit(itemText)
			}
			return lexLineComment
		} else if strings.HasPrefix(l.input[l.pos:], leftComment) {
			if l.pos > l.start {
				l.emit(itemText)
			}
			return lexBlockComment
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
		if strings.HasPrefix(l.input[l.pos:], rightComment) {
			l.pos += len(rightComment)
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
