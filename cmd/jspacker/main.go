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
	var (
		output, base, html         string
		filesTodo                  Inputs
		plugin, noExports, exports bool
		importMap                  = make(importMap)
		err                        error
	)

	flag.Var(&filesTodo, "i", "input file")
	flag.StringVar(&output, "o", "-", "output file")
	flag.StringVar(&base, "b", "", "js base dir")
	flag.BoolVar(&plugin, "p", false, "export file as plugin")
	flag.BoolVar(&noExports, "n", false, "no exports")
	flag.BoolVar(&exports, "e", false, "keep primary file exports")
	flag.Var(importMap, "m", "import map used to resolve import URLs; can be specified as a JSON file or as individual KEY=VALUE pairs")
	flag.StringVar(&html, "H", "", "parse import map from HTML file")
	flag.Parse()

	if plugin && len(filesTodo) != 1 {
		return errors.New("plugin mode requires a single file")
	}

	if output == "" {
		output = "-"
	}

	if base == "" {
		if output == "-" {
			base = "./"
		} else {
			base = path.Dir(output)
		}
	}

	base, err = filepath.Abs(base)
	if err != nil {
		return fmt.Errorf("error getting absolute path for base: %w", err)
	}

	if html != "" {
		if err := readImportsFromHTML(html, importMap); err != nil {
			return err
		}
	}

	var s *javascript.Module

	if plugin {
		if s, err = readPlugin(base, filesTodo[0]); err != nil {
			return err
		}
	} else if s, err = readModuleWithOptions(filesTodo, importMap, base, noExports, exports); err != nil {
		return err
	}

	for len(s.ModuleListItems) > 0 && s.ModuleListItems[0].ImportDeclaration == nil && s.ModuleListItems[0].ExportDeclaration == nil && s.ModuleListItems[0].StatementListItem == nil {
		s.ModuleListItems = s.ModuleListItems[1:]
	}

	return outputJS(output, s)
}

func readImportsFromHTML(html string, importMap importMap) error {
	f, err := os.Open(html)
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
			if err := importMap.Import(strings.NewReader(s.Data)); err != nil {
				return err
			}
		}
	}

	return nil
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

func readModuleWithOptions(filesTodo []string, importMap importMap, base string, noExports, exports bool) (*javascript.Module, error) {
	args := make([]jspacker.Option, 1, len(filesTodo)+4)
	args[0] = jspacker.ParseDynamic

	if len(importMap) > 0 {
		args = append(args, jspacker.ResolveURL(importMap.Resolve))
	}

	if base != "" {
		args = append(args, jspacker.Loader(jspacker.OSLoad(base)))
	}

	if noExports {
		args = append(args, jspacker.NoExports)
	}

	if exports {
		args = append(args, jspacker.PrimaryExports)
	}

	for _, f := range filesTodo {
		args = append(args, jspacker.File(f))
	}

	s, err := jspacker.Package(args...)
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
