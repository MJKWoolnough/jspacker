package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"

	"vimagination.zapto.org/css"
	"vimagination.zapto.org/parser"
)

type CSSLoader interface {
	Resolve(string) CSSLoader
	Open() (io.ReadCloser, error)
}

type cssLoader string

func (c cssLoader) Resolve(path string) CSSLoader {
	if filepath.IsAbs(path) {
		return cssLoader(path)
	}

	return cssLoader(filepath.Join(string(c), path))
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
		url, err := getCSSPath(imp.imports)
		if err != nil {
			return err
		}

		depth := 0

		if imp.layer != nil {
			depth++

			if imp.layer[0].Type == css.TokenIdent {
				w.WriteString("@layer{")
			} else {
				w.WriteString("@layer ")

				for _, tk := range imp.layer[1 : len(imp.layer)-1] {
					w.WriteString(tk.Data)
				}

				w.WriteString("{")
			}
		}

		if imp.supports != nil {
			depth++

			w.WriteString("@supports(")

			for _, tk := range imp.supports[1 : len(imp.supports)-1] {
				w.WriteString(tk.Data)
			}

			w.WriteString("){")
		}

		if imp.media != nil {
			depth++

			w.WriteString("@supports(")

			for _, tk := range imp.media {
				w.WriteString(tk.Data)
			}

			w.WriteString("){")
		}

		if err := combineCSS(loader.Resolve(url), w); err != nil {
			return err
		}

		for range depth {
			w.WriteString("}")
		}
	}

	for _, tk := range rest {
		w.WriteString(tk.Data)
	}

	return nil
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

	p.PhraserState(parseImports)

	return &p
}

func writeTokens(w io.Writer, tks []parser.Token) error {
	for _, tk := range tks {
		if _, err := io.WriteString(w, tk.Data); err != nil {
			return err
		}
	}

	return nil
}

const (
	phraseWhitespace parser.PhraseType = iota
	phraseImport
	phraseImportLayer
	phraseImportSupports
	phraseImportMedia
	phraseRemaining
)

func parseImports(p *parser.Parser) (parser.Phrase, parser.PhraseFunc) {
	if p.Accept(css.TokenWhitespace, css.TokenCDO, css.TokenCDC, css.TokenComment) {
		acceptWhitespaceComments(p)

		return p.Return(phraseWhitespace, parseImports)
	}

	if !p.AcceptToken(parser.Token{Type: css.TokenAtKeyword, Data: "@import"}) {
		return rest(p)
	}

	return parseImport(p)
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
