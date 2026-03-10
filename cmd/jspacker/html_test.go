package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProcessHTMLInput(t *testing.T) {
	for n, test := range [...]struct {
		input  map[string]string
		output string
	}{
		{
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
		{
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
			noExports: true,
		}

		var buf strings.Builder

		if h, err := c.processHTMLInput(); err != nil {
			t.Errorf("test %d: unexpected error: %v", n+1, err)
		} else if err := c.writeHTML(&buf, h); err != nil {
			t.Errorf("test %d: unexpected error: %v", n+1, err)
		} else if str := buf.String(); str != test.output {
			t.Errorf("test %d: expecting output %q, got %q", n+1, test.output, str)
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
