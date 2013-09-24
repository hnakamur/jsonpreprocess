package jsonutil

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
		{itemText, 0, `[1,`},
		{itemWhitespace, 0, ` `},
		{itemText, 0, `2]`},
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
		{itemText, 0, `{`},
		{itemString, 0, `"foo"`},
		{itemText, 0, `:`},
		{itemWhitespace, 0, ` `},
		{itemText, 0, `1}`},
		tEOF,
	}},
	{"text with line comment ", `[1, 2] // this is a line comment`, []item{
		{itemText, 0, `[1,`},
		{itemWhitespace, 0, ` `},
		{itemText, 0, `2]`},
		{itemWhitespace, 0, ` `},
		{itemLineComment, 0, `// this is a line comment`},
		tEOF,
	}},
	{"text with block comment ", "[1, 2, /* this is\na block comment */ 3]", []item{
		{itemText, 0, `[1,`},
		{itemWhitespace, 0, ` `},
		{itemText, 0, `2,`},
		{itemWhitespace, 0, ` `},
		{itemBlockComment, 0, "/* this is\na block comment */"},
		{itemWhitespace, 0, ` `},
		{itemText, 0, `3]`},
		tEOF,
	}},
	{"text with string and comment", `{"url": "http://example.com"} // this is a line comment`, []item{
		{itemText, 0, `{`},
		{itemString, 0, `"url"`},
		{itemText, 0, `:`},
		{itemWhitespace, 0, ` `},
		{itemString, 0, `"http://example.com"`},
		{itemText, 0, `}`},
		{itemWhitespace, 0, ` `},
		{itemLineComment, 0, `// this is a line comment`},
		tEOF,
	}},
	{"block comment inside stringtext with block comment ",
		`{"key": "This is a value /* this is a block comment inside a string */"}`, []item{
			{itemText, 0, `{`},
			{itemString, 0, `"key"`},
			{itemText, 0, `:`},
			{itemWhitespace, 0, ` `},
			{itemString, 0, `"This is a value /* this is a block comment inside a string */"`},
			{itemText, 0, `}`},
			tEOF,
		}},
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
