package flag

import (
	"os"
	"testing"
)

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
