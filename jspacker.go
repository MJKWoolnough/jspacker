// Package jspacker is a JavaScript packer for JavaScript projects.
package jspacker

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"vimagination.zapto.org/javascript"
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

// Package packages up multiple JavaScript modules into a single file, renaming
// bindings to simulate imports.
func Package(opts ...Option) (*javascript.Module, error) {
	c, err := createConfig(opts)
	if err != nil {
		return nil, err
	}

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

	if c.requireMeta {
		c.moduleItems = slices.Insert(c.moduleItems, 0, locationOrigin())
	}

	return &javascript.Module{
		ModuleListItems: c.moduleItems,
	}, nil
}

func createConfig(opts []Option) (*config, error) {
	c := &config{
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

	c.config = c

	for _, o := range opts {
		o(c)
	}

	if len(c.filesToDo) == 0 {
		return nil, ErrNoFiles
	}

	return c, nil
}
