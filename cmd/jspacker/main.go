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
	"path"
	"path/filepath"
	"strings"

	"vimagination.zapto.org/javascript"
	"vimagination.zapto.org/jspacker"
	"vimagination.zapto.org/parser"
)

type Config struct {
	output, base, html         string
	filesTodo                  Inputs
	plugin, noExports, exports bool
	importMap                  importMap
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

type importMap map[string]string

func (i importMap) Set(v string) error {
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

func (i importMap) Import(r io.Reader) error {
	var im struct {
		Imports map[string]string
	}

	if err := json.NewDecoder(r).Decode(&im); err != nil {
		return err
	}

	maps.Copy(i, im.Imports)

	return nil
}

func (i importMap) String() string {
	b, _ := json.Marshal(i)

	return string(b)
}

func (i importMap) Resolve(from, to string) string {
	if m, ok := i[to]; ok {
		return jspacker.RelTo("/", m)
	}

	return jspacker.RelTo(from, to)
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
	config, err := parseConfig()
	if err != nil {
		return err
	}

	var s *javascript.Module

	if config.plugin {
		if s, err = readPlugin(config.base, config.filesTodo[0]); err != nil {
			return err
		}
	} else if s, err = readModuleWithOptions(config); err != nil {
		return err
	}

	for len(s.ModuleListItems) > 0 && s.ModuleListItems[0].ImportDeclaration == nil && s.ModuleListItems[0].ExportDeclaration == nil && s.ModuleListItems[0].StatementListItem == nil {
		s.ModuleListItems = s.ModuleListItems[1:]
	}

	return outputJS(config.output, s)
}

func parseConfig() (*Config, error) {
	config := &Config{importMap: make(importMap)}

	flag.Var(&config.filesTodo, "i", "input file")
	flag.StringVar(&config.output, "o", "-", "output file")
	flag.StringVar(&config.base, "b", "", "js base dir")
	flag.BoolVar(&config.plugin, "p", false, "export file as plugin")
	flag.BoolVar(&config.noExports, "n", false, "no exports")
	flag.BoolVar(&config.exports, "e", false, "keep primary file exports")
	flag.Var(config.importMap, "m", "import map used to resolve import URLs; can be specified as a JSON file or as individual KEY=VALUE pairs")
	flag.StringVar(&config.html, "H", "", "parse import map from HTML file")
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

func readPlugin(base, input string) (*javascript.Module, error) {
	f, err := os.Open(filepath.Join(base, filepath.FromSlash(input)))
	if err != nil {
		return nil, fmt.Errorf("error opening url: %w", err)
	}

	tks := parser.NewReaderTokeniser(f)

	m, err := javascript.ParseModule(&tks)

	f.Close()

	var s *javascript.Module

	if err != nil {
		return nil, fmt.Errorf("error parsing javascript module: %w", err)
	} else if s, err = jspacker.Plugin(m, input); err != nil {
		return nil, fmt.Errorf("error processing javascript plugin: %w", err)
	}

	return s, nil
}

func readModuleWithOptions(c *Config) (*javascript.Module, error) {
	s, err := jspacker.Package(c.Options()...)
	if err != nil {
		return nil, fmt.Errorf("error generating output: %w", err)
	}

	return s, nil
}

func outputJS(output string, s *javascript.Module) error {
	var (
		of  *os.File
		err error
	)

	if output == "-" {
		of = os.Stdout
	} else if of, err = os.Create(output); err != nil {
		return fmt.Errorf("error creating output file: %w", err)
	}

	if _, err = fmt.Fprintf(of, "%+s\n", s); err != nil {
		return fmt.Errorf("error writing to output: %w", err)
	} else if err = of.Close(); err != nil {
		return fmt.Errorf("error closing output: %w", err)
	}

	return nil
}

var ErrInvalidImportMapping = errors.New("invalid import mapping")
