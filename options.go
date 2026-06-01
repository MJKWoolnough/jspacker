package jspacker

import (
	"fmt"
	"iter"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"vimagination.zapto.org/javascript"
	"vimagination.zapto.org/javascript/jsx"
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

type loadOpts struct {
	disableTS bool
	jsx       *template.Template
}

// LoadOpt represents an option for the OSLoad Option.
type LoadOpt func(*loadOpts)

// DisableTS disables Typescript loading in the OSLoad Option.
func DisableTS(l *loadOpts) {
	l.disableTS = true
}

// EnableJSX enables JSX/TSX parsing and processing in the OSLoad Option.
//
// The supplied JSX template should follow the rules specified in the jsx
// package docs:
//
//	https://pkg.go.dev/vimagination.zapto.org/javascript/jsx#Process
func EnableJSX(jsx *template.Template) LoadOpt {
	return func(l *loadOpts) {
		l.jsx = jsx
	}
}

const (
	jsSuffix  = ".js"
	tsSuffix  = ".ts"
	jsxSuffix = ".jsx"
	tsxSuffix = ".tsx"
)

func loadFns(base, urlPath string, allowTS, allowJSX bool, ts, jsx *bool) iter.Seq[func() (*os.File, error)] {
	return func(yield func(func() (*os.File, error)) bool) {
		for _, fn := range [...]func() (*os.File, error){
			func() (*os.File, error) { // Assume that any TSX file will be more up-to-date by default
				if allowTS && allowJSX && strings.HasSuffix(urlPath, jsSuffix) {
					*ts = true
					*jsx = true

					return os.Open(filepath.Join(base, filepath.FromSlash(urlPath[:len(urlPath)-3]+tsxSuffix)))
				}

				return nil, nil
			},
			func() (*os.File, error) { // Assume that any JSX file will be more up-to-date by default
				if allowJSX && strings.HasSuffix(urlPath, jsSuffix) {
					*jsx = true

					return os.Open(filepath.Join(base, filepath.FromSlash(urlPath[:len(urlPath)-3]+jsxSuffix)))
				}

				return nil, nil
			},
			func() (*os.File, error) { // Assume that any TS file will be more up-to-date by default
				if allowTS && strings.HasSuffix(urlPath, jsSuffix) {
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
			func() (*os.File, error) { // Add TSX extension
				if allowTS && allowJSX && !strings.HasSuffix(urlPath, tsSuffix) {
					*ts = true
					*jsx = true

					return os.Open(filepath.Join(base, filepath.FromSlash(urlPath+tsxSuffix)))
				}

				return nil, nil
			},
			func() (*os.File, error) { // Add JSX extension
				if allowJSX && !strings.HasSuffix(urlPath, tsSuffix) {
					*jsx = true

					return os.Open(filepath.Join(base, filepath.FromSlash(urlPath+jsxSuffix)))
				}

				return nil, nil
			},
			func() (*os.File, error) { // Add TS extension
				if allowTS && !strings.HasSuffix(urlPath, tsSuffix) {
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
//
// By default, it will prefer Typescript files over JavaScript files, replacing
// a '.js' extension, if it exists, or with '.ts', or adding it if it does not.
// This behaviour can be disabled by providing the DisableTS option.
//
// JSX support can be added by providing the EnableJSX support with a valid
// template.
func OSLoad(base string, opts ...LoadOpt) func(string) (*javascript.Module, error) {
	var l loadOpts

	for _, opt := range opts {
		opt(&l)
	}

	return func(urlPath string) (*javascript.Module, error) {
		var (
			f   *os.File
			err error
		)
		isTSX := strings.HasSuffix(base, tsxSuffix)
		isTS := isTSX || strings.HasSuffix(base, tsSuffix)
		isJSX := isTSX || strings.HasSuffix(base, jsxSuffix)

		for loader := range loadFns(base, urlPath, !l.disableTS, l.jsx != nil, &isTS, &isJSX) {
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

		if isTS {
			tks = javascript.AsTypescript(tks)
		}

		if isJSX {
			tks = javascript.AsJSX(tks)
		}

		m, err := javascript.ParseModule(tks)
		if err != nil {
			return nil, fmt.Errorf("error parsing file (%s): %w", urlPath, err)
		}

		if isJSX {
			if err = jsx.Process(m, l.jsx); err != nil {
				return nil, fmt.Errorf("error processing file (%s) as JSX: %w", urlPath, err)
			}
		}

		return m, nil
	}
}
