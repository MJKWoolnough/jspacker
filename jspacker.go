// Package jspacker is a JavaScript packer for JavaScript projects.
package jspacker

import (
	"fmt"
	"iter"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"vimagination.zapto.org/javascript"
	"vimagination.zapto.org/javascript/scope"
	"vimagination.zapto.org/javascript/walk"
	"vimagination.zapto.org/parser"
)

type config struct {
	filesToDo     []string
	filesDone     map[string]*dependency
	resolveURL    func(string, string) string
	loader        func(string) (*javascript.Module, error)
	bare          bool
	parseDynamic  bool
	primary       bool
	nextID        uint
	exportAllFrom [][2]*dependency
	moduleItems   []javascript.ModuleItem
	dependency
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

// Package packages up multiple JavaScript modules into a single file, renaming
// bindings to simulate imports.
func Package(opts ...Option) (*javascript.Module, error) {
	c := config{
		filesDone: make(map[string]*dependency),
		dependency: dependency{
			requires: make(map[string]*dependency),
		},
		resolveURL: RelTo,
	}

	if c.loader == nil {
		base, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("error getting current working directory: %w", err)
		}

		c.loader = OSLoad(base)
	}

	c.config = &c

	for _, o := range opts {
		o(&c)
	}

	if len(c.filesToDo) == 0 {
		return nil, ErrNoFiles
	}

	c.moduleItems = slices.Insert(c.moduleItems, 0, locationOrigin())

	for _, url := range c.filesToDo {
		if !strings.HasPrefix(url, "/") {
			return nil, fmt.Errorf("%w: %s", ErrInvalidURL, url)
		} else if _, err := c.dependency.addImport(c.dependency.RelTo(url), c.primary); err != nil {
			return nil, err
		}
	}

	for changed := true; changed; {
		changed = false

		for _, eaf := range c.exportAllFrom {
			for export := range eaf[1].exports {
				if export == "default" {
					continue
				} else if _, ok := eaf[0].exports[export]; !ok {
					eaf[0].exports[export] = &importBinding{
						dependency: eaf[1],
						binding:    export,
					}
					changed = true
				}
			}
		}
	}

	if err := c.dependency.resolveImports(); err != nil {
		return nil, err
	} else if err := c.makeLoader(); err != nil {
		return nil, err
	}

	return &javascript.Module{
		ModuleListItems: c.moduleItems,
	}, nil
}

// Plugin converts a single JavaScript module to make use of the processed
// exports from package.
func Plugin(m *javascript.Module, url string) (*javascript.Module, error) {
	if !strings.HasPrefix(url, "/") {
		return nil, ErrInvalidURL
	}

	var (
		imports              = uint(0)
		importURLs           = make(map[string]string)
		importBindings       = make(importBindingMap)
		importObjectBindings []javascript.BindingElement
		importURLsArrayE     []javascript.ArrayElement
		importURLsArray      []javascript.Argument
		moduleItems          = make([]javascript.ModuleItem, 1, len(m.ModuleListItems))
		d                    = dependency{
			config: &config{
				resolveURL: RelTo,
			},
			url:    url,
			prefix: "_",
		}
	)

	scope, err := scope.ModuleScope(m, nil)
	if err != nil {
		return nil, err
	}

	for _, li := range m.ModuleListItems {
		if li.ImportDeclaration != nil {
			id := li.ImportDeclaration
			durl, _ := javascript.Unquote(id.ModuleSpecifier.Data)
			iurl := d.RelTo(durl)

			ib, ok := importURLs[iurl]
			if !ok {
				imports++

				ib = id2String(imports)
				importURLs[iurl] = ib
				ae := wrapArgument(iurl)
				importURLsArray = append(importURLsArray, ae)
				importURLsArrayE = append(importURLsArrayE, javascript.ArrayElement{
					AssignmentExpression: ae.AssignmentExpression,
				})
				importObjectBindings = append(importObjectBindings, javascript.BindingElement{
					SingleNameBinding: jToken(ib),
				})
			}

			if id.ImportClause != nil {
				if id.NameSpaceImport != nil {
					for _, binding := range scope.Bindings[li.ImportDeclaration.NameSpaceImport.Data] {
						binding.Data = ib
					}
				}

				if id.ImportedDefaultBinding != nil {
					importBindings[id.ImportedDefaultBinding.Data] = wrapMemberIdentifier(ib, jToken("default"))
				}

				if id.NamedImports != nil {
					for _, is := range id.NamedImports.ImportList {
						tk := is.ImportedBinding

						if is.IdentifierName != nil {
							tk = is.IdentifierName
						}

						importBindings[is.ImportedBinding.Data] = wrapMemberIdentifier(ib, tk)
					}
				}
			}
		} else if li.StatementListItem != nil {
			moduleItems = append(moduleItems, li)
		} else if li.ExportDeclaration != nil {
			ed := li.ExportDeclaration
			if ed.VariableStatement != nil {
				moduleItems = append(moduleItems, wrapVariableStatement(ed.VariableStatement))
			} else if ed.Declaration != nil {
				moduleItems = append(moduleItems, wrapDeclaration(ed.Declaration))
			} else if ed.DefaultFunction != nil {
				if ed.DefaultFunction.BindingIdentifier != nil {
					moduleItems = append(moduleItems, wrapFunctionDeclaration(ed.DefaultFunction))
				}
			} else if ed.DefaultClass != nil {
				if ed.DefaultClass.BindingIdentifier != nil {
					moduleItems = append(moduleItems, wrapClassDeclaration(ed.DefaultClass))
				}
			} else if ed.DefaultAssignmentExpression != nil {
				moduleItems = append(moduleItems, wrapAssignmentExpression(*ed.DefaultAssignmentExpression))
			}
		}
	}

	d.processBindings(scope)

	if imports == 0 {
		moduleItems = moduleItems[1:]
	} else if imports == 1 {
		moduleItems[0] = wrapIncludeCall(importObjectBindings[0].SingleNameBinding, importURLsArray)
	} else {
		moduleItems[0] = wrapIncludeAllCall(importObjectBindings, importURLsArrayE)
	}

	s := &javascript.Module{
		ModuleListItems: moduleItems,
	}

	walk.Walk(s, &d)
	walk.Walk(s, importBindings)

	return s, nil
}

type importBindingMap map[string]javascript.MemberExpression

func (i importBindingMap) Handle(t javascript.Type) error {
	if me, ok := t.(*javascript.MemberExpression); ok && me.PrimaryExpression != nil && me.PrimaryExpression.IdentifierReference != nil {
		if nme, ok := i[me.PrimaryExpression.IdentifierReference.Data]; ok {
			*me = nme

			return nil
		}
	}

	return walk.Walk(t, i)
}
