package flag

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
)

type Flag struct {
	Ptr       interface{}
	Names     string
	Default   interface{}
	Env       string
	EnvValSep string
	Usage     string
}

func (f *Flag) Apply(vals ...string) error {
	for _, val := range vals {
		err := applyValToPtr(f.Ptr, val)
		if err != nil {
			return err
		}
	}
	return nil
}

func (f *Flag) parseDefault() []string {
	if !isSlicePtr(f.Ptr) {
		return []string{fmt.Sprint(f.Default)}
	}

	refval := reflect.ValueOf(f.Default)
	vals := make([]string, 0, refval.Len())
	for i, l := 0, refval.Len(); i < l; i++ {
		val := fmt.Sprint(refval.Index(i).Interface())
		if val != "" {
			vals = append(vals, val)
		}
	}
	return vals
}

func (f *Flag) parseEnv() []string {
	val := os.Getenv(f.Env)
	if val == "" {
		return nil
	}

	var vals []string
	if isSlicePtr(f.Ptr) {
		vals = splitAndTrimSpace(val, ",")
	} else {
		vals = []string{val}
	}
	return vals
}

type FlagSet struct {
	selfEnable Flag

	flags         []Flag
	flagIndexes   map[string]int
	subsets       []FlagSet
	subsetIndexes map[string]int
}

func NewFlagSet(usage string) *FlagSet {
	return &FlagSet{
		selfEnable: Flag{
			Usage: usage,
		},
		flagIndexes:   make(map[string]int),
		subsetIndexes: make(map[string]int),
	}
}

func (f *FlagSet) StructFlags(val interface{}) error {
	return f.structFlags(val, "")
}

