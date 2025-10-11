# jspacker

[![CI](https://github.com/MJKWoolnough/jspacker/actions/workflows/go-checks.yml/badge.svg)](https://github.com/MJKWoolnough/jspacker/actions)
[![Go Reference](https://pkg.go.dev/badge/vimagination.zapto.org/jspacker.svg)](https://pkg.go.dev/vimagination.zapto.org/jspacker)
[![Go Report Card](https://goreportcard.com/badge/vimagination.zapto.org/jspacker)](https://goreportcard.com/report/vimagination.zapto.org/jspacker)

--
    import "vimagination.zapto.org/jspacker"

Package jspacker is a javascript packer for javascript projects.

## Highlights

 - Combine multiple Javascript/Typescript modules into a single file.
 - Optional ability to allow dynamic imports.
 - Can create separate plugin scripts that can import from primary script.

## Usage

```go
package main

import (
	"fmt"
	"io/fs"

	"vimagination.zapto.org/javascript"
	"vimagination.zapto.org/jspacker"
	"vimagination.zapto.org/parser"
)

func main() {
	files := map[string]string{
		"/main.js":      `import fn from './lib/utils.js'; const v = 2; console.log(v + fn())`,
		"/lib/utils.js": "export default () => 1;",
	}
	loader := func(file string) (*javascript.Module, error) {
		src, ok := files[file]
		if !ok {
			return nil, fs.ErrNotExist
		}

		tk := parser.NewStringTokeniser(src)

		return javascript.ParseModule(&tk)
	}

	script, err := jspacker.Package(jspacker.File("/main.js"), jspacker.NoExports, jspacker.Loader(loader))
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Printf("%s", script)
	}

	// Output:
	// const b_default = () => 1;
	//
	// const a_v = 2;
	//
	// console.log(a_v + b_default());
}
```

## Command

Includes command `vimagination.zapto.org/jspacker/cmd/jspacker` to combine multiple Javascript or Typescript files:

The `jspacker` command accepts the following flags

```
  -b string     js base dir
  -e            keep primary file exports
  -i value      input file
  -n            no exports
  -o string     output file (default "-")
  -p            export file as plugin
```

### Command Example

The following command will bundle `main.ts` and all of its imports into a single file, `combined.js`.

```bash
jspacker -i /main.ts -n -o combined.js
```

## Documentation

Full API docs can be found at:

https://pkg.go.dev/vimagination.zapto.org/jspacker
