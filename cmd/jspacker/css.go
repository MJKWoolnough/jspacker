package main

import (
	"bytes"
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

type cssLoader string

func (c cssLoader) Resolve(path string) CSSLoader {
	return cssLoader(resolvePath(string(c), path))
}

func resolvePath(orig, path string) string {
	if filepath.IsAbs(path) {
		return path
	}

	return filepath.Join(filepath.Dir(orig), path)
}

func (c cssLoader) Open() (io.ReadCloser, error) {
	return os.Open(string(c))
}

type cssImport struct {
	imports, layer, supports, media []parser.Token
}

func combineCSS(loader CSSLoader, w *bytes.Buffer) error {
	imports, rest, err := processCSS(loader)
	if err != nil {
		return err
	}

	for _, imp := range imports {
		if imp.imports == nil {
			for _, tk := range imp.layer {
				w.WriteString(tk.Data)
			}

			continue
		}

		url, err := getCSSPath(imp.imports)
		if err != nil {
			return err
		}

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

		if err := combineCSS(loader.Resolve(url), w); err != nil {
			return err
		}

		sections.Close(w)
	}

	if len(rest) > 0 {
		switch lastByte(w.Bytes()) {
		case 0, '}', '{', '\n', ';':
		default:
			w.WriteByte('\n')
		}
	}

	for _, tk := range rest {
		w.WriteString(tk.Data)
	}

	return nil
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

func getCSSPath(imp []parser.Token) (string, error) {
	for _, i := range imp {
		switch i.Type {
		case css.TokenURL:
			return css.UnURL(i.Data)
		case css.TokenString:
			return css.Unquote(i.Data)
		}
	}

	return "", nil
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
		if err != nil {
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
		case parser.PhraseError:
			return nil, nil, p.Err
		}
	}
}

func createCSSParser(r io.Reader) *parser.Parser {
	p := parser.New(*css.CreateTokeniser(parser.NewReaderTokeniser(r), false))

	p.PhraserState(parseCSS)

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

func parseCSS(p *parser.Parser) (parser.Phrase, parser.PhraseFunc) {
	if p.AcceptToken(parser.Token{Type: css.TokenAtKeyword, Data: "@charset"}) {
		return parseCharset(p)
	}

	return parseImports(p)
}

func parseCharset(p *parser.Parser) (parser.Phrase, parser.PhraseFunc) {
	p.Accept(css.TokenWhitespace)

	if tk := p.Peek(); tk.Type != css.TokenString || !strings.HasPrefix(tk.Data, `"`) {
		return rest(p)
	}

	p.Next()

	if !p.Accept(css.TokenSemiColon) {
		return rest(p)
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
		return rest(p)
	}

	return parseImport(p)
}

func parseLayer(p *parser.Parser) (parser.Phrase, parser.PhraseFunc) {
	p.Accept(css.TokenWhitespace)

	if p.Accept(css.TokenOpenBrace) {
		return parseLayerBlock(p)
	}

	if !p.Accept(css.TokenIdent) {
		return rest(p)
	}

	p.Accept(css.TokenWhitespace)

	if p.Accept(css.TokenOpenBrace) {
		return parseLayerBlock(p)
	}

	for p.Accept(css.TokenComma) {
		p.Accept(css.TokenWhitespace)

		if !p.Accept(css.TokenIdent) {
			return rest(p)
		}

		p.Accept(css.TokenWhitespace)
	}

	if !p.Accept(css.TokenSemiColon) {
		return rest(p)
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
			return rest(p)
		}
	}
}

func parseImport(p *parser.Parser) (parser.Phrase, parser.PhraseFunc) {
	acceptWhitespaceComments(p)

	if !p.Accept(css.TokenString, css.TokenURL) {
		return rest(p)
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
			return rest(p)
		}

		for p.AcceptToken(parser.Token{Type: css.TokenDelim, Data: "."}) {
			if !p.Accept(css.TokenIdent) {
				return rest(p)
			}
		}

		if !p.Accept(css.TokenCloseParen) {
			return rest(p)
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
			return rest(p)
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

	return rest(p)
}

func acceptWhitespaceComments(p *parser.Parser) parser.TokenType {
	return p.AcceptRun(css.TokenWhitespace, css.TokenCDO, css.TokenCDC, css.TokenComment)
}

func rest(p *parser.Parser) (parser.Phrase, parser.PhraseFunc) {
	if p.ExceptRun(parser.TokenDone, parser.TokenError) == parser.TokenError {
		return p.ReturnError(p.GetError())
	}

	if p.Len() == 0 {
		return p.Done()
	}

	return p.Return(phraseRemaining, (*parser.Parser).Done)
}
