package flag

import (
	"go/ast"
	"reflect"
	"strings"
	"unicode"
)

type register struct {
}

var defaultRegister register

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
	r.updateFlagDesc(flag, flag.Desc)
	r.updateFlagVersion(flag, flag.Version)
}

func (r register) updateFlagDefault(flag *Flag, def interface{}) error {
	refPtr := reflect.ValueOf(flag.Ptr)
	var compatible bool

	refdef := reflect.ValueOf(def)
	if isRefvalSlicePtr(refPtr) {
		compatible = refdef.Kind() == reflect.Slice
		compatible = compatible && isKindCompatible(sliceElemKind(refPtr.Elem()), sliceElemKind(refdef))
	} else {
		compatible = isKindCompatible(refPtr.Elem().Kind(), refdef.Kind())
	}
	if !compatible {
		return newErrorf(errInvalidType, "incompatible default value type: %s", flag.Names)
	}
	flag.Default = def
	return nil
}

func (r register) updateFlagSelects(flag *Flag, val interface{}) error {
	if val == nil {
		return nil
	}

	refval := reflect.ValueOf(flag.Ptr).Elem()
	k := sliceElemKind(refval)
	if isKindNumber(k) {
		fs := convertNumbersToFloats(val)
		flag.Selects = fs
		return nil
	}
	if k == reflect.String {
		if vals, ok := val.([]string); ok && len(vals) != 0 {
			flag.Selects = vals
			return nil
		}
	}
	return newErrorf(errInvalidSelects, "invalid selects: %s, %v", flag.Names, val)
}

func (r register) registerFlag(parent, set *FlagSet, flag Flag) error {
	refval := reflect.ValueOf(flag.Ptr)
	if refval.Kind() != reflect.Ptr {
		return newErrorf(errNonPointer, "illegal flag pointer: %s", flag.Names)
	}
	if typeName(flag.Ptr) == "" {
		return newErrorf(errInvalidType, "unsupported flag type: %s", flag.Names)
	}
	if flag.Default != nil {
		err := r.updateFlagDefault(&flag, flag.Default)
		if err != nil {
			return err
		}
	}
	if flag.Selects != nil {
		err := r.updateFlagSelects(&flag, flag.Selects)
		if err != nil {
			return err
		}
	}

	ns, names := r.cleanFlagNames(flag.Names)
	if duplicates := r.findDuplicates(parent, set, ns); len(duplicates) > 0 {
		if parent != nil {
			return newErrorf(errDuplicateFlagRegister, "duplicate flags with parent/self/children: %s->%s, %v", parent.self.Names, set.self.Names, duplicates)
		}
		return newErrorf(errDuplicateFlagRegister, "duplicate flags with self/children: %s, %v", set.self.Names, duplicates)
	}

	flag.Names = names
	r.cleanFlag(&flag)

	set.flags = append(set.flags, flag)
	r.addIndexes(set.flagIndexes, ns, len(set.flags)-1)
	return nil
}

func (r register) checkSubsetValid(flag *Flag) error {
	if flag.Names == "" {
		return newErrorf(errInvalidNames, "subset names should not be empty")
	}
	return nil
}

func (r register) registerSet(parent, set *FlagSet, flag Flag) (*FlagSet, error) {
	var ns []string

	ns, flag.Names = r.cleanFlagNames(flag.Names)
	err := r.checkSubsetValid(&flag)
	if err != nil {
		return nil, err
	}

	if duplicates := r.findDuplicates(parent, set, ns); len(duplicates) > 0 {
		return nil, newErrorf(errDuplicateFlagRegister, "duplicate subset name: %v", duplicates)
	}

	child := newFlagSet(flag)
	child.self.Default = false
	child.errorHandling = set.errorHandling

	set.subsets = append(set.subsets, *child)
	r.addIndexes(set.subsetIndexes, ns, len(set.subsets)-1)
	return &set.subsets[len(set.subsets)-1], nil
}

