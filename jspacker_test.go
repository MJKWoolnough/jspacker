package jspacker

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"vimagination.zapto.org/javascript"
	"vimagination.zapto.org/parser"
)

type loader map[string]string

func (l loader) load(url string) (*javascript.Module, error) {
	d, ok := l[url]
	if !ok {
		return nil, os.ErrNotExist
	}
	tks := parser.NewStringTokeniser(d)
	return javascript.ParseModule(&tks)
}

func TestPackage(t *testing.T) {
	for n, test := range [...]struct {
		Input   loader
		Output  string
		Options []Option
	}{
		{ // 1
			loader{"/a.js": "1"},
			"const [a_] = [[]].map(props => Object.freeze(Object.defineProperties({}, Object.fromEntries(props.map(([prop, get]) => [prop, {enumerable: true, get}])))));\n\nObject.defineProperty(globalThis, include, {value: (() => {\nconst imports = new Map([[\"/a.js\", a_]]);\nreturn url => (imports.get(url) ?? import(url));\n})()});\n\nconst o = location.origin;\n\n1;",
			[]Option{File("/a.js")},
		},
		{ // 2
			loader{"/a.js": "1"},
			"const [a_] = [[]].map(props => Object.freeze(Object.defineProperties({}, Object.fromEntries(props.map(([prop, get]) => [prop, {enumerable: true, get}])))));\n\nconst o = location.origin;\n\n1;",
			[]Option{File("/a.js"), NoExports},
		},
		{ // 3
			loader{
				"/a.js": "import {c} from './b.js'; console.log(c)",
				"/b.js": "export const c = 1",
			},
			"const [a_, b_] = [[], [[\"c\", () => b_c]]].map(props => Object.freeze(Object.defineProperties({}, Object.fromEntries(props.map(([prop, get]) => [prop, {enumerable: true, get}])))));\n\nObject.defineProperty(globalThis, include, {value: (() => {\nconst imports = new Map([[\"/a.js\", a_], [\"/b.js\", b_]]);\nreturn url => (imports.get(url) ?? import(url));\n})()});\n\nconst o = location.origin;\n\nconst b_c = 1;\n\nconsole.log(b_c);",
			[]Option{File("/a.js")},
		},
		{ // 4
			loader{
				"/a.js": "import {d} from './b.js'; console.log(d)",
				"/b.js": "export {d} from './c.js'",
				"/c.js": "export const d = 1",
			},
			"const [a_, b_, c_] = [[], [[\"d\", () => c_d]], [[\"d\", () => c_d]]].map(props => Object.freeze(Object.defineProperties({}, Object.fromEntries(props.map(([prop, get]) => [prop, {enumerable: true, get}])))));\n\nObject.defineProperty(globalThis, include, {value: (() => {\nconst imports = new Map([[\"/a.js\", a_], [\"/b.js\", b_], [\"/c.js\", c_]]);\nreturn url => (imports.get(url) ?? import(url));\n})()});\n\nconst o = location.origin;\n\nconst c_d = 1;\n\nconsole.log(c_d);",
			[]Option{File("/a.js")},
		},
		{ // 5
			loader{
				"/a.js": "import {c as d} from './b.js'; console.log(d)",
				"/b.js": "export const c = 1",
			},
			"const [a_, b_] = [[], [[\"c\", () => b_c]]].map(props => Object.freeze(Object.defineProperties({}, Object.fromEntries(props.map(([prop, get]) => [prop, {enumerable: true, get}])))));\n\nObject.defineProperty(globalThis, include, {value: (() => {\nconst imports = new Map([[\"/a.js\", a_], [\"/b.js\", b_]]);\nreturn url => (imports.get(url) ?? import(url));\n})()});\n\nconst o = location.origin;\n\nconst b_c = 1;\n\nconsole.log(b_c);",
			[]Option{File("/a.js")},
		},
		{ // 6
			loader{
				"/a.js": "import {f as g} from './b.js'; console.log(g)",
				"/b.js": "export {e as f} from './c.js'",
				"/c.js": "const d = 1;export {d as e}",
			},
			"const [a_, b_, c_] = [[], [[\"f\", () => c_d]], [[\"e\", () => c_d]]].map(props => Object.freeze(Object.defineProperties({}, Object.fromEntries(props.map(([prop, get]) => [prop, {enumerable: true, get}])))));\n\nObject.defineProperty(globalThis, include, {value: (() => {\nconst imports = new Map([[\"/a.js\", a_], [\"/b.js\", b_], [\"/c.js\", c_]]);\nreturn url => (imports.get(url) ?? import(url));\n})()});\n\nconst o = location.origin;\n\nconst c_d = 1;\n\nconsole.log(c_d);",
			[]Option{File("/a.js")},
		},
		{ // 7
			loader{
				"/a.js": "import c from './b.js'; console.log(c)",
				"/b.js": "export default 1",
			},
			"const [a_, b_] = [[], [[\"default\", () => b_default]]].map(props => Object.freeze(Object.defineProperties({}, Object.fromEntries(props.map(([prop, get]) => [prop, {enumerable: true, get}])))));\n\nObject.defineProperty(globalThis, include, {value: (() => {\nconst imports = new Map([[\"/a.js\", a_], [\"/b.js\", b_]]);\nreturn url => (imports.get(url) ?? import(url));\n})()});\n\nconst o = location.origin;\n\nconst b_default = 1;\n\nconsole.log(b_default);",
			[]Option{File("/a.js")},
		},
		{ // 8
			loader{
				"/a.js": "import c from './b.js'; console.log(c)",
				"/b.js": "export {default} from './c.js'",
				"/c.js": "export default 1",
			},
			"const [a_, b_, c_] = [[], [[\"default\", () => c_default]], [[\"default\", () => c_default]]].map(props => Object.freeze(Object.defineProperties({}, Object.fromEntries(props.map(([prop, get]) => [prop, {enumerable: true, get}])))));\n\nObject.defineProperty(globalThis, include, {value: (() => {\nconst imports = new Map([[\"/a.js\", a_], [\"/b.js\", b_], [\"/c.js\", c_]]);\nreturn url => (imports.get(url) ?? import(url));\n})()});\n\nconst o = location.origin;\n\nconst c_default = 1;\n\nconsole.log(c_default);",
			[]Option{File("/a.js")},
		},
		{ // 9
			loader{
				"/a.js":   "import {d} from './b/b.js'; console.log(d)",
				"/b/b.js": "export {d} from '../c/c.js'",
				"/c/c.js": "export const d = 1",
			},
			"const [a_, b_, c_] = [[], [[\"d\", () => c_d]], [[\"d\", () => c_d]]].map(props => Object.freeze(Object.defineProperties({}, Object.fromEntries(props.map(([prop, get]) => [prop, {enumerable: true, get}])))));\n\nObject.defineProperty(globalThis, include, {value: (() => {\nconst imports = new Map([[\"/a.js\", a_], [\"/b/b.js\", b_], [\"/c/c.js\", c_]]);\nreturn url => (imports.get(url) ?? import(url));\n})()});\n\nconst o = location.origin;\n\nconst c_d = 1;\n\nconsole.log(c_d);",
			[]Option{File("/a.js")},
		},
		{ // 10
			loader{
				"/a.js":   "import * as e from './b/b.js'; console.log(e.d)",
				"/b/b.js": "export * from '../c/c.js'",
				"/c/c.js": "export const d = 1",
			},
			"const [a_, b_, c_] = [[], [[\"d\", () => c_d]], [[\"d\", () => c_d]]].map(props => Object.freeze(Object.defineProperties({}, Object.fromEntries(props.map(([prop, get]) => [prop, {enumerable: true, get}])))));\n\nObject.defineProperty(globalThis, include, {value: (() => {\nconst imports = new Map([[\"/a.js\", a_], [\"/b/b.js\", b_], [\"/c/c.js\", c_]]);\nreturn url => (imports.get(url) ?? import(url));\n})()});\n\nconst o = location.origin;\n\nconst c_d = 1;\n\nconst e = b_;\n\nconsole.log(e.d);",
			[]Option{File("/a.js")},
		},
		{ // 11
			loader{
				"/a.js":   "import * as e from './b/b.js'; export {e}",
				"/b/b.js": "export {default as B} from '../c/c.js';",
				"/c/c.js": "export default class C {};",
			},
			"const [a_, b_, c_] = [[[\"e\", () => b_]], [[\"B\", () => c_default]], [[\"default\", () => c_default]]].map(props => Object.freeze(Object.defineProperties({}, Object.fromEntries(props.map(([prop, get]) => [prop, {enumerable: true, get}])))));\n\nObject.defineProperty(globalThis, include, {value: (() => {\nconst imports = new Map([[\"/a.js\", a_], [\"/b/b.js\", b_], [\"/c/c.js\", c_]]);\nreturn url => (imports.get(url) ?? import(url));\n})()});\n\nconst o = location.origin;\n\nclass c_default {}\n\n;\n\nconst e = b_;",
			[]Option{File("/a.js")},
		},
		{ // 12
			loader{
				"/a.js":   "import {b, c} from './b/b.js'; console.log(b, c)",
				"/b/b.js": "import {c} from '../c/c.js';const b = 1;export {b, c};",
				"/c/c.js": "import {b} from '../b/b.js';const c = 2;export {b, c};",
			},
			"const [a_, b_, c_] = [[], [[\"b\", () => b_b], [\"c\", () => c_c]], [[\"b\", () => b_b], [\"c\", () => c_c]]].map(props => Object.freeze(Object.defineProperties({}, Object.fromEntries(props.map(([prop, get]) => [prop, {enumerable: true, get}])))));\n\nObject.defineProperty(globalThis, include, {value: (() => {\nconst imports = new Map([[\"/a.js\", a_], [\"/b/b.js\", b_], [\"/c/c.js\", c_]]);\nreturn url => (imports.get(url) ?? import(url));\n})()});\n\nconst o = location.origin;\n\nconst c_c = 2;\n\nconst b_b = 1;\n\nconsole.log(b_b, c_c);",
			[]Option{File("/a.js")},
		},
		{ // 13
			loader{
				"/a.js":   "import {a as ba, b as bb} from './b/b.js'; import {a as ca, b as cb} from './c/c.js'; console.log(ba, bb, ca, cb)",
				"/b/b.js": "export * from '../c/c.js';export const a = 1;",
				"/c/c.js": "export * from '../b/b.js';export const b = 2;",
			},
			"const [a_, b_, c_] = [[], [[\"a\", () => b_a], [\"b\", () => c_b]], [[\"a\", () => b_a], [\"b\", () => c_b]]].map(props => Object.freeze(Object.defineProperties({}, Object.fromEntries(props.map(([prop, get]) => [prop, {enumerable: true, get}])))));\n\nObject.defineProperty(globalThis, include, {value: (() => {\nconst imports = new Map([[\"/a.js\", a_], [\"/b/b.js\", b_], [\"/c/c.js\", c_]]);\nreturn url => (imports.get(url) ?? import(url));\n})()});\n\nconst o = location.origin;\n\nconst c_b = 2;\n\nconst b_a = 1;\n\nconsole.log(b_a, c_b, b_a, c_b);",
			[]Option{File("/a.js")},
		},
		{ // 14
			loader{
				"/a.js":   "import {a, b, c} from './b/b.js'; console.log(a, b, c)",
				"/b/b.js": "export * from '../c/c.js';export const a = 1;",
				"/c/c.js": "export const a = 2, b = 3, c = 4",
			},
			"const [a_, b_, c_] = [[], [[\"a\", () => b_a], [\"b\", () => c_b], [\"c\", () => c_c]], [[\"a\", () => c_a], [\"b\", () => c_b], [\"c\", () => c_c]]].map(props => Object.freeze(Object.defineProperties({}, Object.fromEntries(props.map(([prop, get]) => [prop, {enumerable: true, get}])))));\n\nObject.defineProperty(globalThis, include, {value: (() => {\nconst imports = new Map([[\"/a.js\", a_], [\"/b/b.js\", b_], [\"/c/c.js\", c_]]);\nreturn url => (imports.get(url) ?? import(url));\n})()});\n\nconst o = location.origin;\n\nconst c_a = 2, c_b = 3, c_c = 4;\n\nconst b_a = 1;\n\nconsole.log(b_a, c_b, c_c);",
			[]Option{File("/a.js")},
		},
		{ // 15
			loader{
				"/a.js": "import {a} from '/b.js'; console.log(a)",
				"/b.js": "export * from '/c.js';",
				"/c.js": "export let a = 1;",
			},
			"const [a_, b_, c_] = [[], [[\"a\", () => c_a]], [[\"a\", () => c_a]]].map(props => Object.freeze(Object.defineProperties({}, Object.fromEntries(props.map(([prop, get]) => [prop, {enumerable: true, get}])))));\n\nObject.defineProperty(globalThis, include, {value: (() => {\nconst imports = new Map([[\"/a.js\", a_], [\"/b.js\", b_], [\"/c.js\", c_]]);\nreturn url => (imports.get(url) ?? import(url));\n})()});\n\nconst o = location.origin;\n\nlet c_a = 1;\n\nconsole.log(c_a);",
			[]Option{File("/a.js")},
		},
		{ // 16
			loader{
				"/a.js": "import {a} from '/b.js'; console.log(a)",
				"/b.js": "export * from '/c.js';",
				"/c.js": "export var a = 1;",
			},
			"const [a_, b_, c_] = [[], [[\"a\", () => c_a]], [[\"a\", () => c_a]]].map(props => Object.freeze(Object.defineProperties({}, Object.fromEntries(props.map(([prop, get]) => [prop, {enumerable: true, get}])))));\n\nObject.defineProperty(globalThis, include, {value: (() => {\nconst imports = new Map([[\"/a.js\", a_], [\"/b.js\", b_], [\"/c.js\", c_]]);\nreturn url => (imports.get(url) ?? import(url));\n})()});\n\nconst o = location.origin;\n\nvar c_a = 1;\n\nconsole.log(c_a);",
			[]Option{File("/a.js")},
		},
		{ // 17
			loader{
				"/a.js": "import fn from './b.js'; fn()",
				"/b.js": "export default function () {}",
			},
			"const [a_, b_] = [[], [[\"default\", () => b_default]]].map(props => Object.freeze(Object.defineProperties({}, Object.fromEntries(props.map(([prop, get]) => [prop, {enumerable: true, get}])))));\n\nObject.defineProperty(globalThis, include, {value: (() => {\nconst imports = new Map([[\"/a.js\", a_], [\"/b.js\", b_]]);\nreturn url => (imports.get(url) ?? import(url));\n})()});\n\nconst o = location.origin;\n\nfunction b_default() {}\n\nb_default();",
			[]Option{File("/a.js")},
		},
		{ // 18
			loader{
				"/a.js": "import cl from './b.js'; new cl()",
				"/b.js": "export default class {}",
			},
			"const [a_, b_] = [[], [[\"default\", () => b_default]]].map(props => Object.freeze(Object.defineProperties({}, Object.fromEntries(props.map(([prop, get]) => [prop, {enumerable: true, get}])))));\n\nObject.defineProperty(globalThis, include, {value: (() => {\nconst imports = new Map([[\"/a.js\", a_], [\"/b.js\", b_]]);\nreturn url => (imports.get(url) ?? import(url));\n})()});\n\nconst o = location.origin;\n\nclass b_default {}\n\nnew b_default();",
			[]Option{File("/a.js")},
		},
		{ // 19
			loader{
				"/a.js": "import vr from './b.js'; console.log(vr)",
				"/b.js": "const b = 1; export default b",
			},
			"const [a_, b_] = [[], [[\"default\", () => b_default]]].map(props => Object.freeze(Object.defineProperties({}, Object.fromEntries(props.map(([prop, get]) => [prop, {enumerable: true, get}])))));\n\nObject.defineProperty(globalThis, include, {value: (() => {\nconst imports = new Map([[\"/a.js\", a_], [\"/b.js\", b_]]);\nreturn url => (imports.get(url) ?? import(url));\n})()});\n\nconst o = location.origin;\n\nconst b_b = 1;\n\nconst b_default = b_b;\n\nconsole.log(b_default);",
			[]Option{File("/a.js")},
		},
		{ // 20
			loader{
				"/a.js": "import vr from './b.js'; console.log(vr)",
				"/b.js": "export default class MyClass {}",
			},
			"const [a_, b_] = [[], [[\"default\", () => b_default]]].map(props => Object.freeze(Object.defineProperties({}, Object.fromEntries(props.map(([prop, get]) => [prop, {enumerable: true, get}])))));\n\nObject.defineProperty(globalThis, include, {value: (() => {\nconst imports = new Map([[\"/a.js\", a_], [\"/b.js\", b_]]);\nreturn url => (imports.get(url) ?? import(url));\n})()});\n\nconst o = location.origin;\n\nclass b_default {}\n\nconsole.log(b_default);",
			[]Option{File("/a.js")},
		},
		{ // 21
			loader{
				"/a.js": "import vr from './b.js'; console.log(vr)",
				"/b.js": "export default class MyClass {static INSTANCE = new MyClass();}",
			},
			"const [a_, b_] = [[], [[\"default\", () => b_default]]].map(props => Object.freeze(Object.defineProperties({}, Object.fromEntries(props.map(([prop, get]) => [prop, {enumerable: true, get}])))));\n\nObject.defineProperty(globalThis, include, {value: (() => {\nconst imports = new Map([[\"/a.js\", a_], [\"/b.js\", b_]]);\nreturn url => (imports.get(url) ?? import(url));\n})()});\n\nconst o = location.origin;\n\nclass b_default {\nstatic INSTANCE = new b_default();\n}\n\nconsole.log(b_default);",
			[]Option{File("/a.js")},
		},
		{ // 22
			loader{
				"/a.js": "import vr from './b.js'; console.log(vr)",
				"/b.js": "export default function aaa() {}",
			},
			"const [a_, b_] = [[], [[\"default\", () => b_default]]].map(props => Object.freeze(Object.defineProperties({}, Object.fromEntries(props.map(([prop, get]) => [prop, {enumerable: true, get}])))));\n\nObject.defineProperty(globalThis, include, {value: (() => {\nconst imports = new Map([[\"/a.js\", a_], [\"/b.js\", b_]]);\nreturn url => (imports.get(url) ?? import(url));\n})()});\n\nconst o = location.origin;\n\nfunction b_default() {}\n\nconsole.log(b_default);",
			[]Option{File("/a.js")},
		},
		{ // 23
			loader{
				"/a.js": "import vr from './b.js'; console.log(vr)",
				"/b.js": "export default function aaa() {aaa()}",
			},
			"const [a_, b_] = [[], [[\"default\", () => b_default]]].map(props => Object.freeze(Object.defineProperties({}, Object.fromEntries(props.map(([prop, get]) => [prop, {enumerable: true, get}])))));\n\nObject.defineProperty(globalThis, include, {value: (() => {\nconst imports = new Map([[\"/a.js\", a_], [\"/b.js\", b_]]);\nreturn url => (imports.get(url) ?? import(url));\n})()});\n\nconst o = location.origin;\n\nfunction b_default() {\nb_default();\n}\n\nconsole.log(b_default);",
			[]Option{File("/a.js")},
		},
	} {
		s, err := Package(append(test.Options, Loader(test.Input.load))...)
		if err != nil {
			t.Fatalf("test %d: unexpected err: %s", n+1, err)
		}
		output := strings.ReplaceAll(fmt.Sprintf("%s", s), "\t", "")
		if output != test.Output {
			t.Errorf("test %d: expecting output: %q\ngot: %q", n+1, test.Output, output)
		}
	}
}

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
		output := fmt.Sprintf("%s", s)
		if output != test.Output {
			t.Errorf("test %d: expecting output: %q\ngot: %q", n+1, test.Output, output)
		}
	}
}
