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
* Structure parsing

# Supported types
* bool
* string
* integer, unsigned integer and float (int8, int16, int32, int64, int, uint..., float32, float64)
* slice of above types
* embed structure as sub-flagset.

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
  Structure can implement the Metadata interface to update flag metadata instead write in structure tag, it's designed
  for long messages.
   
  
# Code
```Go
package flag

import "fmt"

type Tar struct {
	GZ          bool     `names:"-z, --gz" usage:"gzip format"`
	BZ          bool     `names:"-j, --bz" usage:"bzip2 format"`
	XZ          bool     `names:"-J, --xz" usage:"xz format"`
	Create      bool     `names:"-c" usage:"create tar file"`
	Extract     bool     `names:"-x" usage:"extract tar file"`
	File        string   `names:"-f" usage:"output file for create or input file for extract"`
	Directory   string   `names:"-C" usage:"extract directory"`
	SourceFiles []string `args:"true"`
}

func (t *Tar) Metadata() map[string]Flag {
	const (
		usage   = "tar is a tool for manipulate tape archives."
		version = `
			version: v1.0.0
			commit: 10adf10dc10
			date:   2017-01-01 10:00:01
		`
		desc = `
		tar creates and manipulates streaming archive files.  This implementation can extract
		from tar, pax, cpio, zip, jar, ar, and ISO 9660 cdrom images and can create tar, pax,
        cpio, ar, and shar archives.
		`
	)
	return map[string]Flag{
		"": {
			Usage:   usage,
			Version: version,
			Desc:    desc,
		},
		"--gz": {
			Desc: "use gzip format",
		},
	}
}

func ExampleFlagSet_ParseStruct() {
	var tar Tar

	ParseStruct(&tar, "tar", "-zcf", "a.tgz", "a.go", "b.go")
	fmt.Println(tar.GZ)
	fmt.Println(tar.Create)
	fmt.Println(tar.File)
	fmt.Println(tar.SourceFiles)

	// Output:
	// true
	// true
	// a.tgz
	// [a.go b.go]
}

```
## Help message
```
tar is a tool for manipulate tape archives.

Usage:
      flag.test [FLAG]...

Version:
      version: v1.0.0
      commit: 10adf10dc10
      date:   2017-01-01 10:00:01

Description:
      tar creates and manipulates streaming archive files.  This implementation can extract
      from tar, pax, cpio, zip, jar, ar, and ISO 9660 cdrom images and can create tar, pax,
            cpio, ar, and shar archives.

Flags:
      -z, --gz     gzip format (bool)
            use gzip format
      -j, --bz     bzip2 format (bool)
      -J, --xz     xz format (bool)
      -c           create tar file (bool)
      -x           extract tar file (bool)
      -f           output file for create or input file for extract (string)
      -C           extract directory (string)
```

# LICENSE
MIT.
