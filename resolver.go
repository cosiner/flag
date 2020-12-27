package flag

import (
	"fmt"
	"os"
	"reflect"
)

var envParser = os.Getenv

type resolver struct {
	LastSet *FlagSet
}

func (r *resolver) fromDefault(f *Flag) []string {
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

func (r *resolver) fromEnv(f *Flag) []string {
	val := envParser(f.Env)
	if val == "" {
		return nil
	}

	var vals []string
	if isSlicePtr(f.Ptr) {
		vals = splitAndTrimSpace(val, f.ValSep)
	} else {
		vals = []string{val}
	}
	return vals
}

func (r *resolver) applyVals(f *Flag, vals ...string) error {
	for _, val := range vals {
		err := applyValToPtr(f.Names, f.Ptr, val, f.Selects)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *resolver) applyEnvAndDefault(f *FlagSet, applied map[*Flag]bool) error {
	for i := range f.flags {
		flag := &f.flags[i]
		if applied[flag] {
			continue
		}
		applied[flag] = true

		var vals []string
		if flag.Env != "" {
			vals = r.fromEnv(flag)
		}
		if len(vals) == 0 && flag.Default != nil {
			vals = r.fromDefault(flag)
		}
		err := r.applyVals(flag, vals...)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *resolver) resolveFlags(f *FlagSet, context []string, args []argument) error {
	var positional []*Flag
	for i := range f.flags {
		if f.flags[i].Names == flagNamePositional {
			positional = append(positional, &f.flags[i])
		}
	}
	var (
		applied = make(map[*Flag]bool)
		flag    *Flag
		err     error

		positionalIndex int
		applyValue      = func(flag *Flag, val string) error {
			applied[flag] = true
			return r.applyVals(flag, val)
		}
		applyLastFlag = func() error {
			if flag == nil {
				return nil
			}
			return newErrorf(errFlagValueNotProvided, "flag value is not provided: %v.%s", context, flag.Names)
		}
		hasFlag = func(args []argument) bool {
			for i := range args {
				if args[i].Type == argumentFlag {
					return true
				}
			}
			return false
		}
		appendNonFlagArg = func(arg argument, args []argument) error {
			if (positionalIndex >= len(positional) && f.self.ArgsPtr == nil) ||
				(!f.self.ArgsAnywhere && hasFlag(args[1:])) {
				return newErrorf(errNonFlagValue, "unexpected non-flag value: %v %s", context, arg.Value)
			}
			if positionalIndex < len(positional) {
				err = applyValue(positional[positionalIndex], arg.Value)
				if err != nil {
					return err
				}
				positionalIndex++
				return nil
			}

			slice := *f.self.ArgsPtr
			slice = append(slice, arg.Value)
			*f.self.ArgsPtr = slice
			return nil
		}
	)

	for i, arg := range args {
		switch arg.Type {
		case argumentFlag:
			err = applyLastFlag()
			if err != nil {
				return err
			}

			flag = f.searchFlag(arg.Value)
			if flag == nil {
				return newErrorf(errFlagNotFound, "unsupported flag: %v.%s", context, arg.Value)
			}
			if applied[flag] && !isSlicePtr(flag.Ptr) {
				return newErrorf(errDuplicateFlagParsed, "duplicated flag: %v.%s", context, flag.Names)
			}

			if arg.AttachValid {
				// directly consume flag attached value
				err = applyValue(flag, arg.Attached)
				if err != nil {
					return err
				}
				flag = nil
			} else if isBoolPtr(flag.Ptr) {
				// bool flag should not consume next value to not affect positional or non flag parsing
				err = applyValue(flag, "true")
				if err != nil {
					return err
				}
				flag = nil
			}
		case argumentValue:
			if flag == nil {
				err = appendNonFlagArg(args[i], args[i:])
				if err != nil {
					return err
				}
			} else {
				err = applyValue(flag, arg.Value)
				if err != nil {
					return err
				}
				flag = nil
			}
		default:
			panic("unreachable")
		}
	}
	err = applyLastFlag()
	if err != nil {
		return err
	}
	//if positionalIndex < len(positional) {
	//	var names []string
	//	for i := positionalIndex; i < len(positional); i++ {
	//		names = append(names, positional[i].Arglist)
	//	}
	//	return newErrorf(errPositionalFlagNotProvided, "flag not provided: %v.%v", context, names)
	//}

	return r.applyEnvAndDefault(f, applied)
}

func (r *resolver) resolveSet(f *FlagSet, context []string, args *scanArgs) (lastSubset *FlagSet, err error) {
	context = append(context, f.self.Names)
	err = r.resolveFlags(f, context, args.Flags[1:])
	if err != nil {
		return nil, err
	}
	for sub, subArgs := range args.Sets {
		set := &f.subsets[f.subsetIndexes[sub]]
		err = r.applyVals(&set.self, "true")
		if err != nil {
			return nil, err
		}

		last, err := r.resolveSet(set, context, subArgs)
		if err != nil {
			return nil, err
		}
		if sub == args.FirstSubset {
			lastSubset = last
			if lastSubset == nil {
				lastSubset = set
			}
		}
	}

	if lastSubset == nil {
		lastSubset = f
	}
	return lastSubset, nil
}

func (r *resolver) resolve(f *FlagSet, args *scanArgs) error {
	var err error
	r.LastSet, err = r.resolveSet(f, nil, args)
	return err
}

func (r *resolver) reset(f *FlagSet) {
	if f.self.ArgsPtr != nil {
		*f.self.ArgsPtr = nil
	}
	resetPtrVal(f.self.Ptr)
	for i := range f.flags {
		resetPtrVal(f.flags[i].Ptr)
	}
	for i := range f.subsets {
		r.reset(&f.subsets[i])
	}
}
