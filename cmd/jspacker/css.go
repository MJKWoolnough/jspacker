package main

import (
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

func combineCSS(loader cssLoader, w io.Writer) error {
	r, err := loader.Open()
	if err != nil {
		return err
	}

	defer r.Close()

	p := createCSSParser(r)

	for {
		ph, err := p.GetPhrase()
		if err != nil {
			return err
		}

		switch ph.Type {
		case phraseWhitespace:
			if err := writeTokens(w, ph.Data); err != nil {
				return err
			}
		case phraseImport:

		case phraseRemaining:
			if err := writeTokens(w, ph.Data); err != nil {
				return err
			}

			return nil
		case parser.PhraseError:
			return p.Err
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
	phraseRemaining
)

func parseImports(p *parser.Parser) (parser.Phrase, parser.PhraseFunc) {
	if p.Accept(css.TokenWhitespace, css.TokenCDO, css.TokenCDC, css.TokenComment) {
		acceptWhitespaceComments(p)

		return p.Return(phraseWhitespace, parseImports)
	}

	if p.AcceptToken(parser.Token{Type: css.TokenAtKeyword, Data: "@import"}) {
		acceptWhitespaceComments(p)

		if p.Accept(css.TokenString, css.TokenURL) {
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

				acceptWhitespaceComments(p)
			} else if p.AcceptToken(parser.Token{Type: css.TokenIdent, Data: "layer"}) {
				acceptWhitespaceComments(p)
			}

			if p.AcceptToken(parser.Token{Type: css.TokenFunction, Data: "supports("}) {
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
			}

			if p.ExceptRun(css.TokenSemiColon) == css.TokenSemiColon {
				p.Next()

				return p.Return(phraseImport, parseImports)
			}
		}
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
