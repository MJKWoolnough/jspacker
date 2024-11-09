package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path"
	"path/filepath"

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

func run() error {
	var (
		output, base               string
		filesTodo                  Inputs
		plugin, noExports, exports bool
		err                        error
	)

	flag.Var(&filesTodo, "i", "input file")
	flag.StringVar(&output, "o", "-", "output file")
	flag.StringVar(&base, "b", "", "js base dir")
	flag.BoolVar(&plugin, "p", false, "export file as plugin")
	flag.BoolVar(&noExports, "n", false, "no exports")
	flag.BoolVar(&exports, "e", false, "keep primary file exports")
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

	var s *javascript.Module

	if plugin {
		f, err := os.Open(filepath.Join(base, filepath.FromSlash(filesTodo[0])))
		if err != nil {
			return fmt.Errorf("error opening url: %w", err)
		}

		tks := parser.NewReaderTokeniser(f)

		m, err := javascript.ParseModule(&tks)

		f.Close()

		if err != nil {
			return fmt.Errorf("error parsing javascript module: %w", err)
		} else if s, err = jspacker.Plugin(m, filesTodo[0]); err != nil {
			return fmt.Errorf("error processing javascript plugin: %w", err)
		}
	} else {
		args := make([]jspacker.Option, 1, len(filesTodo)+3)
		args[0] = jspacker.ParseDynamic

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

		if s, err = jspacker.Package(args...); err != nil {
			return fmt.Errorf("error generating output: %w", err)
		}
	}

	for len(s.ModuleListItems) > 0 && s.ModuleListItems[0].ImportDeclaration == nil && s.ModuleListItems[0].ExportDeclaration == nil && s.ModuleListItems[0].StatementListItem == nil {
		s.ModuleListItems = s.ModuleListItems[1:]
	}

	var of *os.File

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
