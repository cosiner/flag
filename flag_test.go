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

			C []int    `names:"-c" usage:"C" default:"3,4,5"`
			D []string `names:"-d" usage:"D" default:"6,7,8"`
		}
	}

	var fs Flags
	err := cmdline.StructFlags(&fs)
	if err != nil {
		test.Fatal(err)
	}

	err = cmdline.Parse(os.Args[0], "-a", "false", "-b", "1", "barz", "-c", "1", "2", "3", "-d", "a", "b", "c")
	if err != nil {
		test.Fatal(err)
	}
	test.Logf("%+v", fs)
	test.Log(cmdline.String())
}
