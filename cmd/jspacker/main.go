package main

import (
	"compress/gzip"
	"encoding/json"
	"encoding/xml"
	"errors"
	"flag"
	"fmt"
	"io"
	"maps"
	"os"
	"path"
	"path/filepath"
	"strings"

	"vimagination.zapto.org/jspacker"
)

type Config struct {
	output, base, html                                                string
	filesTodo                                                         Inputs
	plugin, noExports, exports, processHTMLFile, processCSS, compress bool
	importMap                                                         ImportMap
	minifier                                                          Minifier
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

	if c.processHTMLFile {
		return c.processHTML()
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
	flag.BoolVar(&config.processHTMLFile, "P", false, "process input file as HTML, packing JavaScript sources in-place (implies -H with the input file)")
	flag.BoolVar(&config.processCSS, "c", false, "embed linked CSS in HTML file")
	flag.Var(config.importMap, "m", "import map used to resolve import URLs; can be specified as a JSON file or as individual KEY=VALUE pairs")
	flag.StringVar(&config.html, "H", "", "parse import map from HTML file")
	flag.Var(&config.minifier, "M", "minifier to pass code through, specified as JSON array of command words; e.g [\"terser\", \"-m\"]")
	flag.BoolVar(&config.compress, "z", false, "gzip compress output")
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

func (c *Config) outputFile() (io.WriteCloser, error) {
	if c.output == "-" {
		return os.Stdout, nil
	}

	f, err := os.Create(c.output)
	if err != nil {
		return nil, fmt.Errorf("error creating output file: %w", err)
	}

	if !c.compress {
		return f, nil
	}

	g, err := gzip.NewWriterLevel(f, gzip.BestCompression)
	if err != nil {
		return nil, err
	}

	return &compressedFile{
		Writer: g,
		file:   f,
	}, nil
}

type compressedFile struct {
	*gzip.Writer
	file *os.File
}

func (c *compressedFile) Close() (err error) {
	defer func() {
		if errr := c.file.Close(); err == nil {
			err = errr
		}
	}()

	return c.Writer.Close()
}

var ErrInvalidImportMapping = errors.New("invalid import mapping")
