package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"

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
		plugin, noExports, reorder bool
		err                        error
	)

	flag.Var(&filesTodo, "i", "input file")
	flag.StringVar(&output, "o", "-", "output file")
	flag.StringVar(&base, "b", "", "js base dir")
	flag.BoolVar(&plugin, "p", false, "export file as plugin")
	flag.BoolVar(&noExports, "n", false, "no exports")
	flag.BoolVar(&reorder, "x", false, "experimental script re-ordering to enable better minification (BE CAREFUL)")
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

	var s *javascript.Script

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

		for _, f := range filesTodo {
			args = append(args, jspacker.File(f))
		}

		if s, err = jspacker.Package(args...); err != nil {
			return fmt.Errorf("error generating output: %w", err)
		}
	}

	for len(s.StatementList) > 0 && s.StatementList[0].Declaration == nil && s.StatementList[0].Statement == nil {
		s.StatementList = s.StatementList[1:]
	}

	if reorder {
		sort.Stable(statementSorter(s.StatementList[1:]))
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

type statementSorter []javascript.StatementListItem

func (s statementSorter) Len() int {
	return len(s)
}

func (s statementSorter) Less(i, j int) bool {
	if scoreA, scoreB := score(&s[i]), score(&s[j]); scoreA != scoreB {
		return scoreA > scoreB
	}

	return i < j
}

func (s statementSorter) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func score(sli *javascript.StatementListItem) uint8 {
	if sli.Statement != nil {
		if sli.Statement.ExpressionStatement != nil && len(sli.Statement.ExpressionStatement.Expressions) == 1 && sli.Statement.ExpressionStatement.Expressions[0].ConditionalExpression != nil && sli.Statement.ExpressionStatement.Expressions[0].AssignmentOperator == javascript.AssignmentNone {
			if ce, ok := javascript.UnwrapConditional(sli.Statement.ExpressionStatement.Expressions[0].ConditionalExpression).(*javascript.CallExpression); ok {
				if ce.MemberExpression != nil && ce.MemberExpression.MemberExpression != nil && ce.MemberExpression.MemberExpression.PrimaryExpression != nil && ce.MemberExpression.MemberExpression.PrimaryExpression.IdentifierReference != nil && ce.MemberExpression.MemberExpression.PrimaryExpression.IdentifierReference.Data == "customElements" && ce.MemberExpression.IdentifierName != nil && ce.MemberExpression.IdentifierName.Data == "define" {
					return 4
				}
			}
		} else if sli.Statement.VariableStatement != nil {
			return 2
		}
	} else if sli.Declaration != nil {
		if sli.Declaration.ClassDeclaration != nil {
			return 6
		} else if sli.Declaration.FunctionDeclaration != nil {
			return 5
		} else if sli.Declaration.LexicalDeclaration != nil {
			if sli.Declaration.LexicalDeclaration.LetOrConst == javascript.Let {
				return 3
			}
			return 1
		}
	}

	return 0
}
