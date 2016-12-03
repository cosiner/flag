# Flag

Flag is a command-line flag parsing library for [Go](https://golang.org), support slice, structure, env variables parsing.

# Documentation
Documentation can be found at [Godoc](https://godoc.org/github.com/cosiner/flag)

# Example
#### Code
```Go
func TestFlag(test *testing.T) {
	cmdline := NewFlagSet("", `test flags`)

	type Flags struct {
		A    bool `names:"-a" usage:"A"`
		B    int  `names:"-b" usage:"B" default:"2"`
		Barz struct {
			Enable bool

			C []int    `names:"-c,  --col" desc:"tag list" usage:"C" default:"3,4,5"`
			D []string `names:"-d" usage:"D" desc:"tag" default:"6,7,8"`
		} `usage:"barz"`
	}

	var fs Flags
	err := cmdline.StructFlags(&fs)
	if err != nil {
		test.Fatal(err)
	}

	err = cmdline.Parse(os.Args[0], "--help")
	if err != nil {
		test.Fatal(err)
	}
}
```
##### Output
```
flag.test [FLAG | SET]...
test flags
      -a (bool)
            A
      -b (int; default: 2)
            B
      -h, --help (bool)
            show help

      barz [FLAG | SET]...
      barz
            -c, --col 'tag list' ([]int; default: [3 4 5])
                  C
            -d tag ([]string; default: [6 7 8])
                  D
```

# LICENSE
MIT.