func (f *FlagSet) structFlags(val interface{}, excludeField string) error {
	const (
		TAG_NAMES   = "names"
		TAG_USAGE   = "usage"
		TAG_ENV     = "env"
		TAG_ENVSEP  = "envsep"
		TAG_DEFAULT = "default"

		FIELD_SUBSET_ENABLE = "Enable"
	)

	refval := reflect.ValueOf(val)
	if refval.Kind() != reflect.Ptr || refval.Elem().Kind() != reflect.Struct {
		return errors.New("not  pointer of structure")
	}
	refval = refval.Elem()
	reftyp := refval.Type()

	numfield := refval.NumField()
	for i := 0; i < numfield; i++ {
		fieldType := reftyp.Field(i)
		if fieldType.Name == FIELD_SUBSET_ENABLE {
			continue
		}

		fieldVal := refval.Field(i)
		ptr := fieldVal.Addr().Interface()

		names := fieldType.Tag.Get(TAG_NAMES)
		usage := fieldType.Tag.Get(TAG_USAGE)
		env := fieldType.Tag.Get(TAG_ENV)
		envsep := envValSep(fieldType.Tag.Get(TAG_ENVSEP))
		def := fieldType.Tag.Get(TAG_DEFAULT)

		if fieldVal.Kind() != reflect.Struct {
			if typeName(ptr) == "" {
				continue
			}

			if names == "" {
				names = "-" + unexportedName(fieldType.Name)
			}
			defval, err := parseDefault(def, envsep, ptr)
			if err != nil {
				return err
			}
			f.Flag(Flag{
				Names:     names,
				Ptr:       ptr,
				Env:       env,
				EnvValSep: envsep,
				Usage:     usage,
				Default:   defval,
			})
		} else {
			childFieldVal := fieldVal.FieldByName(FIELD_SUBSET_ENABLE)
			if childFieldVal.Kind() != reflect.Bool {
				return fmt.Errorf("illegal child field type: %s", fieldType.Name)
			}

			if names == "" {
				names = unexportedName(fieldType.Name)
			}
			set := f.SubSet(childFieldVal.Addr().Interface().(*bool), names, usage)
			err := set.structFlags(fieldVal.Addr().Interface(), FIELD_SUBSET_ENABLE)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (f *FlagSet) Flag(flag Flag) *FlagSet {
	refval := reflect.ValueOf(flag.Ptr)
	if refval.Kind() != reflect.Ptr {
		panic(fmt.Errorf("illegal flag pointer: %s", flag.Names))
	}
	if typeName(flag.Ptr) == "" {
		panic(fmt.Errorf("unsupported flag type: %s", flag.Names))
	}
	if flag.Default != nil {
		var compatible bool

		refdef := reflect.ValueOf(flag.Default)
		if isSlicePtr(flag.Ptr) {
			compatible = refdef.Kind() == reflect.Slice
			compatible = compatible && isKindCompatible(sliceElemKind(refval.Elem()), sliceElemKind(refdef))
		} else {
			compatible = isKindCompatible(refval.Elem().Kind(), refdef.Kind())
		}
		if !compatible {
			panic(fmt.Errorf("incompatible default value type: %s", flag.Names))
		}
	}

	var (
		index = len(f.flags)
		ns    = splitAndTrimSpace(flag.Names, ",")
	)
	for _, name := range ns {
		if _, has := f.flagIndexes[name]; has {
			panic(fmt.Errorf("duplicate flags: %s", name))
		}

		f.flagIndexes[name] = index
	}
	flag.EnvValSep = envValSep(flag.EnvValSep)
	flag.Names = strings.Join(ns, ", ")
	f.flags = append(f.flags, flag)
	return f
}

func (f *FlagSet) SubSet(ptr *bool, name, usage string) *FlagSet {
	_, has := f.subsetIndexes[name]
	if has {
		panic(fmt.Errorf("duplicate subset name: %s", name))
	}

	set := NewFlagSet(usage)
	set.selfEnable.Names = name
	set.selfEnable.Default = false
	set.selfEnable.Ptr = ptr
	f.subsets = append(f.subsets, *set)
	index := len(f.subsets) - 1
	f.subsetIndexes[name] = index
	return &f.subsets[index]
}

func (f *FlagSet) splitFlags(args []string) (global []string, sub map[string][]string) {
	sub = make(map[string][]string)

	var subname string
	for _, arg := range args {
		secs := strings.SplitN(arg, "=", 2)

		arg = secs[0]
		if subname == "" {
			if _, has := f.subsetIndexes[arg]; has && len(sub[arg]) == 0 {
				subname = arg
			}
		} else {
			if _, has := f.flagIndexes[arg]; has {
				subname = ""
			}
		}

		if subname == "" {
			global = append(global, secs...)
		} else {
			sub[subname] = append(sub[subname], secs...)
		}
	}
	return global, sub
}

func (f *FlagSet) parseGlobalFlags(args []string) error {
	var (
		applied      = make(map[*Flag]bool)
		lastFlagName string
		lastFlag     *Flag

		err error
	)

	var applyEmptyBoolFlag = func() error {
		if lastFlagName == "" {
			return nil
		}

		if isBoolPtr(lastFlag.Ptr) {
			return lastFlag.Apply("true")
		}
		return fmt.Errorf("empty flag: %s", lastFlagName)
	}
	for _, arg := range args {
		index, has := f.flagIndexes[arg]
		if has {
			f := &f.flags[index]
			err = applyEmptyBoolFlag()
			if err != nil {
				return err
			}

			lastFlag = f
			lastFlagName = arg
		} else if lastFlag == nil {
			return fmt.Errorf("unsupported flag: %s", arg)
		} else {
			err = lastFlag.Apply(arg)
			if err != nil {
				return err
			}

			if lastFlagName != "" {
				applied[lastFlag] = true
				lastFlagName = ""
			}
		}
	}
	err = applyEmptyBoolFlag()
	if err != nil {
		return err
	}

	for i := range f.flags {
		flag := &f.flags[i]
		if !applied[flag] {
			applied[flag] = true

			if flag.Env != "" {
				vals := flag.parseEnv()
				if len(vals) > 0 {
					err = flag.Apply(vals...)
					if err != nil {
						return err
					}

					continue
				}
			}
			if flag.Default != nil {
				flag.Apply(flag.parseDefault()...)
			}
		}
	}
	return nil
}

func (f *FlagSet) defineHelpFlags() *bool {
	const (
		HELP_FLAG_SHORT = "-h"
		HELP_FLAG_LONG  = "--help"
	)

	_, hasShort := f.flagIndexes[HELP_FLAG_SHORT]
	_, hasLong := f.flagIndexes[HELP_FLAG_LONG]
	if hasShort || hasLong {
		return nil
	}

	var showHelp bool
	f.Flag(Flag{
		Ptr:   &showHelp,
		Names: HELP_FLAG_SHORT + "," + HELP_FLAG_LONG,
		Usage: "showw help",
	})
	return &showHelp
}

func (f *FlagSet) Parse(args ...string) error {
	showHelp := f.defineHelpFlags()

	if len(args) == 0 {
		args = os.Args
	}

	global, sub := f.splitFlags(args[1:])
	err := f.parseGlobalFlags(global)
	if err != nil {
		return err
	}

	for sub, args := range sub {
		index := f.subsetIndexes[sub]
		set := &f.subsets[index]
		err = set.selfEnable.Apply("true")
		if err != nil {
			return err
		}
		err = set.Parse(args...)
		if err != nil {
			return err
		}
	}

	if showHelp != nil && *showHelp {
		fmt.Println(f)
		os.Exit(0)
	}
	return nil
}

func (f *FlagSet) writeToBuffer(buf *bytes.Buffer, indent, name string) {
	const INDENT = "  "

	var write = func(indent, s string) {
		buf.WriteString(indent)
		buf.WriteString(s)
	}
	var writeln = func(indent, s string) {
		write(indent, s)
		buf.WriteByte('\n')
	}

	if name != "" {
		writeln(indent, name)
	}
	if f.selfEnable.Usage != "" {
		writeln(indent, f.selfEnable.Usage)
	}

	flagsLen := len(f.flags)
	subsetsLen := len(f.subsets)
	if flagsLen > 0 {
		writeln("", "")
		if subsetsLen > 0 {
			writeln(indent, "GLOBAL FLAGS：")
		}

		flagIndent := indent + INDENT
		flagUsageIndent := flagIndent + INDENT
		for i := range f.flags {
			flag := &f.flags[i]

			write(flagIndent, flag.Names)

			if flag.Env != "" {
				write("", fmt.Sprintf(" (ENV: '%s'", flag.Env))
				if isSlicePtr(flag.Ptr) {
					write("", fmt.Sprintf(", splitted by '%s'", flag.EnvValSep))
				}
				write("", ")")
			}
			if flag.Default != nil {
				write("", fmt.Sprintf(" (DEFAULT: %v)", flag.Default))
			}
			writeln("", fmt.Sprintf(" (TYPE: %s)", typeName(flag.Ptr)))
			if flag.Usage != "" {
				writeln(flagUsageIndent, flag.Usage)
			}
		}
	}

	if subsetsLen != 0 {
		writeln("", "")
		if flagsLen > 0 {
			writeln(indent, "SUBSET FLAGS:")
		}

		subsetIndent := indent + INDENT
		for i := range f.subsets {
			set := &f.subsets[i]
			set.writeToBuffer(buf, subsetIndent, set.selfEnable.Names)
		}
	}
}

func (f *FlagSet) String() string {
	buf := bytes.NewBuffer(make([]byte, 0, 512))
	f.writeToBuffer(buf, "", "")
	return buf.String()
}
