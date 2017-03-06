package flag_test

import (
	"reflect"
	"testing"

	"github.com/cosiner/argv"
	"github.com/cosiner/flag"
)

type Tar struct {
	Args []string

	GZ        bool   `names:"-z, --gz" usage:"gzip format"`
	BZ        bool   `names:"-j, --bz" usage:"bzip2 format"`
	XZ        bool   `names:"-J, --xz" usage:"xz format"`
	Create    bool   `names:"-c" usage:"create tar file"`
	Extract   bool   `names:"-x" usage:"extract tar file"`
	File      string `names:"-f" usage:"output file for create or input file for extract"`
	Directory string `names:"-C" usage:"extract directory"`
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
	}
}

func TestFlags(t *testing.T) {
	var cases = []struct {
		Env  map[string]string
		Cmds []string
		Tar
	}{
		{
			Cmds: []string{
				"tar -zcf a.tgz a b c",
				"tar -zc -f=a.tgz a b c",
				"tar -z -c -f a.tgz a b c",
				"tar --gz -c -f a.tgz a b c",
			},
			Tar: Tar{
				Args:   []string{"a", "b", "c"},
				GZ:     true,
				Create: true,
				File:   "a.tgz",
			},
		},
		{
			Cmds: []string{
				"tar -- -file -",
			},
			Tar: Tar{
				Args: []string{"-file", "-"},
			},
		},
		{
			Cmds: []string{
				"tar -- -file --",
			},
			Tar: Tar{
				Args: []string{"-file", "--"},
			},
		},
		{
			Cmds: []string{
				"tar - -z",
			},
			Tar: Tar{
				GZ: true,
			},
		},
		{
			Cmds: []string{
				"tar -Jxf a.txz -C /",
				"tar -Jxf a.txz -C/",
				"tar -Jxf a.txz -C=/",
			},
			Tar: Tar{
				XZ:        true,
				Extract:   true,
				File:      "a.txz",
				Directory: "/",
			},
		},
	}
	var tar Tar
	flags := flag.NewFlagSet(flag.Flag{}).ErrHandling(0)
	err := flags.StructFlags(&tar)
	if err != nil {
		t.Fatal(err)
	}

	for i, c := range cases {
		for j, cmd := range c.Cmds {
			argv, err := argv.Argv([]rune(cmd), c.Env, nil)
			if err != nil {
				t.Fatal(i, j, err)
			}
			err = flags.Parse(argv[0]...)
			if err != nil {
				t.Fatal(i, j, err)
			}
			if !reflect.DeepEqual(tar, c.Tar) {
				t.Errorf("match failed: %d %d, expect: %+v, got %+v", i, j, c.Tar, tar)
			}
			flags.Reset()
		}
	}
}