func (r register) registerStructure(parent, set *FlagSet, st interface{}) error {
	// parent is used to checking duplicate flags and indicate that subset must has a 'Enable' field
	const (
		tagNames   = "names"
		tagArglist = "arglist"
		tagUsage   = "usage"
		tagDesc    = "desc"
		tagVersion = "version"

		tagEnv          = "env"
		tagValsep       = "valsep"
		tagDefault      = "default"
		tagSelects      = "selects"
		tagArgs         = "args"
		tagArgsAnywhere = "anywhere"

		fieldSubsetEnable = "Enable"
		fieldArgs         = "Args"
	)

	refval := reflect.ValueOf(st)
	if refval.Kind() != reflect.Ptr || refval.Elem().Kind() != reflect.Struct {
		return newErrorf(errNonPointer, "not pointer of structure")
	}

	var (
		parseQueue = []reflect.Value{refval.Elem()}
		metadatas  []Metadata
	)
	for {
		l := len(parseQueue)
		if l == 0 {
			break
		}
		refval := parseQueue[0]
		copy(parseQueue, parseQueue[1:])
		parseQueue = parseQueue[:l-1]

		reftyp := refval.Type()
		numfield := refval.NumField()
		for i := 0; i < numfield; i++ {
			fieldType := reftyp.Field(i)
			if !ast.IsExported(fieldType.Name) {
				continue
			}

			fieldVal := refval.Field(i)

			args := fieldType.Tag.Get(tagArgs)
			isArgs, err := parseBool(args, "false")
			if err != nil {
				return newErrorf(errInvalidValue, "non-bool tag args value: %s.%s %s", set.self.Names, fieldType.Name, args)
			}
			if fieldType.Name == fieldArgs || isArgs {
				argsAnywhere := fieldType.Tag.Get(tagArgsAnywhere)
				anywhere, err := parseBool(argsAnywhere, "false")
				if err != nil {
					return newErrorf(errInvalidValue, "non-bool tag anywhere value: %s.%s %s", set.self.Names, fieldType.Name, argsAnywhere)
				}
				if set.self.ArgsPtr != nil {
					return newErrorf(errDuplicateFlagRegister, "duplicate args field: %s", set.self.Names)
				}
				if _, ok := fieldVal.Interface().([]string); !ok {
					return newErrorf(errInvalidType, "invalid %s:Args field type, expect []string", set.self.Names)
				}
				set.self.ArgsPtr = fieldVal.Addr().Interface().(*[]string)
				set.self.ArgsAnywhere = anywhere
				continue
			}

			ptr := fieldVal.Addr().Interface()
			if fieldType.Name == fieldSubsetEnable {
				if fieldType.Type.Kind() != reflect.Bool {
					return newErrorf(errInvalidType, "illegal type of field '%s', expect bool", fieldSubsetEnable)
				}
				if set.self.Ptr == nil {
					set.self.Ptr = ptr
				}
				continue
			}

			var (
				names   = fieldType.Tag.Get(tagNames)
				usage   = fieldType.Tag.Get(tagUsage)
				desc    = fieldType.Tag.Get(tagDesc)
				version = fieldType.Tag.Get(tagVersion)
				arglist = fieldType.Tag.Get(tagArglist)
			)
			if names == "-" {
				continue
			}
			_, ok := ptr.(NoFlag)
			if ok {
				continue
			}

			if fieldVal.Kind() != reflect.Struct {
				var (
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
			} else if fieldType.Anonymous {
				parseQueue = append(parseQueue, fieldVal)
			} else {
				if names == "" {
					names = unexportedName(fieldType.Name)
				}
				child, err := r.registerSet(parent, set, Flag{
					Names:   names,
					Arglist: arglist,
					Usage:   usage,
					Desc:    desc,
					Version: version,
				})
				if err != nil {
					return err
				}
				err = r.registerStructure(set, child, fieldVal.Addr().Interface())
				if err != nil {
					return err
				}
			}
		}
		if md, ok := st.(Metadata); ok {
			metadatas = append(metadatas, md)
		}
	}
	if parent != nil && set.self.Ptr == nil {
		return newErrorf(errInvalidStructure, "child structure must has a 'Enable' field")
	}
	for _, md := range metadatas {
		for children, meta := range md.Metadata() {
			err := r.updateMeta(set, children, meta)
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
	if err == nil && registered && len(set.subsets) > 0 {
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
			c++
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

func (r register) searchChildrenFlag(set *FlagSet, children string) (*Flag, *FlagSet, error) {
	var (
		currSet  = set
		currFlag *Flag

		sections = splitAndTrimSpace(children, flagNameSeparatorForSplit)
		last     = len(sections) - 1
	)
	for i, sec := range sections {
		index, has := currSet.subsetIndexes[sec]
		if has {
			currSet = &currSet.subsets[index]
			continue
		}
		if i != last {
			return nil, nil, newErrorf(errFlagNotFound, "subset/flag %s is not found", sec)
		}
		index, has = currSet.flagIndexes[sec]
		if !has {
			return nil, nil, newErrorf(errFlagNotFound, "subset/flag %s is not found", sec)
		}
		currFlag = &currSet.flags[index]
	}
	if currFlag == nil {
		currFlag = &currSet.self
	}
	return currFlag, currSet, nil
}

func (r register) updateMeta(set *FlagSet, children string, meta Flag) error {
	flag, subset, err := r.searchChildrenFlag(set, children)
	if err != nil {
		return err
	}
	if meta.Desc != "" {
		flag.Desc = meta.Desc
	}
	if subset != nil && meta.Version != "" {
		flag.Version = meta.Version
	}
	if meta.Arglist != "" {
		flag.Arglist = meta.Arglist
	}
	if meta.Usage != "" {
		flag.Usage = meta.Usage
	}
	if meta.Default != nil {
		err = r.updateFlagDefault(flag, meta.Default)
		if err != nil {
			return err
		}
	}
	if meta.Selects != nil {
		err = r.updateFlagSelects(flag, meta.Selects)
		if err != nil {
			return err
		}
	}
	if meta.Env != "" {
		flag.Env = meta.Env
	}
	r.cleanFlag(flag)
	return nil
}
