# Flag

Flag is a simple but powerful commandline flag parsing library for [Go](https://golang.org).

# Documentation
Documentation can be found at [Godoc](https://godoc.org/github.com/cosiner/flag)


# Features
* Slice
* Default value
* Selectable values
* Environment variable parsing
* Infinite sub command levels

# Supported types
* bool
* string
* integer, unsigned integer and float (int8, int16, int32, int64, int, uint..., float32, float64)
* slice of above types

# Argument splitting rules
* `-` identify that the next argument must be a flag. It's a value if it's the last argument. 
   E.g., `['./cmd', '-', '-output', 'a.out']`.
* `--` identify that the next argument must be a value. It's a value if it's the last argument. 
  E.g., `['./cmd', '-files', '--', '-a=b', '--', '-b.md']`.
* Flags contains '='(except as the begin or end) will be separated into a flag and a value. 
  E.g., `['./cmd', '-a=b']`.
* Flags begin with '--' must be a flag. 
  E.g., `['./cmd', '--input', 'file']`.
* Flags begin with '-' must be a flag, also it can be split to multiple flags.
  - If each character is flag, it will be split completely. 
    E.g., `['tar', '-zcf', 'a.tgz', '-input', 'a']` 
    will be split as `['tar', '-z', '-c', '-f', 'a.tgz', '-input', 'a']`.   
  - If first character is a flag, it will be split into two part.
    E.g., `['-I/usr/include']` will be split as `['-I', '/usr/include']`.
  - Otherwise, it was recognized as a whole flag. 
    E.g., `['./cmd', '-file', 'a.out']`.
* Other words will be recognized as a flag set name if possible, otherwise it's a value argument.
    E.g., `['./cmd', 'build', '-o', 'cmd']`.
    
# Parsing
* Flag/FlagSet
  * Names(tag: 'names'): split by ',', fully custom: short, long, with or without '-'.
  * Arglist(tag: 'arglist'): show commandline of flag or flag set, 
    E.g., `-input INPUT -output OUTPUT... -t 'tag list'`.
  * Usage(tag: 'usage'): the short help message for this flag or flag set, 
    E.g., `build       compile packages and dependencies`.
  * Desc(tag: 'desc'): long description for this flag or flag set,  
    will be split to multiple lines and format with same indents.
  * Ptr(field pointer for Flag, field 'Enable bool' for FlagSet): result pointer
  
* Flag (structure field)
  * Default(tag: 'default'): default value
  * Selects(tag: 'selects'): selectable values, must be slice.
  * Env(tag: 'env'): environment variable, only used when flag not appeared in arguments.
  * ValSep(tag: 'valsep'): slice value separator for environment variable's value,
  
* FlagSet (embed structure)
  * Expand(tag: expand): always expand subset info in help message.
  * Version(tag: version): app version, will be split to multiple lines and format with same indents.
  * ArgsPtr(field: 'Args'): pointer to accept all the last non-flag values, 
    nil if don't need and error will be reported automatically.
  
* FlagMeta
```Go
type FlagMetadata interface {
    // the key can be flag group split by ',' E.g. "" and "build, -o", the "" is for
    // root FlagSet. Only Arglist, Usage, Desc, Version will be updated.
    Metadata() map[string]Flag 
}
````
  Structure can implement the FlagMetadata method to update flag metadata instead write in structure tag, it's designed
  for long messages.
   
  
# Code
```Go

import (
	"fmt"
	"testing"

	"github.com/cosiner/argv"
	"github.com/cosiner/flag"
)

func TestFlag(t *testing.T) {
	argv, err := argv.Argv([]rune("./flag  -ab false  -e 1  1234 5 6 7"), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	args := argv[0]

	type Flags struct {
		Args []string

		A   bool     `desc:"A is flag a" default:"true"`
		B   bool     `args:"B"`
		E   string   `desc:"E" important:"1"`
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
		} `args:"BARZ" important:"1" usage:"barz command" collapse:"true" desc:"aaabbbbbbbbbbbbb\nbbbbbbbbbbccc\nddd"`
	}

	var fs Flags
	cmdline := flag.NewFlagSet(flag.Flag{
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
	})
	cmdline.ParseStruct(&fs, args...)
	fmt.Printf("%s\n", cmdline.ToString(false))
	fmt.Printf("%+v\n", fs)
}

```
## Output
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
      -e            (string)
            E

      -a            (bool; default: true)
            A is flag a
      -b            (bool)
      -f            ([]string; default: [2]; selects: [2 3 4])
            F
            aa
      -h, --help    show help (bool)
      -v, --verbose show verbose help (bool)

Sets:
      barz [FLAG]... barz command

      foo [SET]...   foo command

{Args:[1234 5 6 7] A:true B:false E:1 F:[] Foo:{Enable:false FFoo:{Enable:false} BVarz:{Enable:false}} Barz:{Enable:false C:[] D:[]}}
```

# LICENSE
MIT.
