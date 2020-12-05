package jsonpreprocess

import (
	"bufio"
	"bytes"
	"errors"
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
	return fmt.Sprintf("%s:%q", i.typ, i.val)
}

type itemType int

func (t itemType) String() string {
	switch t {
	case itemError:
		return "error"
	case itemString:
		return "string"
	case itemLeftBrace:
		return "leftBrace"
	case itemRightBrace:
		return "rightBrace"
	case itemLeftBracket:
		return "leftBracket"
	case itemRightBracket:
		return "rightBracket"
	case itemComma:
		return "comma"
	case itemColon:
		return "colon"
	case itemNumber:
		return "number"
	case itemTrue:
		return "true"
	case itemFalse:
		return "false"
	case itemNull:
		return "null"
	case itemWhitespace:
		return "whitespace"
	case itemIdentifier:
		return "identifier"
	case itemBlockComment:
		return "blockComment"
	case itemLineComment:
		return "lineComment"
	default:
		panic(errors.New("unexpected itemType"))
	}
}

const (
	itemError itemType = iota
	itemString
	itemBlockComment
	itemLineComment
	itemLeftBrace
	itemRightBrace
	itemLeftBracket
	itemRightBracket
	itemComma
	itemColon
	itemNumber
	itemTrue
	itemFalse
	itemNull
	itemIdentifier
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
	if lead < 0xC0 {
		return 1
	} else if lead < 0xE0 {
		return 2
	} else if lead < 0xF0 {
		return 3
	} else {
		return 4
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

func (l *lexer) acceptRun(valid string) {
	for strings.IndexRune(valid, l.peek()) >= 0 {
		l.next()
	}
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
Loop:
	for {
		r := l.peek()
		switch r {
		case '"':
			return lexString
		case ':':
			l.next()
			l.emit(itemColon)
		case ',':
			l.next()
			l.emit(itemComma)
		case '{':
			l.next()
			l.emit(itemLeftBrace)
		case '}':
			l.next()
			l.emit(itemRightBrace)
		case '[':
			l.next()
			l.emit(itemLeftBracket)
		case ']':
			l.next()
			l.emit(itemRightBracket)
		case '#':
			return lexLineComment
		case '/':
			if l.hasPrefix(lineComment) {
				return lexLineComment
			} else if l.hasPrefix(leftComment) {
				return lexBlockComment
			} else {
				return l.errorf("invalid character after slash")
			}
		default:
			if unicode.IsSpace(r) {
				return lexWhitespace
			} else if r == eof {
				l.next()
				break Loop
			} else if strings.IndexRune("0123456789-", l.peek()) >= 0 {
				return lexNumber
			} else {
				return lexIdentifier
			}
		}
	}
	l.emit(itemEOF)
	return nil
}

const (
	hexdigit  = "0123456789ABCDEFabcdef"
	digit     = "0123456789"
	digit1To9 = "123456789"
)

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
					if !l.accept(hexdigit) {
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

func lexNumber(l *lexer) stateFn {
	l.accept("-")
	if l.accept(digit1To9) {
		l.acceptRun(digit)
	} else if !l.accept("0") {
		return l.errorf("bad digit for number")
	}
	if l.accept(".") {
		l.acceptRun(digit)
	}
	if l.accept("eE") {
		l.accept("+-")
		if !l.accept(digit) {
			return l.errorf("digit expected for number exponent")
		}
		l.acceptRun(digit)
	}
	l.emit(itemNumber)
	return lexText
}

func lexIdentifier(l *lexer) stateFn {
	r := l.peek()
	if unicode.IsLetter(r) || r == '$' || r == '_' {
		// do nothing
	} else if r == '\\' {
		if !l.accept("u") {
			return l.errorf("'u' for unicode escape sequence expected")
		}
		for i := 0; i < 4; i++ {
			if !l.accept(hexdigit) {
				return l.errorf("expected 4 hexadecimal digits for unicode escape sequence")
			}
		}
	} else {
		return l.errorf("identifier expected")
	}

	for r = l.peek(); isIdentifierPart(r); {
		l.next()
		r = l.peek()
	}

	text := l.buffer.String()
	if text == "true" {
		l.emit(itemTrue)
	} else if text == "false" {
		l.emit(itemFalse)
	} else if text == "null" {
		l.emit(itemNull)
	} else {
		l.emit(itemIdentifier)
	}
	return lexText
}

func isIdentifierPart(r rune) bool {
	return (unicode.IsLetter(r) || unicode.IsMark(r) || unicode.IsDigit(r) ||
		unicode.IsPunct(r)) && !strings.ContainsRune("{}[]:,", r)
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
	l.emit(itemEOF)
	return nil
}
