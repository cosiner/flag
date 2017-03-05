package flag

import (
	"errors"
	"fmt"
	"go/ast"
	"reflect"
	"strings"
	"unicode"
)

type register struct {
}

var defaultReguster register

func (r register) addIndexes(indexes map[string]int, keys []string, index int) {
	for _, key := range keys {
		indexes[key] = index
	}
}

func (r register) findDuplicates(parent, set *FlagSet, names []string) []string {
	var duplicates []string
	for _, name := range names {
		if set.isFlagOrSubset(name) || (parent != nil && parent.isFlagOrSubset(name)) {
			duplicates = append(duplicates, name)
			continue
		}
		for i := range set.subsets {
			if set.subsets[i].isFlagOrSubset(name) {
				duplicates = append(duplicates, name)
				break
			}
		}
	}
	return duplicates
}

const (
	flagNameSeparatorForSplit = ","
	flagNameSeparatorForJoin  = ", "
)

func (r register) joinFlagNames(names []string) string {
	return strings.Join(names, flagNameSeparatorForJoin)
}

func (r register) cleanFlagNames(names string) ([]string, string) {
	ns := splitAndTrimSpace(names, flagNameSeparatorForSplit)
	return ns, r.joinFlagNames(ns)
}

func (r register) cleanFlag(flag *Flag) {
	if flag.ValSep == "" {
		flag.ValSep = ","
	}
	if strings.Contains(flag.Arglist, " ") {
		flag.Arglist = "'" + flag.Arglist + "'"
	}
	r.updateFlagDesc(flag, flag.Desc)
	r.updateFlagVersion(flag, flag.Version)
}

func (r register) registerFlag(parent, set *FlagSet, flag Flag) error {
	refval := reflect.ValueOf(flag.Ptr)
	if refval.Kind() != reflect.Ptr {
		return fmt.Errorf("illegal flag pointer: %s", flag.Names)
	}
	if typeName(flag.Ptr) == "" {
		return fmt.Errorf("unsupported flag type: %s", flag.Names)
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
			return fmt.Errorf("incompatible default value type: %s", flag.Names)
		}
	}
	if flag.Selects != nil {
		var err error
		flag.Selects, err = parseSelectsValue(flag.Ptr, flag.Selects)
		if err != nil {
			return fmt.Errorf("%s: %s", flag.Names, err.Error())
		}
	}

	ns, names := r.cleanFlagNames(flag.Names)
	if duplicates := r.findDuplicates(parent, set, ns); len(duplicates) > 0 {
		return fmt.Errorf("duplicate flags with parent/self/childs: %v", duplicates)
	}

	flag.Names = names
	r.cleanFlag(&flag)

	set.flags = append(set.flags, flag)
	r.addIndexes(set.flagIndexes, ns, len(set.flags)-1)
	return nil
}

func (r register) registerSet(parent, set *FlagSet, flag Flag) (*FlagSet, error) {
	var ns []string
	ns, flag.Names = r.cleanFlagNames(flag.Names)
	if duplicates := r.findDuplicates(parent, set, ns); len(duplicates) > 0 {
		return nil, fmt.Errorf("duplicate subset name: %v", duplicates)
	}

	child := newFlagSet(flag)
	child.self.Default = false
	child.errorHandling = set.errorHandling

	set.subsets = append(set.subsets, *child)
	r.addIndexes(set.subsetIndexes, ns, len(set.subsets)-1)
	return &set.subsets[len(set.subsets)-1], nil
}

