package jspacker

import (
	"fmt"
	"iter"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"vimagination.zapto.org/javascript"
	"vimagination.zapto.org/parser"
)

// Option in a type that can be passed to Package to set an option.
type Option func(*config)

// File is an Option that specifies a starting file for Package.
func File(url string) Option {
	return func(c *config) {
		c.filesToDo = append(c.filesToDo, url)
	}
}

// NoExports disables the creation of exports for any potential plugins.
func NoExports(c *config) {
	c.bare = true
}

// Loader sets the func that will take URLs and produce a parsed module.
func Loader(l func(string) (*javascript.Module, error)) Option {
	return func(c *config) {
		c.loader = l
	}
}

// ParseDynamic turns on dynamic import/include parsing.
func ParseDynamic(c *config) {
	c.parseDynamic = true
}

// PrimaryExports keeps the export statements from the passed files.
func PrimaryExports(c *config) {
	c.primary = true
}

// ResolveURL allows for custom import URL resolution.
//
// The function inputs are the URL for the current module and the import URL
// that needs resolving.
func ResolveURL(fn func(from, to string) string) Option {
	return func(c *config) {
		c.resolveURL = fn
	}
}

const (
	jsSuffix = ".js"
	tsSuffix = ".ts"
)

func loadFns(base, urlPath string, ts *bool) iter.Seq[func() (*os.File, error)] {
	return func(yield func(func() (*os.File, error)) bool) {
		for _, fn := range [...]func() (*os.File, error){
			func() (*os.File, error) { // Assume that any TS file will be more up-to-date by default
				if strings.HasSuffix(urlPath, jsSuffix) {
					*ts = true

					return os.Open(filepath.Join(base, filepath.FromSlash(urlPath[:len(urlPath)-3]+tsSuffix)))
				}

				return nil, nil
			},
			func() (*os.File, error) { // Normal
				f, err := os.Open(filepath.Join(base, filepath.FromSlash(urlPath)))
				if err == nil {
					*ts = strings.HasSuffix(urlPath, tsSuffix)
				}

				return f, err
			},
			func() (*os.File, error) { // As URL
				if u, err := url.Parse(urlPath); err == nil && u.Path != urlPath {
					f, err := os.Open(filepath.Join(base, filepath.FromSlash(u.Path)))
					if err == nil {
						*ts = strings.HasSuffix(urlPath, tsSuffix)
					}

					return f, err
				}

				return nil, nil
			},
			func() (*os.File, error) { // Add TS extension
				if !strings.HasSuffix(urlPath, tsSuffix) {
					*ts = true

					return os.Open(filepath.Join(base, filepath.FromSlash(urlPath+tsSuffix)))
				}

				return nil, nil
			},
			func() (*os.File, error) { // Add JS extension
				if !strings.HasSuffix(urlPath, jsSuffix) {
					return os.Open(filepath.Join(base, filepath.FromSlash(urlPath+jsSuffix)))
				}

				return nil, nil
			},
		} {
			if !yield(fn) {
				return
			}
		}
	}
}

// OSLoad is the default loader for Package, with the base set to CWD.
func OSLoad(base string) func(string) (*javascript.Module, error) {
	return func(urlPath string) (*javascript.Module, error) {
		var (
			f   *os.File
			err error
		)
		ts := strings.HasSuffix(base, tsSuffix)

		for loader := range loadFns(base, urlPath, &ts) {
			fb, errr := loader()
			if fb != nil {
				f = fb

				break
			} else if err == nil {
				err = errr
			}
		}

		if f == nil {
			return nil, fmt.Errorf("error opening file (%s): %w", urlPath, err)
		}

		defer f.Close()

		rt := parser.NewReaderTokeniser(f)

		var tks javascript.Tokeniser = &rt

		if ts {
			tks = javascript.AsTypescript(&rt)
		}

		m, err := javascript.ParseModule(tks)
		if err != nil {
			return nil, fmt.Errorf("error parsing file (%s): %w", urlPath, err)
		}

		return m, nil
	}
}
