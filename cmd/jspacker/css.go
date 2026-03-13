package main

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"vimagination.zapto.org/css"
	"vimagination.zapto.org/parser"
)

type CSSLoader interface {
	Resolve(string) CSSLoader
	Open() (io.ReadCloser, error)
}

type cssLoader struct {
	base, path, source string
}

func (c cssLoader) Resolve(path string) CSSLoader {
	return cssLoader{base: c.base, path: resolvePath(c.path, path)}
}

func resolvePath(orig, path string) string {
	if filepath.IsAbs(path) {
		return path
	}

	return filepath.Join(filepath.Dir(orig), path)
}

func (c cssLoader) Open() (io.ReadCloser, error) {
	if c.source != "" {
		return io.NopCloser(strings.NewReader(c.source)), nil
	}

	return os.Open(filepath.Join(c.base, c.path))
}

type cssImport struct {
	imports, layer, supports, media []parser.Token
}

func combineCSS(loader CSSLoader, w *bytes.Buffer, minimise bool) error {
	imports, rest, err := processCSS(loader)
	if err != nil {
		return err
	}

	for _, imp := range imports {
		if imp.imports == nil {
			printTokens(w, imp.layer, minimise)

			continue
		}

		if err := writeCSS(loader, w, imp, minimise); err != nil {
			return err
		}
	}

	if len(rest) > 0 {
		switch lastByte(w.Bytes()) {
		case 0, '}', '{', '\n', ';':
		default:
			w.WriteByte('\n')
		}
	}

	printTokens(w, rest, minimise)

	return nil
}

func printTokens(w *bytes.Buffer, tks []parser.Token, minimise bool) {
	for _, tk := range tks {
		if minimise && tk.Type == css.TokenWhitespace {
			switch lastByte(w.Bytes()) {
			case ' ', '\n', '\r', '\f', '\t', '}', '{', '>', ':', ';', ',':
			default:
				w.WriteString(" ")
			}

			continue
		}

		if minimise && lastByte(w.Bytes()) == ' ' {
			switch firstByte(tk.Data) {
			case '+', '-':
			default:
				w.Truncate(w.Len() - 1)
			}
		}

		w.WriteString(tk.Data)
	}
}

func writeCSS(loader CSSLoader, w *bytes.Buffer, imp cssImport, minimise bool) error {
	url := getCSSPath(imp.imports)

	var sections cssSection

	if imp.layer != nil {
		if imp.layer[0].Type == css.TokenIdent {
			sections.Write(w, "@layer", nil)
		} else {
			sections.Write(w, "@layer ", imp.layer[1:len(imp.layer)-2])
		}
	}

	if imp.supports != nil {
		sections.Write(w, "@supports(", imp.supports[1:len(imp.supports)-1])
	}

	if imp.media != nil {
		sections.Write(w, "@media ", imp.media)
	}

	if err := combineCSS(loader.Resolve(url), w, minimise); err != nil {
		return err
	}

	sections.Close(w)

	return nil
}

func firstByte(str string) byte {
	if len(str) == 0 {
		return 0
	}

	return str[0]
}

func lastByte(bytes []byte) byte {
	if len(bytes) == 0 {
		return 0
	}

	return bytes[len(bytes)-1]
}

type cssSection int

func (c *cssSection) Write(w *bytes.Buffer, at string, tokens []parser.Token) {
	(*c)++

	w.WriteString(at)

	for _, tk := range tokens {
		if tk.Type == css.TokenSemiColon {
			continue
		}

		w.WriteString(tk.Data)
	}

	w.WriteString("{")
}

func (c *cssSection) Close(w *bytes.Buffer) {
	for range *c {
		w.WriteString("}")
	}
}

func getCSSPath(imp []parser.Token) string {
	var url string

	for _, i := range imp {
		var fn func(string) (string, error)

		switch i.Type {
		case css.TokenURL:
			fn = css.UnURL
		case css.TokenString:
			fn = css.Unquote
		default:
			continue
		}

		url, _ = fn(i.Data)
	}

	return url
}

