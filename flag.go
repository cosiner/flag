package flag

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
	"path/filepath"
)

// Flag represents the state of a flag
type Flag struct {
	Ptr       interface{}
	Names     string
	Default   interface{}
	Env       string
	EnvValSep string
	Usage     string
}

// Apply update flag state, for slice flags, values will be appended to state,
// for others, only latest value will be used.
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

// FlagSet represents a set of defined flag
type FlagSet struct {
	self Flag

	flags         []Flag
	flagIndexes   map[string]int
	subsets       []FlagSet
	subsetIndexes map[string]int
}

// NewFlagSet returns a new, empty flag set with the specified name and usage
func NewFlagSet(name, usage string) *FlagSet {
	if name == "" {
		name = filepath.Base(os.Args[0])
	}
	return &FlagSet{
		self: Flag{
			Names: name,
			Usage: usage,
		},
		flagIndexes:   make(map[string]int),
		subsetIndexes: make(map[string]int),
	}
}

// StructFlags parsing structure fields as Flag or FLagSet, base type such as int,
// string, bool will be defined as a Flag, embed structure will be defined as a
// sub-FlagSet.
//
// Structure fields can define some tags as flag properities:
//    names:   flag names
//    usage:   flag usage
//    env:     flag environment variable name
//    envsep:  flag value separator for slice values
//    default: flag default values, slice values will be splitted by envsep or ',
// Embed structure must has a 'Enable' field with type bool.
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
				return fmt.Errorf("illegal child field type: %s", FIELD_SUBSET_ENABLE)
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

// Flag defines a flag in current FlagSet
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

// SubSet defines a sub-FlagSet in current FlagSet
func (f *FlagSet) SubSet(ptr *bool, name, usage string) *FlagSet {
	_, has := f.subsetIndexes[name]
	if has {
		panic(fmt.Errorf("duplicate subset name: %s", name))
	}

	set := NewFlagSet(name, usage)
	set.self.Default = false
	set.self.Ptr = ptr
	f.subsets = append(f.subsets, *set)
	index := len(f.subsets) - 1
	f.subsetIndexes[name] = index
	return &f.subsets[index]
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
		Usage: "show help",
	})
	return &showHelp
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

func (f *FlagSet) applyEnvOrDefault(applied map[*Flag]bool) error {
	for i := range f.flags {
		flag := &f.flags[i]
		if applied[flag] {
			continue
		}
		applied[flag] = true

		var vals []string
		if flag.Env != "" {
			vals = flag.parseEnv()
		}
		if len(vals) == 0 && flag.Default != nil {
			vals = flag.parseDefault()
		}
		err := flag.Apply(vals...)
		if err != nil {
			return err
		}
	}
	return nil
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
			if lastFlagName != "" {
				applied[lastFlag] = true
				lastFlagName = ""
			} else {
				if !isSlicePtr(lastFlag.Ptr) {
					return fmt.Errorf("flag %s accept only one argument", lastFlag.Names)
				}
			}

			err = lastFlag.Apply(arg)
			if err != nil {
				return err
			}
		}
	}
	err = applyEmptyBoolFlag()
	if err != nil {
		return err
	}

	return f.applyEnvOrDefault(applied)
}

// Parse parsing specified arguments, first argument will be ignored. Arguments must
// be ordered in format 'NAME [FLAG | SET...]'.
//
// If there is no help flags(-h, --help) defined, Parse will define these, and
// print help string and then exit with code 0 if one of these two flag appeared.
func (f *FlagSet) Parse(args ...string) error {
	return f.parse(true, args)
}

func (f *FlagSet) parse(isTop bool, args []string) error {
	var showHelp *bool
	if isTop {
		showHelp = f.defineHelpFlags()
	}

	if len(args) == 0 {
		args = os.Args
	}

	global, sub := f.splitFlags(args[1:])
	err := f.parseGlobalFlags(global)
	if err != nil {
		return err
	}

	helpSet := f
	for sub, args := range sub {
		index := f.subsetIndexes[sub]
		set := &f.subsets[index]
		err = set.self.Apply("true")
		if err != nil {
			return err
		}
		err = set.parse(false, args)
		if err != nil {
			return err
		}
		if helpSet == f {
			helpSet = set
		}
	}

	if showHelp != nil && *showHelp {
		fmt.Println(helpSet)
		os.Exit(0)
	}
	return nil
}

func (f *FlagSet) writeToBuffer(buf *bytes.Buffer, indent string) {
	const INDENT = "      "

	var write = func(indent, s string) {
		buf.WriteString(indent)
		buf.WriteString(s)
	}
	var writeln = func(indent, s string) {
		write(indent, s)
		buf.WriteByte('\n')
	}

	writeln(indent, fmt.Sprintf("%s [FLAG | SET]...", f.self.Names))

	if f.self.Usage != "" {
		writeln(indent, f.self.Usage)
	}

	flagsLen := len(f.flags)
	subsetsLen := len(f.subsets)
	if flagsLen > 0 {
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

			var def string
			if flag.Default != nil {
				def = fmt.Sprintf(", default %v", flag.Default)
			}
			writeln("", fmt.Sprintf(" (%s%s)", typeName(flag.Ptr), def))

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
			set.writeToBuffer(buf, subsetIndent)
		}
	}
}

// String return a readable help string for current FlagSet
func (f *FlagSet) String() string {
	buf := bytes.NewBuffer(make([]byte, 0, 512))
	f.writeToBuffer(buf, "")
	return buf.String()
}
