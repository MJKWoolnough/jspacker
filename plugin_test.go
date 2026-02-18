package jspacker

import (
	"fmt"
	"testing"

	"vimagination.zapto.org/javascript"
	"vimagination.zapto.org/parser"
)

func TestPlugin(t *testing.T) {
	for n, test := range [...]struct {
		Input  string
		URL    string
		Output string
	}{
		{ // 1
			"import a from './b.js';console.log(a)",
			"/a.js",
			"const a_ = await include(\"/b.js\");\n\nconsole.log(a_.default);",
		},
		{ // 2
			"import a from '../b.js';console.log(a)",
			"/a/a.js",
			"const a_ = await include(\"/b.js\");\n\nconsole.log(a_.default);",
		},
		{ // 3
			"import a, {b, c} from './b.js';console.log(a, b, c)",
			"/a.js",
			"const a_ = await include(\"/b.js\");\n\nconsole.log(a_.default, a_.b, a_.c);",
		},
		{ // 4
			"import * as a from './b.js';console.log(a)",
			"/a.js",
			"const a_ = await include(\"/b.js\");\n\nconsole.log(a_);",
		},
	} {
		tks := parser.NewStringTokeniser(test.Input)

		m, err := javascript.ParseModule(&tks)
		if err != nil {
			t.Fatalf("test %d: unexpected err: %s", n+1, err)
		}

		s, err := Plugin(m, test.URL)
		if err != nil {
			t.Fatalf("test %d: unexpected err: %s", n+1, err)
		}

		if output := fmt.Sprintf("%s", s); output != test.Output {
			t.Errorf("test %d: expecting output: %q\ngot: %q", n+1, test.Output, output)
		}
	}
}
