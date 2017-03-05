# Flag

Flag is a command-line flag parsing library for [Go](https://golang.org), support slice, structure, select, default value, env variables parsing.

# Documentation
Documentation can be found at [Godoc](https://godoc.org/github.com/cosiner/flag)

# Example
#### Code
```Go
import (
	"testing"

	"github.com/cosiner/argv"
	"github.com/cosiner/flag"
)

func TestFlag(t *testing.T) {
	argv, err := argv.Argv([]rune("./flag  -h "), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	args := argv[0]

	type Flags struct {
		A   bool     `desc:"A is flag a" default:"true"`
		B   bool     `args:"B"`
		E   string   `desc:"E"`
		F   []string `desc:"F\naa" default:"2" selects:"2,3,4"`
		Foo struct {
			Enable bool
			FFoo   struct {
				Enable bool
			} `args:"FOO FOO" usage:"foo command" collapse:"true"`
			BVarz struct {
				Enable bool
			} `args:"BARZ" usage:"barz command" collapse:"true" desc:"aaabbbbbbbbbbbbb\nbbbbbbbbbbccc\nddd"`
		} `version:"v1.0.1" args:"FOO FOO" usage:"foo command" collapse:"true"`
		Barz struct {
			Enable bool

			C []int    `desc:"tag list" desc:"C" default:"3,4,5" selects:"3,4,5"`
			D []string `desc:"D" desc:"tag" default:"6,7,8"`
		} `args:"BARZ" usage:"barz command" collapse:"true" desc:"aaabbbbbbbbbbbbb\nbbbbbbbbbbccc\nddd"`
	}

	var fs Flags
	flag.NewFlagSet(flag.Flag{
		Names: args[0],
		Version: `
	version: v1.0.0
	commit: 10adf10dc10
	date:   2017-01-01 10:00:01
		`,
		Usage: "Flag is a flag library.",
		Desc: `
		Flag is a simple flag library for Go.
		It support slice, default value, and
		selects.
		`,
	}).ParseStruct(&fs, args...)
}
```
##### Output
```
Flag is a flag library.

Usage:
      ./flag [FLAG|SET]...

Version:
      version: v1.0.0
      commit: 10adf10dc10
      date:   2017-01-01 10:00:01

Description:
      Flag is a simple flag library for Go.
      It support slice, default value, and
      selects.

Flags:
      -a            (bool; default: true)
            A is flag a
      -b            (bool)
      -e            (string)
            E
      -f            ([]string; default: [2]; selects: [2 3 4])
            F
            aa
      -h, --help    show help (bool)
      -v, --verbose show verbose help (bool)

Sets:
      foo [SET]...   foo command
      barz [FLAG]... barz command

```

# LICENSE
MIT.