func processCSS(loader CSSLoader) ([]cssImport, []parser.Token, error) {
	r, err := loader.Open()
	if err != nil {
		return nil, nil, err
	}

	defer r.Close()

	p := createCSSParser(r)

	var imports []cssImport

	for {
		ph, err := p.GetPhrase()
		if errors.Is(err, io.EOF) {
			return imports, nil, nil
		} else if err != nil {
			return nil, nil, err
		}

		switch ph.Type {
		case phraseCharset, phraseLayer:
			imports = append(imports, cssImport{layer: ph.Data})
		case phraseImport:
			imports = append(imports, cssImport{imports: ph.Data})
		case phraseImportLayer:
			imports[len(imports)-1].layer = ph.Data
		case phraseImportSupports:
			imports[len(imports)-1].supports = ph.Data
		case phraseImportMedia:
			imports[len(imports)-1].media = ph.Data
		case phraseRemaining:
			return imports, ph.Data, nil
		}
	}
}

func createCSSParser(r io.Reader) *parser.Parser {
	p := parser.New(*css.CreateTokeniser(parser.NewReaderTokeniser(r), false))

	p.PhraserState(parseSheet)

	return &p
}

const (
	phraseWhitespace parser.PhraseType = iota
	phraseImport
	phraseImportLayer
	phraseImportSupports
	phraseImportMedia
	phraseCharset
	phraseLayer
	phraseRemaining
)

func parseSheet(p *parser.Parser) (parser.Phrase, parser.PhraseFunc) {
	if p.AcceptToken(parser.Token{Type: css.TokenAtKeyword, Data: "@charset"}) {
		return parseCharset(p)
	}

	return parseImports(p)
}

func parseCharset(p *parser.Parser) (parser.Phrase, parser.PhraseFunc) {
	p.Accept(css.TokenWhitespace)

	if tk := p.Peek(); tk.Type != css.TokenString || !strings.HasPrefix(tk.Data, `"`) {
		return parseRemaining(p)
	}

	p.Next()

	if !p.Accept(css.TokenSemiColon) {
		return parseRemaining(p)
	}

	return p.Return(phraseCharset, parseImports)
}

func parseImports(p *parser.Parser) (parser.Phrase, parser.PhraseFunc) {
	if p.Accept(css.TokenWhitespace, css.TokenCDO, css.TokenCDC, css.TokenComment) {
		acceptWhitespaceComments(p)

		return p.Return(phraseWhitespace, parseImports)
	}

	if p.AcceptToken(parser.Token{Type: css.TokenAtKeyword, Data: "@layer"}) {
		return parseLayer(p)
	} else if !p.AcceptToken(parser.Token{Type: css.TokenAtKeyword, Data: "@import"}) {
		return parseRemaining(p)
	}

	return parseImport(p)
}

func parseLayer(p *parser.Parser) (parser.Phrase, parser.PhraseFunc) {
	p.Accept(css.TokenWhitespace)

	if p.Accept(css.TokenOpenBrace) {
		return parseLayerBlock(p)
	}

	if !p.Accept(css.TokenIdent) {
		return parseRemaining(p)
	}

	p.Accept(css.TokenWhitespace)

	if p.Accept(css.TokenOpenBrace) {
		return parseLayerBlock(p)
	}

	for p.Accept(css.TokenComma) {
		p.Accept(css.TokenWhitespace)

		if !p.Accept(css.TokenIdent) {
			return parseRemaining(p)
		}

		p.Accept(css.TokenWhitespace)
	}

	if !p.Accept(css.TokenSemiColon) {
		return parseRemaining(p)
	}

	return p.Return(phraseLayer, parseImports)
}

