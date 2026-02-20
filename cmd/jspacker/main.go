package main

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"flag"
	"fmt"
	"io"
	"maps"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"vimagination.zapto.org/javascript"
	"vimagination.zapto.org/jspacker"
	"vimagination.zapto.org/parser"
)

type Config struct {
	output, base, html                      string
	filesTodo                               Inputs
	plugin, noExports, exports, processHTML bool
	importMap                               ImportMap
	minifier                                Minifier
}

type Inputs []string

func (i *Inputs) Set(v string) error {
	*i = append(*i, v)
	return nil
}

func (i *Inputs) String() string {
	return ""
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

type ImportMap map[string]string

func (i ImportMap) Set(v string) error {
	if v == "-" {
		return i.Import(os.Stdin)
	} else if strings.HasPrefix(v, "@") {
		k, v, ok := strings.Cut(v, "=")
		if !ok {
			return ErrInvalidImportMapping
		}

		i[k] = v
	} else {
		f, err := os.Open(v)
		if err != nil {
			return fmt.Errorf("failed to open import map: %w", err)
		}

		if err := i.Import(f); err != nil {
			return err
		}

		return f.Close()
	}

	return nil
}

func (i ImportMap) Import(r io.Reader) error {
	var im struct {
		Imports map[string]string
	}

	if err := json.NewDecoder(r).Decode(&im); err != nil {
		return err
	}

	maps.Copy(i, im.Imports)

	return nil
}

func (i ImportMap) String() string {
	b, _ := json.Marshal(i)

	return string(b)
}

func (i ImportMap) Resolve(from, to string) string {
	if m, ok := i[to]; ok {
		return jspacker.RelTo("/", m)
	}

	return jspacker.RelTo(from, to)
}

type Minifier []string

func (m *Minifier) Set(v string) error {
	return json.Unmarshal([]byte(v), m)
}

func (m *Minifier) String() string {
	b, _ := json.Marshal(m)

	return string(b)
}

type Script struct {
	Type string `xml:"type,attr"`
	Data string `xml:",chardata"`
}

type htmlPage struct {
	Head struct {
		Scripts []Script `xml:"script"`
	} `xml:"head"`
}

func run() error {
	c, err := parseConfig()
	if err != nil {
		return err
	}

	if c.processHTML {
		return c.processHTMLInput()
	}

	return c.processJavascript()
}

func parseConfig() (*Config, error) {
	config := &Config{importMap: make(ImportMap)}

	flag.Var(&config.filesTodo, "i", "input file")
	flag.StringVar(&config.output, "o", "-", "output file")
	flag.StringVar(&config.base, "b", "", "js base dir")
	flag.BoolVar(&config.plugin, "p", false, "export file as plugin")
	flag.BoolVar(&config.noExports, "n", false, "no exports")
	flag.BoolVar(&config.exports, "e", false, "keep primary file exports")
	flag.BoolVar(&config.processHTML, "P", false, "process input file as HTML, packing JavaScript sources in-place (implies -H with the input file)")
	flag.Var(config.importMap, "m", "import map used to resolve import URLs; can be specified as a JSON file or as individual KEY=VALUE pairs")
	flag.StringVar(&config.html, "H", "", "parse import map from HTML file")
	flag.Var(&config.minifier, "M", "minifier to pass code through, specified as JSON array of command words; e.g [\"terser\", \"-m\"]")
	flag.Parse()

	if config.plugin && len(config.filesTodo) != 1 {
		return nil, errors.New("plugin mode requires a single file")
	}

	if err := config.setPaths(); err != nil {
		return nil, err
	}

	if config.html != "" {
		if err := config.readImportsFromHTML(); err != nil {
			return nil, fmt.Errorf("error parsing import map from HTML: %w", err)
		}
	}

	return config, nil
}

func (c *Config) processHTMLInput() error {
	if len(c.filesTodo) != 1 {
		return ErrInvalidHTMLInput
	}

	f, err := os.Open(c.filesTodo[0])
	if err != nil {
		return err
	}

	defer f.Close()

	h := newHTMLState(f)

	for {
		if err := h.processToken(); errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return err
		}
	}

	return c.writeHTML(h)
}

func (c *Config) writeHTML(h *htmlState) (err error) {
	f, err := c.outputFile()
	if err != nil {
		return err
	}

	defer func() {
		if errr := f.Close(); err == nil {
			err = errr
		}
	}()

	if c.base == "" {
		c.base, err = os.Getwd()
		if err != nil {
			return err
		}
	}

	return c.writeHTMLContents(f, h)
}

func (c *Config) writeHTMLContents(f *os.File, h *htmlState) error {
	html := h.buf.String()

	var lastPos int64

	for _, script := range h.scripts {
		f.WriteString(html[lastPos:script.tagStart])

		if script.isMap {
			if err := c.importMap.Import(strings.NewReader(html[script.contentStart:script.contentEnd])); err != nil {
				return err
			}
		} else if err := c.processScript(f, html, script); err != nil {
			return err
		}

		lastPos = script.tagEnd
	}

	f.WriteString(html[lastPos:])

	return nil
}

