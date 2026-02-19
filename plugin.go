package jspacker

import (
	"strings"

	"vimagination.zapto.org/javascript"
	"vimagination.zapto.org/javascript/scope"
	"vimagination.zapto.org/javascript/walk"
)

type plugin struct {
	imports              uint
	importURLs           map[string]string
	importBindings       importBindingMap
	importObjectBindings []javascript.BindingElement
	importURLsArrayE     []javascript.ArrayElement
	importURLsArray      []javascript.Argument
	javascript.Module
	d dependency
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

// Plugin converts a single JavaScript module to make use of the processed
// exports from package.
func Plugin(m *javascript.Module, url string) (*javascript.Module, error) {
	if !strings.HasPrefix(url, "/") {
		return nil, ErrInvalidURL
	}

	p := plugin{
		importURLs:     make(map[string]string),
		importBindings: make(importBindingMap),
		Module: javascript.Module{
			ModuleListItems: make([]javascript.ModuleItem, 1, len(m.ModuleListItems)),
		},
		d: dependency{
			config: &config{
				resolveURL: RelTo,
			},
			url:    url,
			prefix: "_",
		},
	}

	scope, err := scope.ModuleScope(m, nil)
	if err != nil {
		return nil, err
	}

	p.process(m, scope)
	p.d.processBindings(scope)
	p.addIncludes()
	walk.Walk(&p.Module, &p.d)
	walk.Walk(&p.Module, p.importBindings)

	return &p.Module, nil
}

func (p *plugin) process(m *javascript.Module, scope *scope.Scope) {
	for _, li := range m.ModuleListItems {
		if li.ImportDeclaration != nil {
			p.processImport(li.ImportDeclaration, scope)
		} else if li.StatementListItem != nil {
			p.ModuleListItems = append(p.ModuleListItems, li)
		} else if li.ExportDeclaration != nil {
			p.processExport(li.ExportDeclaration)
		}
	}
}

func (p *plugin) processImport(id *javascript.ImportDeclaration, scope *scope.Scope) {
	durl, _ := javascript.Unquote(id.ModuleSpecifier.Data)
	iurl := p.d.RelTo(durl)

	ib, ok := p.importURLs[iurl]
	if !ok {
		p.imports++

		ib = id2String(p.imports)
		p.importURLs[iurl] = ib
		ae := wrapArgument(iurl)
		p.importURLsArray = append(p.importURLsArray, ae)
		p.importURLsArrayE = append(p.importURLsArrayE, javascript.ArrayElement{
			AssignmentExpression: ae.AssignmentExpression,
		})
		p.importObjectBindings = append(p.importObjectBindings, javascript.BindingElement{
			SingleNameBinding: jToken(ib),
		})
	}

	if id.ImportClause != nil {
		if id.NameSpaceImport != nil {
			for _, binding := range scope.Bindings[id.NameSpaceImport.Data] {
				binding.Data = ib
			}
		}

		if id.ImportedDefaultBinding != nil {
			p.importBindings[id.ImportedDefaultBinding.Data] = wrapMemberIdentifier(ib, jToken("default"))
		}

		if id.NamedImports != nil {
			for _, is := range id.NamedImports.ImportList {
				tk := is.ImportedBinding

				if is.IdentifierName != nil {
					tk = is.IdentifierName
				}

				p.importBindings[is.ImportedBinding.Data] = wrapMemberIdentifier(ib, tk)
			}
		}
	}
}

func (p *plugin) processExport(ed *javascript.ExportDeclaration) {
	if ed.VariableStatement != nil {
		p.ModuleListItems = append(p.ModuleListItems, wrapVariableStatement(ed.VariableStatement))
	} else if ed.Declaration != nil {
		p.ModuleListItems = append(p.ModuleListItems, wrapDeclaration(ed.Declaration))
	} else if ed.DefaultFunction != nil {
		if ed.DefaultFunction.BindingIdentifier != nil {
			p.ModuleListItems = append(p.ModuleListItems, wrapFunctionDeclaration(ed.DefaultFunction))
		}
	} else if ed.DefaultClass != nil {
		if ed.DefaultClass.BindingIdentifier != nil {
			p.ModuleListItems = append(p.ModuleListItems, wrapClassDeclaration(ed.DefaultClass))
		}
	} else if ed.DefaultAssignmentExpression != nil {
		p.ModuleListItems = append(p.ModuleListItems, wrapAssignmentExpression(*ed.DefaultAssignmentExpression))
	}
}

func (p *plugin) addIncludes() {
	if p.imports == 0 {
		p.ModuleListItems = p.ModuleListItems[1:]
	} else if p.imports == 1 {
		p.ModuleListItems[0] = wrapIncludeCall(p.importObjectBindings[0].SingleNameBinding, p.importURLsArray)
	} else {
		p.ModuleListItems[0] = wrapIncludeAllCall(p.importObjectBindings, p.importURLsArrayE)
	}
}
