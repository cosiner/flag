package flag_test

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