func (c *Config) processScript(f *os.File, html string, script script) error {
	opts := c.Options()

	if _, err := f.WriteString(`<script type="module">`); err != nil {
		return err
	}

	if script.src == "" {
		opts = append(opts, jspacker.Loader(ScriptLoader(html[script.contentStart:script.contentEnd], c.base)))
		c.filesTodo[0] = "/\x00"
	} else {
		c.filesTodo[0] = path.Join("/", script.src)
	}

	m, err := jspacker.Package(c.Options()...)
	if err != nil {
		return fmt.Errorf("error generating output: %w", err)
	}

	if err = c.writeOutput(f, m); err != nil {
		return err
	}

	_, err = f.WriteString(`</script>`)

	return err
}

func (c *Config) processJavascript() error {
	var (
		s   *javascript.Module
		err error
	)

	if c.plugin {
		if s, err = readPlugin(c.base, c.filesTodo[0]); err != nil {
			return err
		}
	} else if s, err = c.readModuleWithOptions(); err != nil {
		return err
	}

	for len(s.ModuleListItems) > 0 && s.ModuleListItems[0].ImportDeclaration == nil && s.ModuleListItems[0].ExportDeclaration == nil && s.ModuleListItems[0].StatementListItem == nil {
		s.ModuleListItems = s.ModuleListItems[1:]
	}

	return c.outputJS(s)
}

func (c *Config) setPaths() error {
	var err error

	if c.output == "" {
		c.output = "-"
	}

	if c.base == "" {
		if c.output == "-" {
			c.base = "./"
		} else {
			c.base = path.Dir(c.output)
		}
	}

	c.base, err = filepath.Abs(c.base)
	if err != nil {
		return fmt.Errorf("error getting absolute path for base: %w", err)
	}

	return nil
}

func (c *Config) readImportsFromHTML() error {
	f, err := os.Open(c.html)
	if err != nil {
		return err
	}

	defer f.Close()

	dec := xml.NewDecoder(f)
	dec.Strict = false
	dec.AutoClose = xml.HTMLAutoClose
	dec.Entity = xml.HTMLEntity

	var h htmlPage

	if err := dec.Decode(&h); err != nil {
		return err
	}

	for _, s := range h.Head.Scripts {
		if s.Type == "importmap" {
			if err := c.importMap.Import(strings.NewReader(s.Data)); err != nil {
				return err
			}
		}
	}

	return nil
}

func (c *Config) Options() []jspacker.Option {
	options := make([]jspacker.Option, 1, len(c.filesTodo)+4)
	options[0] = jspacker.ParseDynamic

	if len(c.importMap) > 0 {
		options = append(options, jspacker.ResolveURL(c.importMap.Resolve))
	}

	if c.base != "" {
		options = append(options, jspacker.Loader(jspacker.OSLoad(c.base)))
	}

	if c.noExports {
		options = append(options, jspacker.NoExports)
	}

	if c.exports {
		options = append(options, jspacker.PrimaryExports)
	}

	for _, f := range c.filesTodo {
		options = append(options, jspacker.File(f))
	}

	return options
}

func readPlugin(base, input string) (*javascript.Module, error) {
	f, err := os.Open(filepath.Join(base, filepath.FromSlash(input)))
	if err != nil {
		return nil, fmt.Errorf("error opening url: %w", err)
	}

	defer f.Close()

	tks := parser.NewReaderTokeniser(f)

	var s *javascript.Module

	m, err := javascript.ParseModule(&tks)
	if err != nil {
		return nil, fmt.Errorf("error parsing JavaScript module: %w", err)
	} else if s, err = jspacker.Plugin(m, input); err != nil {
		return nil, fmt.Errorf("error processing JavaScript plugin: %w", err)
	}

	return s, nil
}

func (c *Config) readModuleWithOptions() (*javascript.Module, error) {
	s, err := jspacker.Package(c.Options()...)
	if err != nil {
		return nil, fmt.Errorf("error generating output: %w", err)
	}

	return s, nil
}

func (c *Config) outputFile() (*os.File, error) {
	if c.output == "-" {
		return os.Stdout, nil
	}

	f, err := os.Create(c.output)
	if err != nil {
		return nil, fmt.Errorf("error creating output file: %w", err)
	}

	return f, nil
}

func (c *Config) outputJS(s *javascript.Module) (err error) {
	f, err := c.outputFile()

	defer func() {
		if errr := f.Close(); err == nil {
			err = fmt.Errorf("error closing output: %w", errr)
		}
	}()

	return c.writeOutput(f, s)
}

func (c *Config) writeOutput(w io.Writer, m *javascript.Module) (err error) {
	if len(c.minifier) > 0 {
		pr, pw, errr := os.Pipe()
		if err != nil {
			return errr
		}

		cmd := exec.Command(c.minifier[0], c.minifier[1:]...)
		cmd.Stdin = pr
		cmd.Stdout = w

		w = pw

		if err = cmd.Start(); err != nil {
			return err
		}

		defer func() {
			if errr := cmd.Wait(); err == nil {
				err = errr
			}
		}()
		defer pw.Close()
	}

	if _, err = fmt.Fprintf(w, "%+s\n", m); err != nil {
		return fmt.Errorf("error writing to output: %w", err)
	}

	return nil
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

func ScriptLoader(src, base string) func(string) (*javascript.Module, error) {
	loader := jspacker.OSLoad(base)

	return func(file string) (*javascript.Module, error) {
		if file != "/\x00" {
			return loader(file)
		}

		tk := parser.NewStringTokeniser(src)

		return javascript.ParseModule(&tk)
	}
}

var (
	ErrInvalidImportMapping = errors.New("invalid import mapping")
	ErrInvalidHTMLInput     = errors.New("must specify single HTML input file")
)
