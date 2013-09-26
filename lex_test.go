package jsonpreprocess

import (
	"bytes"
	"testing"
)

type lexTest struct {
	name  string
	input string
	items []item
}

var tEOF = item{itemEOF, 0, ""}

var lexTests = []lexTest{
	{"empty", "", []item{tEOF}},
	{"spaces", " \t\n", []item{{itemWhitespace, 0, " \t\n"}, tEOF}},
	{"text", `[1, 2]`, []item{
		{itemLeftBracket, 0, `[`},
		{itemNumber, 0, `1`},
		{itemComma, 0, `,`},
		{itemWhitespace, 0, ` `},
		{itemNumber, 0, `2`},
		{itemRightBracket, 0, `]`},
		tEOF,
	}},
	{"string", `"foo"`, []item{{itemString, 0, `"foo"`}, tEOF}},
	{"quotation mark escape", `"\""`, []item{{itemString, 0, `"\""`}, tEOF}},
	{"reverse solidus escape", `"\\"`, []item{{itemString, 0, `"\\"`}, tEOF}},
	{"solidus escape", `"\/"`, []item{{itemString, 0, `"\/"`}, tEOF}},
	{"backspace escape", `"\b"`, []item{{itemString, 0, `"\b"`}, tEOF}},
	{"formfeed escape", `"\f"`, []item{{itemString, 0, `"\f"`}, tEOF}},
	{"newline escape", `"\n"`, []item{{itemString, 0, `"\n"`}, tEOF}},
	{"carriage return escape", `"\r"`, []item{{itemString, 0, `"\r"`}, tEOF}},
	{"horizontal tab escape", `"\t"`, []item{{itemString, 0, `"\t"`}, tEOF}},
	{"unicode escape", `"\u1234"`, []item{{itemString, 0, `"\u1234"`}, tEOF}},
	{"invalid escape", `"\x23"`, []item{
		{itemError, 0, "unsupported escape character"},
	}},
	{"invalid unicode escape", `"\u123g"`, []item{
		{itemError, 0, "expected 4 hexadecimal digits"},
	}},
	{"unclosed string", `"foo`, []item{
		{itemError, 0, "unclosed string"},
	}},
	{"control character in string", "\"foo\tbar\"", []item{
		{itemError, 0, "cannot contain control characters in strings"},
	}},
	{"text with string", `{"foo": 1}`, []item{
		{itemLeftBrace, 0, `{`},
		{itemString, 0, `"foo"`},
		{itemColon, 0, `:`},
		{itemWhitespace, 0, ` `},
		{itemNumber, 0, `1`},
		{itemRightBrace, 0, `}`},
		tEOF,
	}},
	{"text with line comment ", `[1, 2] // this is a line comment`, []item{
		{itemLeftBracket, 0, `[`},
		{itemNumber, 0, `1`},
		{itemComma, 0, `,`},
		{itemWhitespace, 0, ` `},
		{itemNumber, 0, `2`},
		{itemRightBracket, 0, `]`},
		{itemWhitespace, 0, ` `},
		{itemLineComment, 0, `// this is a line comment`},
		tEOF,
	}},
	{"text with block comment ", "[1, 2, /* this is\na block comment */ 3]", []item{
		{itemLeftBracket, 0, `[`},
		{itemNumber, 0, `1`},
		{itemComma, 0, `,`},
		{itemWhitespace, 0, ` `},
		{itemNumber, 0, `2`},
		{itemComma, 0, `,`},
		{itemWhitespace, 0, ` `},
		{itemBlockComment, 0, "/* this is\na block comment */"},
		{itemWhitespace, 0, ` `},
		{itemNumber, 0, `3`},
		{itemRightBracket, 0, `]`},
		tEOF,
	}},
	{"text with string and comment", `{"url": "http://example.com"} // this is a line comment`, []item{
		{itemLeftBrace, 0, `{`},
		{itemString, 0, `"url"`},
		{itemColon, 0, `:`},
		{itemWhitespace, 0, ` `},
		{itemString, 0, `"http://example.com"`},
		{itemRightBrace, 0, `}`},
		{itemWhitespace, 0, ` `},
		{itemLineComment, 0, `// this is a line comment`},
		tEOF,
	}},
	{"block comment inside stringtext with block comment ", `{"key": "This is a value /* this is a block comment inside a string */"}`, []item{
		{itemLeftBrace, 0, `{`},
		{itemString, 0, `"key"`},
		{itemColon, 0, `:`},
		{itemWhitespace, 0, ` `},
		{itemString, 0, `"This is a value /* this is a block comment inside a string */"`},
		{itemRightBrace, 0, `}`},
		tEOF,
	}},
	{"loose json with identifier key", `{foo: 1}`, []item{
		{itemLeftBrace, 0, `{`},
		{itemIdentifier, 0, `foo`},
		{itemColon, 0, `:`},
		{itemWhitespace, 0, ` `},
		{itemNumber, 0, `1`},
		{itemRightBrace, 0, `}`},
		tEOF,
	}},
	{"loose json with identifier key contains keyword", `{truey: true}`, []item{
		{itemLeftBrace, 0, `{`},
		{itemIdentifier, 0, `truey`},
		{itemColon, 0, `:`},
		{itemWhitespace, 0, ` `},
		{itemTrue, 0, `true`},
		{itemRightBrace, 0, `}`},
		tEOF,
	}},
	{"number zero", `0`, []item{{itemNumber, 0, `0`}, tEOF}},
	{"fraction number with zero integer part", `0.12`, []item{{itemNumber, 0, `0.12`}, tEOF}},
	{"negative number", `-0.12`, []item{{itemNumber, 0, `-0.12`}, tEOF}},
	{"fraction number with non-zero integer part", `10.12`, []item{{itemNumber, 0, `10.12`}, tEOF}},
	{"number with no-sign exponent", `1e2`, []item{{itemNumber, 0, `1e2`}, tEOF}},
	{"number with minus exponent", `1e-2`, []item{{itemNumber, 0, `1e-2`}, tEOF}},
	{"number with plus exponent", `1E+99`, []item{{itemNumber, 0, `1E+99`}, tEOF}},
	{"number with fraction and plus exponent", `1.23E+99`, []item{{itemNumber, 0, `1.23E+99`}, tEOF}},
}

func collect(t *lexTest) (items []item) {
	buf := bytes.NewBufferString(t.input)
	l := lex(buf)
	for {
		item := l.nextItem()
		items = append(items, item)
		if item.typ == itemEOF || item.typ == itemError {
			break
		}
	}
	return
}

func equal(i1, i2 []item, checkPos bool) bool {
	if len(i1) != len(i2) {
		return false
	}
	for k := range i1 {
		if i1[k].typ != i2[k].typ {
			return false
		}
		if i1[k].val != i2[k].val {
			return false
		}
		if checkPos && i1[k].pos != i2[k].pos {
			return false
		}
	}
	return true
}

func TestLex(t *testing.T) {
	for _, test := range lexTests {
		items := collect(&test)
		if !equal(items, test.items, false) {
			t.Errorf("%s: got\n\t%+v\nexpected\n\t%v", test.name, items, test.items)
		}
	}
}
