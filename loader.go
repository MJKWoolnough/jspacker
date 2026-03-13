package jspacker

import (
	"cmp"
	"fmt"
	"iter"
	"maps"
	"slices"

	"vimagination.zapto.org/javascript"
	"vimagination.zapto.org/parser"
)

func jToken(data string) *javascript.Token {
	return &javascript.Token{Token: parser.Token{Data: data}}
}

func (c *config) makeLoader() error {
	obs, err := c.processFiles()
	if err != nil {
		return err
	}

	if len(obs) == 0 {
		return nil
	}

	c.moduleItems = slices.Insert(c.moduleItems, 0, wrapConst(obs))

	if c.bare && (!c.parseDynamic || !c.dynamicRequirement) {
		return nil
	}

	imports := make([]javascript.ArrayElement, 0, len(c.filesDone))

	for url, file := range sortedMap(c.filesDone) {
		imports = append(imports, wrapURLNameSpace(url, file.prefix))
	}

	c.moduleItems = slices.Insert(c.moduleItems, 1, wrapImports(imports))

	return nil
}

func (c *config) processFiles() ([]javascript.LexicalBinding, error) {
	obs := make([]javascript.LexicalBinding, 0, len(c.filesDone))

	for _, file := range sortedMap(c.filesDone) {
		if !file.requireNamespace && c.bare && !c.parseDynamic && !c.dynamicRequirement {
			continue
		}

		fields := make([]javascript.PropertyDefinition, 0, len(file.exports))

		for binding := range sortedMap(file.exports) {
			b := file.resolveExport(binding)

			if b == nil {
				return nil, fmt.Errorf("error resolving export %s (%s): %w", binding, file.url, ErrInvalidExport)
			}

			fields = append(fields, makeGetter(binding, b.Token))
		}

		obs = append(obs, wrapNameSpaceFields(file.prefix, fields))
	}

	return obs, nil
}

func sortedMap[K cmp.Ordered, V any](m map[K]V) iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		keys := slices.Collect(maps.Keys(m))

		slices.Sort(keys)

		for _, k := range keys {
			if !yield(k, m[k]) {
				return
			}
		}
	}
}
