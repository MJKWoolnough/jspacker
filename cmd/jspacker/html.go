package main

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"vimagination.zapto.org/javascript"
	"vimagination.zapto.org/jspacker"
	"vimagination.zapto.org/parser"
)

func (c *Config) processHTML() (err error) {
	if c.base == "" {
		c.base, err = os.Getwd()
		if err != nil {
			return err
		}
	}

	h, err := c.processHTMLInput()
	if err != nil {
		return err
	}

	f, err := c.outputFile()
	if err != nil {
		return err
	}

	defer func() {
		if errr := f.Close(); err == nil {
			err = errr
		}
	}()

	return c.writeHTML(f, h)
}

func (c *Config) processHTMLInput() (*htmlState, error) {
	if len(c.filesTodo) != 1 {
		return nil, ErrInvalidHTMLInput
	}

	f := os.Stdin

	if c.filesTodo[0] != "-" {
		var err error

		f, err = os.Open(filepath.Join(c.base, c.filesTodo[0]))
		if err != nil {
			return nil, err
		}
	}

	defer f.Close()

	h := newHTMLState(f)

	for {
		if err := h.processToken(); errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return nil, fmt.Errorf("error parsing HTML input: %w", err)
		}
	}

	return h, nil
}

func (c *Config) writeHTML(w io.Writer, h *htmlState) error {
	html := h.buf.String()

	var lastPos int64

	for _, script := range h.scripts {
		if _, err := io.WriteString(w, html[lastPos:script.tagStart]); err != nil {
			return err
		}

		if script.isMap {
			if err := c.importMap.Import(strings.NewReader(html[script.contentStart:script.contentEnd])); err != nil {
				return err
			}
		} else if err := c.processScript(w, html, script); err != nil {
			return err
		}

		lastPos = script.tagEnd
	}

	_, err := io.WriteString(w, html[lastPos:])

	return err
}

func (c *Config) processScript(w io.Writer, html string, script script) error {
	opts := c.Options()

	if _, err := io.WriteString(w, `<script type="module">`); err != nil {
		return err
	}

	if script.src == "" {
		opts = append(opts, jspacker.Loader(scriptLoader(html[script.contentStart:script.contentEnd], c.base)))
		c.filesTodo[0] = "/\x00"
	} else {
		c.filesTodo[0] = path.Join("/", script.src)
	}

	m, err := jspacker.Package(c.Options()...)
	if err != nil {
		return fmt.Errorf("error generating output: %w", err)
	}

	if err = c.writeOutput(w, m); err != nil {
		return err
	}

	_, err = io.WriteString(w, `</script>`)

	return err
}

type htmlState struct {
	buf      strings.Builder
	scripts  []script
	dec      *xml.Decoder
	lastPos  int64
	inScript bool
}

func newHTMLState(r io.Reader) *htmlState {
	h := new(htmlState)
	h.dec = xml.NewDecoder(io.TeeReader(r, &h.buf))
	h.dec.Strict = false
	h.dec.AutoClose = xml.HTMLAutoClose
	h.dec.Entity = xml.HTMLEntity

	return h
}

func (h *htmlState) processToken() error {
	tk, err := h.dec.Token()
	if err != nil {
		return err
	}

	switch tk := tk.(type) {
	case xml.StartElement:
		h.addScript(tk)
	case xml.EndElement:
		h.endScript(tk)
	}

	h.lastPos = h.dec.InputOffset()

	return nil
}

func (h *htmlState) addScript(tk xml.StartElement) {
	if h.inScript || tk.Name.Local != "script" {
		return
	}

	s := script{
		tagStart:     h.lastPos,
		contentStart: h.dec.InputOffset(),
	}

	for _, attr := range tk.Attr {
		switch attr.Name.Local {
		case "type":
			switch attr.Value {
			case "importmap":
				s.isMap = true
			}
		case "src":
			s.src = attr.Value
		}
	}

	h.inScript = true
	h.scripts = append(h.scripts, s)
}

func (h *htmlState) endScript(tk xml.EndElement) {
	if !h.inScript || tk.Name.Local != "script" {
		return
	}

	h.inScript = false
	h.scripts[len(h.scripts)-1].contentEnd = h.lastPos
	h.scripts[len(h.scripts)-1].tagEnd = h.dec.InputOffset()
}

type script struct {
	tagStart, tagEnd, contentStart, contentEnd int64
	src                                        string
	isMap                                      bool
}

func scriptLoader(src, base string) func(string) (*javascript.Module, error) {
	loader := jspacker.OSLoad(base)

	return func(file string) (*javascript.Module, error) {
		if file != "/\x00" {
			return loader(file)
		}

		tk := parser.NewStringTokeniser(src)

		return javascript.ParseModule(&tk)
	}
}

var ErrInvalidHTMLInput = errors.New("must specify single HTML input file")
