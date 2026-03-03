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

	p := parser.New(*css.CreateTokeniser(parser.NewReaderTokeniser(r), false))

	p.PhraserState(parseImports)

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
	return p.Done()
}
