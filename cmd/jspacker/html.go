package main

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
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

	h.processCSS = c.processCSS

	for {
		if err := h.processToken(); errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return nil, fmt.Errorf("error parsing HTML input: %w", err)
		}
	}

	return h, nil
}

var (
	styleStart = []byte(`<style type="text/css">`)
	styleEnd   = []byte(`</style>`)
)

func (c *Config) writeHTML(w io.Writer, h *htmlState) error {
	html := h.buf.String()

	var lastPos int64

	for _, tag := range h.tags {
		if _, err := io.WriteString(w, html[lastPos:tag.tagStart]); err != nil {
			return err
		}

		switch tag.tagType {
		case tagImportMap:
			if err := c.importMap.Import(strings.NewReader(html[tag.contentStart:tag.contentEnd])); err != nil {
				return err
			}
		case tagScript:
			if err := c.processScript(w, html, tag); err != nil {
				return err
			}
		case tagStyle:
			if err := c.processStyle(w, html, tag); err != nil {
				return err
			}
		case tagLink:
			if err := c.processLink(w, tag); err != nil {
				return err
			}
		}

		lastPos = tag.tagEnd
	}

	_, err := io.WriteString(w, html[lastPos:])

	return err
}

func (c *Config) processScript(w io.Writer, html string, script tag) error {
	var opts []jspacker.Option

	if _, err := io.WriteString(w, `<script type="module">`); err != nil {
		return err
	}

	if script.src == "" {
		c.filesTodo[0] = "/\x00"
		opts = append(c.Options(), jspacker.Loader(scriptLoader(html[script.contentStart:script.contentEnd], c.base)))
	} else {
		c.filesTodo[0] = c.importMap.Resolve("/", script.src)
		opts = c.Options()
	}

	m, err := jspacker.Package(opts...)
	if err != nil {
		return fmt.Errorf("error generating output: %w", err)
	}

	if err = c.writeOutput(w, m); err != nil {
		return err
	}

	_, err = io.WriteString(w, `</script>`)

	return err
}

func (c *Config) processStyle(w io.Writer, html string, tag tag) error {
	var buf bytes.Buffer

	if err := combineCSS(cssLoader{base: c.base, path: "/", source: html[tag.contentStart:tag.contentEnd]}, &buf); err != nil {
		return err
	}

	if _, err := w.Write(styleStart); err != nil {
		return err
	}

	buf.WriteString(string(styleEnd))

	if _, err := io.Copy(w, &buf); err != nil {
		return err
	}

	return nil
}

func (c *Config) processLink(w io.Writer, tag tag) error {
	var buf bytes.Buffer

	if err := combineCSS((cssLoader{base: c.base, path: c.html}).Resolve(tag.src), &buf); err != nil {
		return err
	}

	if _, err := w.Write(styleStart); err != nil {
		return err
	}

	buf.WriteString(string(styleEnd))

	if _, err := io.Copy(w, &buf); err != nil {
		return err
	}

	return nil
}

type htmlState struct {
	processCSS                bool
	buf                       strings.Builder
	tags                      []tag
	dec                       *xml.Decoder
	lastPos                   int64
	inScript, inStyle, inLink bool
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
		switch tk.Name.Local {
		case "script":
			h.addScript(tk)
		case "style":
			h.addStyle()
		case "link":
			h.addLink(tk)
		}
	case xml.EndElement:
		switch tk.Name.Local {
		case "script":
			h.endScript(tk)
		case "style":
			h.endStyle()
		case "link":
			h.endLink()
		}
	}

	h.lastPos = h.dec.InputOffset()

	return nil
}

func (h *htmlState) addScript(tk xml.StartElement) {
	if h.inScript {
		return
	}

	s := tag{
		tagType:      tagScript,
		tagStart:     h.lastPos,
		contentStart: h.dec.InputOffset(),
	}

	for _, attr := range tk.Attr {
		switch attr.Name.Local {
		case "type":
			switch attr.Value {
			case "importmap":
				s.tagType = tagImportMap
			}
		case "src":
			s.src = attr.Value
		}
	}

	h.inScript = true
	h.tags = append(h.tags, s)
}

func (h *htmlState) addStyle() {
	if h.inStyle || !h.processCSS {
		return
	}

	h.inStyle = true
	h.tags = append(h.tags, tag{
		tagType:      tagStyle,
		tagStart:     h.lastPos,
		contentStart: h.dec.InputOffset(),
	})
}

func (h *htmlState) addLink(tk xml.StartElement) {
	if h.inLink || !h.processCSS {
		return
	}

	s := tag{
		tagType:      tagLink,
		tagStart:     h.lastPos,
		contentStart: h.dec.InputOffset(),
	}

	var rel string

	for _, attr := range tk.Attr {
		switch attr.Name.Local {
		case "rel":
			rel = attr.Value
		case "href":
			s.src = attr.Value
		}
	}

	if rel != "stylesheet" {
		return
	}

	h.inLink = true
	h.tags = append(h.tags, s)
}

func (h *htmlState) endScript(tk xml.EndElement) {
	if !h.inScript || tk.Name.Local != "script" {
		return
	}

	h.inScript = false
	h.tags[len(h.tags)-1].contentEnd = h.lastPos
	h.tags[len(h.tags)-1].tagEnd = h.dec.InputOffset()
}

func (h *htmlState) endStyle() {
	if !h.inStyle {
		return
	}

	h.inStyle = false
	h.tags[len(h.tags)-1].contentEnd = h.lastPos
	h.tags[len(h.tags)-1].tagEnd = h.dec.InputOffset()
}

func (h *htmlState) endLink() {
	if !h.inLink {
		return
	}

	h.inLink = false
	h.tags[len(h.tags)-1].contentEnd = h.lastPos
	h.tags[len(h.tags)-1].tagEnd = h.dec.InputOffset()
}

type tagType uint8

const (
	tagScript tagType = iota
	tagImportMap
	tagStyle
	tagLink
)

type tag struct {
	tagType                                    tagType
	tagStart, tagEnd, contentStart, contentEnd int64
	src                                        string
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
