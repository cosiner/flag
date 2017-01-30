package flag

import (
	"testing"

	"github.com/cosiner/argv"
)

func TestFlag(t *testing.T) {
	args := argv.Argv("./flag -f 2 3 4")

	cmdline := NewFlagSet(args[0], `test flags`)

	type Flags struct {
		A    bool     `names:"-a" usage:"A"`
		B    int      `names:"-b" usage:"B" default:"2" selects:"2,3,4"`
		E    string   `names:"-e" usage:"E" default:"2" selects:"2,3,4"`
		F    []string `names:"-f" usage:"F" default:"2" selects:"2,3,4"`
		Barz struct {
			Enable bool

			C []int    `names:"-c,  --col" desc:"tag list" usage:"C" default:"3,4,5" selects:"3,4,5"`
			D []string `names:"-d" usage:"D" desc:"tag" default:"6,7,8"`
		} `usage:"barz"`
	}

	var fs Flags
	err := cmdline.ParseStruct(&fs, args...)
	if err != nil {
		t.Error(err)
	} else {
		t.Logf(cmdline.String())
	}
}
