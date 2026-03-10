package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

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
