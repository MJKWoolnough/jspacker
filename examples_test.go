package jspacker_test

import (
	"fmt"
	"io/fs"

	"vimagination.zapto.org/javascript"
	"vimagination.zapto.org/jspacker"
	"vimagination.zapto.org/parser"
)

func Example() {
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
