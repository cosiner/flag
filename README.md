# Flag
[![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg?style=flat)](https://godoc.org/github.com/cosiner/flag) 

Flag is a simple but powerful commandline flag parsing library for [Go](https://golang.org).

# Documentation
Documentation can be found at [Godoc](https://godoc.org/github.com/cosiner/flag)

# Supported features
* bool
  * `-f`, `-f=false`, `-f=true`, there is no `-f true` and `-f false` to avoid conflicting 
    with positional flag and non-flag values
* string,number
  * `-f a.go -n 100`
* slice:
  * `-f a.go -f b.go -f c.go`
* hint flag as value
  * `--` to hint next argument is value: `rm -- -a.go`, 
    `rm -- -a.go -b.go` will throws error for `-b.go` is invalid flag
  * `--*` to hint latter all arguments are value: `rm -- -a.go -b.go -c.go`
* useful tricks:
  * `-f a.go`, `-f=a.go`, `--file=a.go`
  * `-zcf=a.go`, `-zcf a.go`
  * `-I/usr/include`: only works for `-[a-zA-Z][^a-zA-Z].+`
* catch non-flag arguments:
  * `rm -rf a.go b.go c.go`, catchs `[a.go, b.go, c.go]` 
* positional flag:
  * `cp -f src.go dst.go`, catchs `SOURCE=a.go DESTINATION=dst.go`
  * This is implemented as a special case of non-flag arguments, positional flags will be applied first, and remain values
* value apply/checking
  * default value
  * environment value
  * value list for user selecting
* multiple flag names for one flag
* subcommand.

# Definition via structure field tag
* `names`: flag/command names, comma-speparated, default uses camelCase of field name(with a `-` prefix for flag)
  * `-` to skip this field
  * `@` to indicate that this is a positional flag
  * support multiple name and formats: eg: `-f, --file, -file'
* `arglist`: argument list for command or argument name for flag
  * for positional flag, this will also be used as it's display name if defined, otherwise field name is used.
  * command example: eg: `[OPTION]... SOURCE DESTINATION`, or `[FLAG]... FILE [ARG}...` 
  * flag example: eg: 'FILE', 'DESTINATION'
* `version`: version message for command
* `usage`: short description
* `desc`: long description
* `env`: environment name for flag, if user doesn't passed this flag, environment value will be used
* `default`: default value for flag, if user doesn't passed this flag and environment value not defined, it will be used 
* `args`: used to catching non-flag arguments, it's type must be `[]string`

* special cases
  * `Enable`, there must be a `Enable` field inside command to indicate whether user are using this command.
  * `Args`: this field will be used to store non-flag arguments if `args` tag is not defined
  * `Metadata`: structure could implement this interface to override settings defined by tags
  ```Go
    type Metadata interface {
	    // the key is comma-separated flag name to find command/flag,
        // each component represents a level, "" represents root command
        Metadata() map[string]Flag
    }
  ```
  
# Example
## Flags
```Go
package main

import (
	"fmt"

	flag "github.com/cosiner/flag"
)

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

func (t *Tar) Metadata() map[string]flag.Flag {
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
	return map[string]flag.Flag{
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

func main() {
	var tar Tar

	flag.NewFlagSet(flag.Flag{}).ParseStruct(&tar, os.Args...)
	fmt.Println(tar.GZ)
	fmt.Println(tar.Create)
	fmt.Println(tar.File)
	fmt.Println(tar.SourceFiles)

	// Output for `$ go build -o "tar" . && ./tar -zcf a.tgz a.go b.go`:
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

## FlagSet
```Go
package main

import (
	"fmt"

	flag "github.com/cosiner/flag"
)


type GoCmd struct {
	Build struct {
		Enable  bool
		Already bool   `names:"-a" important:"1" desc:"force rebuilding of packages that are already up-to-date."`
		Race    bool   `important:"1" desc:"enable data race detection.\nSupported only on linux/amd64, freebsd/amd64, darwin/amd64 and windows/amd64."`
		Output  string `names:"-o" arglist:"output" important:"1" desc:"only allowed when compiling a single package"`

		LdFlags  string   `names:"-ldflags" arglist:"'flag list'" desc:"arguments to pass on each go tool link invocation."`
		Packages []string `args:"true"`
	} `usage:"compile packages and dependencies"`
	Clean struct {
		Enable bool
	} `usage:"remove object files"`
}

func (*GoCmd) Metadata() map[string]flag.Flag {
	return map[string]flag.Flag{
		"": {
			Usage:   "Go is a tool for managing Go source code.",
			Arglist: "command [argument]",
		},
		"build": {
			Arglist: "[-o output] [-i] [build flags] [packages]",
			Desc: `
		Build compiles the packages named by the import paths,
		along with their dependencies, but it does not install the results.
		...
		The build flags are shared by the build, clean, get, install, list, run,
		and test commands:
			`,
		},
	}
}

func main() {
	var g GoCmd

	set := flag.NewFlagSet(flag.Flag{})
	set.ParseStruct(&g, os.Args...)

	if g.Build.Enable {
		if len(g.Build.Packages) == 0 {
			fmt.Fprintln(os.Stderr, "Error: you should at least specify one package")
			fmt.Println("")
			build, _ := set.FindSubset("build")
			build.Help(false) // display usage information for the "go build" command only
		} else {
			fmt.Println("Going to build with the following parameters:")
			fmt.Println(g.Build)
		}
	} else if g.Clean.Enable {
		fmt.Println("Going to clean with the following parameters:")
		fmt.Println(g.Clean)
	} else {
		set.Help(false) // display usage information, with list of supported commands
	}
}

// Test with `$ go build -o "go" . && ./go --help`
```
##Help Message
```
Go is a tool for managing Go source code.

Usage:
      flag.test command [argument]

Sets:
      build        compile packages and dependencies
      clean        remove object files
      doc          show documentation for package or symbol
      env          print Go environment information
      bug          start a bug report
      fix          run go tool fix on packages
      fmt          run gofmt on package sources
```
```
compile packages and dependencies

Usage:
      build [-o output] [-i] [build flags] [packages]

Description:
      Build compiles the packages named by the import paths,
      along with their dependencies, but it does not install the results.
      ...
      The build flags are shared by the build, clean, get, install, list, run,
      and test commands:

Flags:
      -a                  
            force rebuilding of packages that are already up-to-date.
      -race               
            enable data race detection.
            Supported only on linux/amd64, freebsd/amd64, darwin/amd64 and windows/amd64.
      -o output           
            only allowed when compiling a single package

      -ldflags 'flag list'
            arguments to pass on each go tool link invocation.
```
# LICENSE
MIT.
