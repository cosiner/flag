package flag

import (
	"bytes"
	"fmt"
	"strings"
)

const (
	minInfoLen = 12
	maxInfoLen = 24
)

type writer struct {
	buf           *bytes.Buffer
	isTop         bool
	inheritIndent string
	forceVerbose  bool
	maxInfoLen    int
}

func (w *writer) maxFlagInfoLen(f *FlagSet) int {
	var maxLen int
	for i := range f.flags {
		l := len(f.flags[i].Names)
		if f.flags[i].Arglist != "" {
			l += 1 + len(f.flags[i].Arglist)
		}
		if maxLen < l {
			maxLen = l
		}
	}
	if maxLen < minInfoLen {
		maxLen = minInfoLen
	}
	if maxLen > maxInfoLen {
		maxLen = maxInfoLen
	}
	return maxLen
}

func (w *writer) maxSubsetInfoLen(f *FlagSet, needArglist bool) int {
	var maxLen int
	for i := range f.subsets {
		l := len(f.subsets[i].self.Names)
		if needArglist {
			args := w.arglist(&f.subsets[i])
			if args != "" {
				l += 1 + len(args)
			}
		}
		if maxLen < l {
			maxLen = l
		}
	}
	if maxLen < minInfoLen {
		maxLen = minInfoLen
	}
	if maxLen > maxInfoLen {
		maxLen = maxInfoLen
	}
	return maxLen
}

func (w *writer) arglist(f *FlagSet) string {
	if f.self.Arglist == "-" {
		return ""
	}
	if f.self.Arglist != "" {
		return f.self.Arglist
	}
	var (
		arglist             string
		flagCount, setCount = len(f.flags), len(f.subsets)
	)
	if flagCount != 0 {
		if setCount != 0 {
			arglist = "[FLAG|COMMAND]..."
		} else {
			arglist = "[FLAG]..."
		}
	} else {
		if setCount != 0 {
			arglist = "[COMMAND]..."
		}
	}
	return arglist
}

func (w *writer) write(elem ...string) {
	for _, s := range elem {
		w.buf.WriteString(s)
	}
}

func (w *writer) nextIndent(curr string) string {
	const indent = "\t"
	return curr + indent
}

func (w *writer) writeln(elem ...string) {
	w.write(elem...)
	w.buf.WriteByte('\n')
}

func (w *writer) writeWithPads(names string, maxLen int) {
	w.write(names)
	if padlen := maxLen - len(names); padlen > 0 {
		w.write(strings.Repeat(" ", padlen))
	}
}

func (w *writer) writeLines(indent string, lines []string) {
	for _, line := range lines {
		w.writeln(indent, line)
	}
}

func (w *writer) parseFlagInfo(flag *Flag, args string) string {
	info := flag.Names
	if args != "" {
		info += " " + args
	}
	return info
}

func (w *writer) writeFlagInfo(currIndent string, flag *Flag, isTop bool, args string, maxInfoLen int) {
	w.write(currIndent)
	if isTop {
		if flag.Usage != "" {
			w.writeln(currIndent, flag.Usage)
			w.writeln()
		}
		w.writeln(currIndent, "Usage:")
		w.write(w.nextIndent(currIndent))
	}
	flagInfo := w.parseFlagInfo(flag, args)
	w.writeWithPads(flagInfo, maxInfoLen)
	if !isTop && flag.Usage != "" {
		w.write(" ", flag.Usage)
	}
}

func (w *writer) writeFlagValueInfo(flag *Flag) {
	w.write(" (")
	w.write("type: ", typeName(flag.Ptr))
	if flag.Env != "" || flag.Default != nil || flag.Selects != nil {
		if flag.Env != "" {
			w.write("; env: ", flag.Env)
			if isSlicePtr(flag.Ptr) {
				w.write(", splitted by ", fmt.Sprintf("'%s'", flag.ValSep))
			}
		}
		if flag.Default != nil {
			w.write("; default: ", fmt.Sprintf("%v", flag.Default))
		}
		if flag.Selects != nil {
			w.write("; selects: ", fmt.Sprintf("%v", flag.Selects))
		}
	}
	w.write(")")
}

func (w *writer) writeFlagSet(f *FlagSet) {
	var (
		currIndent       = w.inheritIndent
		flagIndent       = w.nextIndent(currIndent)
		outline          = !w.forceVerbose
		flagCount        = len(f.flags)
		subsetCount      = len(f.subsets)
		versionLineCount = len(f.self.versionLines)
		descLineCount    = len(f.self.descLines)
	)

	var arglist string
	if w.isTop {
		arglist = w.arglist(f)
	}
	w.writeFlagInfo(currIndent, &f.self, w.isTop, arglist, w.maxInfoLen)
	w.writeln()

	if outline && !w.isTop {
		return
	}
	if versionLineCount > 0 {
		if w.isTop {
			w.writeln()
			w.writeln(currIndent, "Version:")
			w.writeLines(flagIndent, f.self.versionLines)
		}
	}

	if descLineCount > 0 {
		if w.isTop {
			w.writeln()
			w.writeln(currIndent, "Description:")
			w.writeLines(flagIndent, f.self.descLines)
		}
	}

	if flagCount > 0 {
		if versionLineCount > 0 || descLineCount > 0 || w.isTop {
			w.writeln()
		}
		if w.isTop {
			w.writeln(currIndent, "Flags:")
		}
		var (
			maxFlagInfoLen = w.maxFlagInfoLen(f)
			nextFlagIndent = w.nextIndent(flagIndent)
		)
		for i := range f.flags {
			flag := &f.flags[i]

			w.writeFlagInfo(flagIndent, flag, false, flag.Arglist, maxFlagInfoLen)
			w.writeFlagValueInfo(flag)
			w.writeln()
			w.writeLines(nextFlagIndent, flag.descLines)
		}
	}

	if subsetCount > 0 {
		if w.isTop || descLineCount > 0 || flagCount > 0 {
			w.writeln()
		}
		if w.isTop {
			w.writeln(currIndent, "Commands:")
		}
		var (
			maxSubsetLen = w.maxSubsetInfoLen(f, !outline)
			subsetIndent = flagIndent
		)
		for i := range f.subsets {
			set := &f.subsets[i]

			nw := writer{
				buf:           w.buf,
				inheritIndent: subsetIndent,
				forceVerbose:  w.forceVerbose,
				maxInfoLen:    maxSubsetLen,
			}
			nw.writeFlagSet(set)
		}
	}
}
