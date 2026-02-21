package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"vimagination.zapto.org/javascript"
	"vimagination.zapto.org/jspacker"
	"vimagination.zapto.org/parser"
)

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