func parseLayerBlock(p *parser.Parser) (parser.Phrase, parser.PhraseFunc) {
	depth := 1

	for {
		switch p.ExceptRun(css.TokenCloseBrace, css.TokenOpenBrace) {
		case css.TokenOpenBrace:
			p.Next()

			depth++
		case css.TokenCloseBrace:
			p.Next()

			depth--

			if depth == 0 {
				return p.Return(phraseLayer, parseImports)
			}
		default:
			return parseRemaining(p)
		}
	}
}

func parseImport(p *parser.Parser) (parser.Phrase, parser.PhraseFunc) {
	acceptWhitespaceComments(p)

	if !p.Accept(css.TokenString, css.TokenURL) {
		return parseRemaining(p)
	}

	acceptWhitespaceComments(p)

	if p.AcceptToken(parser.Token{Type: css.TokenSemiColon, Data: ";"}) {
		return p.Return(phraseImport, parseImports)
	}

	return p.Return(phraseImport, parseLayerSupportOrMedia)
}

func parseLayerSupportOrMedia(p *parser.Parser) (parser.Phrase, parser.PhraseFunc) {
	acceptWhitespaceComments(p)

	if p.AcceptToken(parser.Token{Type: css.TokenFunction, Data: "layer("}) {
		if !p.Accept(css.TokenIdent) {
			return parseRemaining(p)
		}

		for p.AcceptToken(parser.Token{Type: css.TokenDelim, Data: "."}) {
			if !p.Accept(css.TokenIdent) {
				return parseRemaining(p)
			}
		}

		if !p.Accept(css.TokenCloseParen) {
			return parseRemaining(p)
		}

	} else if !p.AcceptToken(parser.Token{Type: css.TokenIdent, Data: "layer"}) {
		return parseSupportOrMedia(p)
	}

	acceptWhitespaceComments(p)

	if p.AcceptToken(parser.Token{Type: css.TokenSemiColon, Data: ";"}) {
		return p.Return(phraseImportLayer, parseImports)
	}

	return p.Return(phraseImportLayer, parseSupportOrMedia)
}

func parseSupportOrMedia(p *parser.Parser) (parser.Phrase, parser.PhraseFunc) {
	acceptWhitespaceComments(p)

	if !p.AcceptToken(parser.Token{Type: css.TokenFunction, Data: "supports("}) {
		return parseMedia(p)
	}

	depth := 1

Loop:
	for {
		switch p.ExceptRun(css.TokenCloseParen, css.TokenOpenParen, css.TokenFunction) {
		case css.TokenCloseParen:
			p.Next()

			depth--

			if depth == 0 {
				break Loop
			}
		case css.TokenOpenParen, css.TokenFunction:
			p.Next()

			depth++
		default:
			return parseRemaining(p)
		}
	}

	acceptWhitespaceComments(p)

	if p.AcceptToken(parser.Token{Type: css.TokenSemiColon, Data: ";"}) {
		return p.Return(phraseImportSupports, parseImports)
	}

	return p.Return(phraseImportSupports, parseMedia)
}

func parseMedia(p *parser.Parser) (parser.Phrase, parser.PhraseFunc) {
	acceptWhitespaceComments(p)

	if p.ExceptRun(css.TokenSemiColon) == css.TokenSemiColon {
		p.Next()

		return p.Return(phraseImportMedia, parseImports)
	}

	return parseRemaining(p)
}

func acceptWhitespaceComments(p *parser.Parser) parser.TokenType {
	return p.AcceptRun(css.TokenWhitespace, css.TokenCDO, css.TokenCDC, css.TokenComment)
}

func parseRemaining(p *parser.Parser) (parser.Phrase, parser.PhraseFunc) {
	if p.ExceptRun(parser.TokenDone, parser.TokenError) == parser.TokenError {
		return p.ReturnError(p.GetError())
	}

	if p.Len() == 0 {
		return p.Done()
	}

	return p.Return(phraseRemaining, (*parser.Parser).Done)
}
