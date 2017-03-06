package flag

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
)

// Flag represents the state of a flag
type Flag struct {
	Names        string   // names
	Arglist      string   // arguments list
	Usage        string   // usage
	Desc         string   // description
	descLines    []string // parsed description lines
	Version      string   // version
	Important    bool     // important flag, will be print before unimportant flags
	versionLines []string // parsed version lines
	Expand       bool     // expand subsets in help message

	Ptr     interface{} // value pointer
	ArgsPtr *[]string   // NArgs pointer
	Default interface{} // default value
	Selects interface{} // select value
	Env     string      // environment name
	ValSep  string      // environment value separator
}

func (f *FlagSet) searchFlag(name string) *Flag {
	index, has := f.flagIndexes[name]
	if !has {
		return nil
	}
	return &f.flags[index]
}

func (f *FlagSet) isFlag(name string) bool {
	_, has := f.flagIndexes[name]
	return has
}

func (f *FlagSet) isSubset(name string) bool {
	_, has := f.subsetIndexes[name]
	return has
}

func (f *FlagSet) isFlagOrSubset(name string) bool {
	return f.isFlag(name) || f.isSubset(name)
}

type ErrorHandling uint16

const (
	ErrPrint ErrorHandling = 1 << iota
	ErrExit
	ErrPanic

	DefaultErrorHandling = ErrPrint | ErrExit
)

func (e ErrorHandling) do(eh ErrorHandling) bool {
	return e&eh != 0
}

func (e ErrorHandling) handle(err error) error {
	if err == nil {
		return nil
	}

	if e.do(ErrPanic) {
		panic(err)
	}
	if e.do(ErrPrint) {
		fmt.Fprintln(os.Stderr, err)
	}
	if e.do(ErrExit) {
		os.Exit(2)
	}
	return err
}

type FlagSet struct {
	self Flag

	flags       []Flag
	flagIndexes map[string]int

	subsets       []FlagSet
	subsetIndexes map[string]int

	errorHandling   ErrorHandling
	noHelpFlag      bool
	helpFlagDefined bool
}

func NewFlagSet(flag Flag) *FlagSet {
	if flag.Names == "" {
		flag.Names = filepath.Base(os.Args[0])
	}
	return newFlagSet(flag)
}

func newFlagSet(flag Flag) *FlagSet {
	defaultReguster.cleanFlag(&flag)
	return &FlagSet{
		self:          flag,
		flagIndexes:   make(map[string]int),
		subsetIndexes: make(map[string]int),
		errorHandling: DefaultErrorHandling,
	}
}

func (f *FlagSet) UpdateMeta(children string, meta Flag) error {
	return defaultReguster.updateMeta(f, children, meta)
}

func (f *FlagSet) ErrHandling(ehs ...ErrorHandling) *FlagSet {
	var e ErrorHandling
	for _, eh := range ehs {
		e |= eh
	}
	f.errorHandling = e
	for i := range f.subsets {
		f.subsets[i].ErrHandling(f.errorHandling)
	}
	return f
}

func (f *FlagSet) NeedHelpFlag(need bool) *FlagSet {
	f.noHelpFlag = !need
	for i := range f.subsets {
		f.subsets[i].NeedHelpFlag(need)
	}
	return f
}

func (f *FlagSet) Flag(flag Flag) error {
	return f.errorHandling.handle(defaultReguster.registerFlag(nil, f, flag))
}

func (f *FlagSet) Subset(flag Flag) (*FlagSet, error) {
	child, err := defaultReguster.registerSet(nil, f, flag)
	return child, f.errorHandling.handle(err)
}

type FlagMetadata interface {
	Metadata() map[string]Flag
}

func (f *FlagSet) StructFlags(val interface{}) error {
	return f.errorHandling.handle(defaultReguster.registerStructure(nil, f, val, ""))
}

func (f *FlagSet) Parse(args ...string) error {
	if len(args) == 0 {
		args = os.Args
	}
	if !f.noHelpFlag && !f.helpFlagDefined {
		err := defaultReguster.registerHelpFlags(nil, f)
		if err != nil {
			return f.errorHandling.handle(err)
		}
	}
	var (
		s scanner
		r resolver
	)
	s.scan(f, args)
	err := r.resolve(f, &s.Result)
	if err != nil {
		return f.errorHandling.handle(err)
	}

	show, verbose := defaultReguster.helpFlagValues(f)
	if show {
		fmt.Print(r.LastSet.ToString(verbose))
		os.Exit(0)
	}
	return nil
}

func (f *FlagSet) ParseStruct(val interface{}, args ...string) error {
	err := f.StructFlags(val)
	if err != nil {
		return err
	}
	return f.Parse(args...)
}

func (f *FlagSet) ToString(verbose bool) string {
	var buf bytes.Buffer
	(&writer{
		buf:          &buf,
		isTop:        true,
		forceVerbose: verbose,
	}).writeSet(f)
	return buf.String()
}

func (f *FlagSet) Help(verbose bool) {
	fmt.Print(f.ToString(verbose))
}

func (f *FlagSet) Reset() {
	var r resolver
	r.reset(f)
}

var (
	Commandline = NewFlagSet(Flag{})
)

func ParseStruct(val interface{}, args ...string) error {
	return Commandline.ParseStruct(val, args...)
}

func Help(verbose bool) {
	Commandline.Help(verbose)
}
