package main

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"vimagination.zapto.org/css"
	"vimagination.zapto.org/parser"
)

func TestCSSParser(t *testing.T) {
	for n, test := range [...]struct {
		Input  string
		Output []parser.Phrase
	}{
		{ // 1
			Input: "",
			Output: []parser.Phrase{
				{Type: parser.PhraseDone, Data: nil},
			},
		},
		{ // 2
			Input: " \n\t",
			Output: []parser.Phrase{
				{Type: phraseWhitespace, Data: []parser.Token{
					{Type: css.TokenWhitespace, Data: " \n\t"},
				}},
				{Type: parser.PhraseDone, Data: nil},
			},
		},
		{ // 3
			Input: " @import url(some/url); rest",
			Output: []parser.Phrase{
				{Type: phraseWhitespace, Data: []parser.Token{
					{Type: css.TokenWhitespace, Data: " "},
				}},
				{Type: phraseImport, Data: []parser.Token{
					{Type: css.TokenAtKeyword, Data: "import"},
					{Type: css.TokenWhitespace, Data: " "},
					{Type: css.TokenURL, Data: "url(some/url)"},
					{Type: css.TokenSemiColon, Data: ";"},
				}},
				{Type: phraseWhitespace, Data: []parser.Token{
					{Type: css.TokenWhitespace, Data: " "},
				}},
				{Type: phraseRemaining, Data: []parser.Token{
					{Type: css.TokenIdent, Data: "rest"},
				}},
				{Type: parser.PhraseDone, Data: nil},
			},
		},
		{ // 4
			Input: "@import url(some/url);\n@import 'url' layer(a) ; rest",
			Output: []parser.Phrase{
				{Type: phraseImport, Data: []parser.Token{
					{Type: css.TokenAtKeyword, Data: "import"},
					{Type: css.TokenWhitespace, Data: " "},
					{Type: css.TokenURL, Data: "url(some/url)"},
					{Type: css.TokenSemiColon, Data: ";"},
				}},
				{Type: phraseWhitespace, Data: []parser.Token{
					{Type: css.TokenWhitespace, Data: "\n"},
				}},
				{Type: phraseImport, Data: []parser.Token{
					{Type: css.TokenAtKeyword, Data: "@import"},
					{Type: css.TokenWhitespace, Data: " "},
					{Type: css.TokenString, Data: "'url'"},
					{Type: css.TokenWhitespace, Data: " "},
				}},
				{Type: phraseImportLayer, Data: []parser.Token{
					{Type: css.TokenFunction, Data: "layer("},
					{Type: css.TokenIdent, Data: "a"},
					{Type: css.TokenCloseParen, Data: ")"},
					{Type: css.TokenWhitespace, Data: " "},
					{Type: css.TokenSemiColon, Data: ";"},
				}},
				{Type: phraseWhitespace, Data: []parser.Token{
					{Type: css.TokenWhitespace, Data: " "},
				}},
				{Type: phraseRemaining, Data: []parser.Token{
					{Type: css.TokenIdent, Data: "rest"},
				}},
				{Type: parser.PhraseDone, Data: nil},
			},
		},
		{ // 5
			Input: "@import 'url' layer(a.b) supports(a b() c);",
			Output: []parser.Phrase{
				{Type: phraseImport, Data: []parser.Token{
					{Type: css.TokenAtKeyword, Data: "@import"},
					{Type: css.TokenWhitespace, Data: " "},
					{Type: css.TokenString, Data: "'url'"},
					{Type: css.TokenWhitespace, Data: " "},
				}},
				{Type: phraseImportLayer, Data: []parser.Token{
					{Type: css.TokenFunction, Data: "layer("},
					{Type: css.TokenIdent, Data: "a"},
					{Type: css.TokenDelim, Data: "."},
					{Type: css.TokenIdent, Data: "b"},
					{Type: css.TokenCloseParen, Data: ")"},
					{Type: css.TokenWhitespace, Data: " "},
				}},
				{Type: phraseImportSupports, Data: []parser.Token{
					{Type: css.TokenFunction, Data: "supports("},
					{Type: css.TokenIdent, Data: "a"},
					{Type: css.TokenWhitespace, Data: " "},
					{Type: css.TokenFunction, Data: "b("},
					{Type: css.TokenCloseParen, Data: ")"},
					{Type: css.TokenWhitespace, Data: " "},
					{Type: css.TokenIdent, Data: "c"},
					{Type: css.TokenCloseParen, Data: ")"},
					{Type: css.TokenSemiColon, Data: ";"},
				}},
				{Type: parser.PhraseDone, Data: nil},
			},
		},
		{ // 6
			Input: "@import 'url' layer supports(a);",
			Output: []parser.Phrase{
				{Type: phraseImport, Data: []parser.Token{
					{Type: css.TokenAtKeyword, Data: "@import"},
					{Type: css.TokenWhitespace, Data: " "},
					{Type: css.TokenString, Data: "'url'"},
					{Type: css.TokenWhitespace, Data: " "},
				}},
				{Type: phraseImportLayer, Data: []parser.Token{
					{Type: css.TokenIdent, Data: "layer"},
					{Type: css.TokenWhitespace, Data: " "},
				}},
				{Type: phraseImportSupports, Data: []parser.Token{
					{Type: css.TokenFunction, Data: "supports("},
					{Type: css.TokenIdent, Data: "a"},
					{Type: css.TokenCloseParen, Data: ")"},
					{Type: css.TokenSemiColon, Data: ";"},
				}},
				{Type: parser.PhraseDone, Data: nil},
			},
		},
		{ // 7
			Input: "@import 'url' supports(a) screen and (orientation: landscape);",
			Output: []parser.Phrase{
				{Type: phraseImport, Data: []parser.Token{
					{Type: css.TokenAtKeyword, Data: "@import"},
					{Type: css.TokenWhitespace, Data: " "},
					{Type: css.TokenString, Data: "'url'"},
					{Type: css.TokenWhitespace, Data: " "},
				}},
				{Type: phraseImportSupports, Data: []parser.Token{
					{Type: css.TokenFunction, Data: "supports("},
					{Type: css.TokenIdent, Data: "a"},
					{Type: css.TokenCloseParen, Data: ")"},
					{Type: css.TokenWhitespace, Data: " "},
				}},
				{Type: phraseImportMedia, Data: []parser.Token{
					{Type: css.TokenIdent, Data: "screen"},
					{Type: css.TokenWhitespace, Data: " "},
					{Type: css.TokenIdent, Data: "and"},
					{Type: css.TokenWhitespace, Data: " "},
					{Type: css.TokenOpenParen, Data: "("},
					{Type: css.TokenIdent, Data: "orientation"},
					{Type: css.TokenColon, Data: ":"},
					{Type: css.TokenWhitespace, Data: " "},
					{Type: css.TokenIdent, Data: "landscape"},
					{Type: css.TokenCloseParen, Data: ")"},
					{Type: css.TokenSemiColon, Data: ";"},
				}},
				{Type: parser.PhraseDone, Data: nil},
			},
		},
		{ // 8
			Input: `@charset "utf-8";`,
			Output: []parser.Phrase{
				{Type: phraseCharset, Data: []parser.Token{
					{Type: css.TokenAtKeyword, Data: "@charset"},
					{Type: css.TokenWhitespace, Data: " "},
					{Type: css.TokenString, Data: `"utf-8"`},
					{Type: css.TokenSemiColon, Data: ";"},
				}},
				{Type: parser.PhraseDone, Data: nil},
			},
		},
		{ // 9
			Input: ` @charset "utf-8";`,
			Output: []parser.Phrase{
				{Type: phraseWhitespace, Data: []parser.Token{
					{Type: css.TokenWhitespace, Data: " "},
				}},
				{Type: phraseRemaining, Data: []parser.Token{
					{Type: css.TokenAtKeyword, Data: "@charset"},
					{Type: css.TokenWhitespace, Data: " "},
					{Type: css.TokenString, Data: `"utf-8"`},
					{Type: css.TokenSemiColon, Data: ";"},
				}},
			},
		},
		{ // 10
			Input: `@charset 'utf-8';`,
			Output: []parser.Phrase{
				{Type: phraseRemaining, Data: []parser.Token{
					{Type: css.TokenAtKeyword, Data: "@charset"},
					{Type: css.TokenWhitespace, Data: " "},
					{Type: css.TokenString, Data: `'utf-8'`},
					{Type: css.TokenSemiColon, Data: ";"},
				}},
				{Type: parser.PhraseDone, Data: nil},
			},
		},
		{ // 11
			Input: `@charset "utf-8"`,
			Output: []parser.Phrase{
				{Type: phraseRemaining, Data: []parser.Token{
					{Type: css.TokenAtKeyword, Data: "@charset"},
					{Type: css.TokenWhitespace, Data: " "},
					{Type: css.TokenString, Data: `"utf-8"`},
				}},
				{Type: parser.PhraseDone, Data: nil},
			},
		},
		{ // 12
			Input: "@charset \"utf-8\";\n@import url(some/url); rest",
			Output: []parser.Phrase{
				{Type: phraseCharset, Data: []parser.Token{
					{Type: css.TokenAtKeyword, Data: "@charset"},
					{Type: css.TokenWhitespace, Data: " "},
					{Type: css.TokenString, Data: `"utf-8"`},
					{Type: css.TokenSemiColon, Data: ";"},
				}},
				{Type: phraseWhitespace, Data: []parser.Token{
					{Type: css.TokenWhitespace, Data: "\n"},
				}},
				{Type: phraseImport, Data: []parser.Token{
					{Type: css.TokenAtKeyword, Data: "import"},
					{Type: css.TokenWhitespace, Data: " "},
					{Type: css.TokenURL, Data: "url(some/url)"},
					{Type: css.TokenSemiColon, Data: ";"},
				}},
				{Type: phraseWhitespace, Data: []parser.Token{
					{Type: css.TokenWhitespace, Data: " "},
				}},
				{Type: phraseRemaining, Data: []parser.Token{
					{Type: css.TokenIdent, Data: "rest"},
				}},
				{Type: parser.PhraseDone, Data: nil},
			},
		},
	} {
		p := createCSSParser(strings.NewReader(test.Input))

	Loop:
		for m, ph := range test.Output {
			if phr, err := p.GetPhrase(); phr.Type != ph.Type {
				if phr.Type == parser.PhraseError {
					t.Errorf("test %d.%d: unexpected error: %s", n+1, m+1, err)
				} else {
					t.Errorf("test %d.%d: Incorrect type, expecting %d, got %d", n+1, m+1, ph.Type, phr.Type)
				}

				break
			} else {
				for o, tk := range phr.Data {
					if otk := ph.Data[o]; tk.Type != otk.Type {
						if tk.Type == parser.TokenError {
							t.Errorf("test %d.%d.%d: unexpected error: %s", n+1, m+1, o+1, tk.Data)
						} else {
							t.Errorf("test %d.%d.%d: Incorrect type, expecting %d, got %d", n+1, m+1, o+1, otk.Type, tk.Type)
						}

						break Loop
					}
				}
			}
		}
	}
}