func (r register) registerStructure(parent, set *FlagSet, st interface{}, excludeField string) error {
	const (
		tagNames   = "names"
		tagArglist = "arglist"
		tagUsage   = "usage"
		tagDesc    = "desc"
		tagVersion = "version"

		tagEnv            = "env"
		tagValsep         = "valsep"
		tagDefault        = "default"
		tagSelects        = "selects"
		tagExpand         = "expand"
		fieldSubsetEnable = "Enable"
	)

	refval := reflect.ValueOf(st)
	if refval.Kind() != reflect.Ptr || refval.Elem().Kind() != reflect.Struct {
		return errors.New("not pointer of structure")
	}
	refval = refval.Elem()
	reftyp := refval.Type()
	numfield := refval.NumField()
	for i := 0; i < numfield; i++ {
		fieldType := reftyp.Field(i)
		if fieldType.Name == excludeField || !ast.IsExported(fieldType.Name) {
			continue
		}
		var (
			fieldVal = refval.Field(i)
			names    = fieldType.Tag.Get(tagNames)
			usage    = fieldType.Tag.Get(tagUsage)
			desc     = fieldType.Tag.Get(tagDesc)
			version  = fieldType.Tag.Get(tagVersion)
			arglist  = fieldType.Tag.Get(tagArglist)
		)
		if fieldVal.Kind() != reflect.Struct {
			var (
				ptr     = fieldVal.Addr().Interface()
				env     = fieldType.Tag.Get(tagEnv)
				def     = fieldType.Tag.Get(tagDefault)
				valsep  = fieldType.Tag.Get(tagValsep)
				selects = fieldType.Tag.Get(tagSelects)
			)
			if names == "" {
				names = "-" + unexportedName(fieldType.Name)
			}
			if valsep == "" {
				valsep = ","
			}
			if typeName(ptr) == "" {
				continue
			}
			defVal, err := parseDefault(def, valsep, ptr)
			if err != nil {
				return err
			}
			selectsVal, err := parseSelectsString(selects, valsep, ptr)
			if err != nil {
				return err
			}
			err = r.registerFlag(parent, set, Flag{
				Names:   names,
				Arglist: arglist,
				Usage:   usage,
				Desc:    desc,
				Version: version,

				Ptr:     ptr,
				Env:     env,
				ValSep:  valsep,
				Default: defVal,
				Selects: selectsVal,
			})
			if err != nil {
				return err
			}
		} else {
			enableFieldVal := fieldVal.FieldByName(fieldSubsetEnable)
			if enableFieldVal.Kind() != reflect.Bool {
				return fmt.Errorf("illegal child field type: %s", fieldSubsetEnable)
			}
			var (
				expand = fieldType.Tag.Get(tagExpand)
				ptr    = enableFieldVal.Addr().Interface().(*bool)
			)
			if names == "" {
				names = unexportedName(fieldType.Name)
			}
			if expand == "" {
				expand = "false"
			}
			child, err := r.registerSet(parent, set, Flag{
				Names:   names,
				Arglist: arglist,
				Usage:   usage,
				Desc:    desc,
				Version: version,

				Ptr: ptr,
			})
			if err != nil {
				return err
			}
			child.expand, err = parseBool(expand)
			if err != nil {
				return fmt.Errorf("parse expand value %s as bool failed", expand)
			}
			err = r.registerStructure(set, child, fieldVal.Addr().Interface(), fieldSubsetEnable)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (r register) registerBoolFlags(parent, set *FlagSet, names []string, usage string) (bool, error) {
	if len(names) == 0 {
		return false, nil
	}
	if duplicates := r.findDuplicates(parent, set, names); len(duplicates) > 0 {
		return false, nil
	}
	var value bool
	err := r.registerFlag(parent, set, Flag{
		Ptr:   &value,
		Names: strings.Join(names, ","),
		Usage: usage,
	})
	return err == nil, err
}

func (r register) registerHelpFlags(parent, set *FlagSet) error {
	registered, err := r.registerBoolFlags(parent, set, []string{"-h", "--help"}, "show help")
	if err == nil && registered {
		_, err = r.registerBoolFlags(parent, set, []string{"-v", "--verbose"}, "show verbose help")
	}
	return err
}

func (r register) boolFlagVal(set *FlagSet, flag string) (val, has bool) {
	index, has := set.flagIndexes[flag]
	if !has {
		return false, false
	}
	return *set.flags[index].Ptr.(*bool), true
}

func (r register) helpFlagValues(set *FlagSet) (show, verbose bool) {
	var has bool
	show, has = r.boolFlagVal(set, "-h")
	if show {
		if has {
			verbose, _ = r.boolFlagVal(set, "-v")
		}
	}
	return
}

func (r register) prefixSpaceCount(s string) int {
	var c int
	for _, r := range s {
		if unicode.IsSpace(r) {
			c += 1
		} else {
			break
		}
	}
	return c
}

func (r register) splitLines(line string) []string {
	var (
		lines    = strings.Split(line, "\n")
		begin    = -1
		end      = -1
		minSpace = -1
	)
	for i := range lines {
		if strings.TrimSpace(lines[i]) != "" {
			if begin < 0 {
				begin = i
			}
			end = i + 1
			c := r.prefixSpaceCount(lines[i])
			if minSpace > c || minSpace < 0 {
				minSpace = c
			}
		}
	}
	if begin < 0 {
		return nil
	}
	lines = lines[begin:end]
	for i := range lines {
		line := lines[i]
		if len(line) >= minSpace {
			line = line[minSpace:]
		}
		lines[i] = line
	}
	return lines
}

func (r register) updateFlagDesc(flag *Flag, desc string) {
	flag.Desc = desc
	flag.descLines = r.splitLines(flag.Desc)
}

func (r register) updateFlagVersion(flag *Flag, version string) {
	flag.Version = version
	flag.versionLines = r.splitLines(flag.Version)
}

func (r register) updateDesc(set *FlagSet, childs, desc string) error {
	var (
		currSet  = set
		currFlag *Flag

		sections = splitAndTrimSpace(childs, flagNameSeparatorForSplit)
		last     = len(sections) - 1
	)
	for i, sec := range sections {
		index, has := currSet.subsetIndexes[sec]
		if has {
			currSet = &currSet.subsets[index]
			continue
		}
		if i != last {
			return fmt.Errorf("subset %s is not found", sec)
		}
		index, has = currSet.flagIndexes[sec]
		if !has {
			return fmt.Errorf("subset or flag %s is not found", sec)
		}
		currFlag = &currSet.flags[index]
	}
	if currFlag == nil {
		currFlag = &currSet.self
	}
	r.updateFlagDesc(currFlag, desc)
	return nil
}