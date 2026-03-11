package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProcessHTMLInput(t *testing.T) {
	for n, test := range [...]struct {
		input         map[string]string
		output        string
		inErr, outErr error
	}{
		{ // 1
			input: map[string]string{
				"index.html": `<html>
	<head>
		<title>Test</title>
	</head>
</html>`,
			},
			output: `<html>
	<head>
		<title>Test</title>
	</head>
</html>`,
		},
		{ // 2
			input: map[string]string{
				"index.html": `<html>
	<head>
		<title>Test</title>
		<script type="module">import './a.js';</script>
	</head>
</html>`,
				"a.js": "a;",
			},
			output: `<html>
	<head>
		<title>Test</title>
		<script type="module">const a_ = {},
b_ = {};

Object.defineProperty(globalThis, include, {value: (() => {
		const imports = new Map([["/\x00", a_], ["/a.js", b_]]);
		return url => (imports.get(url) ?? import(url));
	})()});

a;
</script>
	</head>
</html>`,
		},
		{ // 3
			input: map[string]string{
				"index.html": `<html>
	<head>
		<title>Test</title>
		<script type="module" src="a.js"></script>
	</head>
</html>`,
				"a.js": "a;",
			},
			output: `<html>
	<head>
		<title>Test</title>
		<script type="module">const a_ = {};

Object.defineProperty(globalThis, include, {value: (() => {
		const imports = new Map([["/a.js", a_]]);
		return url => (imports.get(url) ?? import(url));
	})()});

a;
</script>
	</head>
</html>`,
		},
		{ // 4
			input: map[string]string{
				"index.html": `<html>
	<head>
		<title>Test</title>
		<script type="module" src="@abc"></script>
	</head>
</html>`,
				"a.js": "a;",
			},
			outErr: fs.ErrNotExist,
		},
		{ // 5
			input: map[string]string{
				"index.html": `<html>
	<head>
		<title>Test</title>
		<script type="importmap">{"imports":{"@abc":"a.js"}}</script>
		<script type="module" src="@abc"></script>
	</head>
</html>`,
				"a.js": "a;",
			},
			output: `<html>
	<head>
		<title>Test</title>
		
		<script type="module">const a_ = {};

Object.defineProperty(globalThis, include, {value: (() => {
		const imports = new Map([["/a.js", a_]]);
		return url => (imports.get(url) ?? import(url));
	})()});

a;
</script>
	</head>
</html>`,
		},
	} {
		tmp := t.TempDir()

		for file, data := range test.input {
			if err := os.WriteFile(filepath.Join(tmp, file), []byte(data), 0600); err != nil {
				t.Fatalf("test %d: unexpected error: %v", n+1, err)
			}
		}

		c := Config{
			filesTodo: []string{"/index.html"},
			base:      tmp,
			importMap: make(ImportMap),
			noExports: true,
		}

		var buf strings.Builder

		if h, err := c.processHTMLInput(); !errors.Is(err, test.inErr) {
			t.Errorf("test %d.1: expecting error %v, got %v", n+1, test.inErr, err)
		} else if err = c.writeHTML(&buf, h); test.inErr == nil && !errors.Is(err, test.outErr) {
			t.Errorf("test %d.2: expecting error %v, got %v", n+1, test.outErr, err)
		} else if str := buf.String(); test.inErr == nil && test.outErr == nil && str != test.output {
			t.Errorf("test %d.3: expecting output %q, got %q", n+1, test.output, str)
		}
	}
}

func TestScriptLoader(t *testing.T) {
	tmp := t.TempDir()
	fn := scriptLoader("b;", tmp)

	if err := os.WriteFile(filepath.Join(tmp, "a.js"), []byte("a;"), 0600); err != nil {
		t.Fatalf("unexpected error: %v", err)
	} else if m, err := fn("a.js"); err != nil {
		t.Errorf("test 1: unexpected error: %v", err)
	} else if str := fmt.Sprintf("%+s", m); str != "a;" {
		t.Errorf("test 2: expecting string %q, got %q", "a;", str)
	} else if m, err = fn("/\x00"); err != nil {
		t.Errorf("test 3: unexpected error: %v", err)
	} else if str = fmt.Sprintf("%+s", m); str != "b;" {
		t.Errorf("test 4: expecting string %q, got %q", "b;", str)
	}
}