func TestCSSLoader(t *testing.T) {
	for n, test := range [...]struct {
		Input  cssLoader
		Path   string
		Output cssLoader
	}{
		{ // 1
			Input:  "/a.css",
			Path:   "/b.css",
			Output: "/b.css",
		},
		{ // 2
			Input:  "/a.css",
			Path:   "b.css",
			Output: "/b.css",
		},
		{ // 3
			Input:  "/a/b.css",
			Path:   "c.css",
			Output: "/a/c.css",
		},
		{ // 4
			Input:  "/a/b.css",
			Path:   "../c.css",
			Output: "/c.css",
		},
		{ // 5
			Input:  "/a/b.css",
			Path:   "/c.css",
			Output: "/c.css",
		},
	} {
		if output := test.Input.Resolve(test.Path).(cssLoader); output != test.Output {
			t.Errorf("%d: expecting %q, got %q", n+1, test.Output, output)
		}
	}
}

type memCSSLoader struct {
	path   string
	sheets map[string]string
}

func (m *memCSSLoader) Resolve(path string) CSSLoader {
	return &memCSSLoader{
		path:   resolvePath(m.path, path),
		sheets: m.sheets,
	}
}

func (m *memCSSLoader) Open() (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader(m.sheets[m.path])), nil
}

func TestCombineCSS(t *testing.T) {
	for n, test := range [...]struct {
		sheets map[string]string
		output string
		err    error
	}{
		{
			sheets: map[string]string{"/a.css": "abc"},
			output: "abc",
		},
		{
			sheets: map[string]string{"/a.css": `@import "b.css";abc`, "/b.css": "def"},
			output: "def\nabc",
		},
		{
			sheets: map[string]string{"/a.css": `@import "b.css" screen;abc`, "/b.css": "def"},
			output: "@media screen{def}abc",
		},
		{
			sheets: map[string]string{"/a.css": `@import "b.css" layer screen;abc`, "/b.css": "def"},
			output: "@layer{@media screen{def}}abc",
		},
		{
			sheets: map[string]string{"/a.css": `@import "b.css" layer;abc`, "/b.css": "def"},
			output: "@layer{def}abc",
		},
		{
			sheets: map[string]string{"/a.css": `@import "b.css" layer(a) other;abc`, "/b.css": "def"},
			output: "@layer a{@media other{def}}abc",
		},
		{
			sheets: map[string]string{"/a.css": `@import "b.css" layer(a);abc`, "/b.css": "def"},
			output: "@layer a{def}abc",
		},
		{
			sheets: map[string]string{"/a.css": `@import "b.css" supports(a) screen;abc`, "/b.css": "def"},
			output: "@supports(a){@media screen{def}}abc",
		},
		{
			sheets: map[string]string{"/a.css": `@import "b.css" supports(a);abc`, "/b.css": "def"},
			output: "@supports(a){def}abc",
		},
		{
			sheets: map[string]string{"/a.css": `@import "b.css" layer supports(a);abc`, "/b.css": "def"},
			output: "@layer{@supports(a){def}}abc",
		},
		{
			sheets: map[string]string{"/a.css": `@import "b.css" layer(a.b) supports(a) media;abc`, "/b.css": "def"},
			output: "@layer a.b{@supports(a){@media media{def}}}abc",
		},
		{
			sheets: map[string]string{"/a.css": "@charset \"utf-8\";abc"},
			output: "@charset \"utf-8\";abc",
		},
		{
			sheets: map[string]string{"/a.css": "@charset \"utf-8\";@import url(b.css);abc", "/b.css": "def"},
			output: "@charset \"utf-8\";def\nabc",
		},
	} {
		var buf bytes.Buffer

		if err := combineCSS(&memCSSLoader{"/a.css", test.sheets}, &buf); !errors.Is(err, test.err) {
			t.Errorf("test %d: expecting error %v, got %v", n+1, test.err, err)
		} else if out := buf.String(); out != test.output {
			t.Errorf("test %d: expecting output %q, got %q", n+1, test.output, out)
		}
	}
}
