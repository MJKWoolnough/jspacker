package jspacker

import (
	"strings"

	"vimagination.zapto.org/javascript"
	"vimagination.zapto.org/javascript/scope"
	"vimagination.zapto.org/javascript/walk"
)

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
