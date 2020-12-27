package flag

import (
	"fmt"
	"text/tabwriter"
)

const (
	minInfoLen = 12
	maxInfoLen = 24
)

type helpWriter struct {
	buf    *tabwriter.Writer
	isTop  bool
	indent string

	maxVerboseLevel  int
	currVerboseLevel int
}

func (w *helpWriter) maxFlagInfoLen(f *FlagSet) int {
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

func (w *helpWriter) nextIndent(curr string) string {
	const indent = "\t"
	return curr + indent
}

func (w *helpWriter) write(elem ...string) {
	for _, s := range elem {
		w.buf.Write([]byte(s))
	}
}

func (w *helpWriter) writeln(elem ...string) {
	w.write(elem...)
	w.buf.Write([]byte{'\n'})
}

func (w *helpWriter) writeLines(indent string, lines []string) {
	for _, line := range lines {
		w.writeln(indent, line)
	}
}

func (w *helpWriter) writeTopCommandInfo(currIndent string, f *FlagSet) {
	var arglist string
	switch f.self.Arglist {
	case "-":
	default:
		arglist = f.self.Arglist
	case "":
		flagCount, cmdCount := len(f.flags), len(f.subsets)
		if flagCount != 0 {
			if cmdCount != 0 {
				arglist = "[FLAG|COMMAND]..."
			} else {
				arglist = "[FLAG]..."
			}
		} else {
			if cmdCount != 0 {
				arglist = "[COMMAND]..."
			}
		}
	}
	if f.self.Usage != "" {
		w.writeln(currIndent, f.self.Usage)
		w.writeln()
	}
	w.writeln(currIndent, "Usage: ", f.self.Names+" "+arglist)
}

func (w *helpWriter) writeChildInfo(currIndent string, flag *Flag, isCommand bool) {
	w.write(currIndent)
	info := flag.Names
	if !isCommand {
		if flag.Arglist != "" && flag.Arglist != "-" {
			info += " " + flag.Arglist
		}
	}
	w.write(info)
	if flag.Usage != "" {
		w.write("\t", flag.Usage)
	}
	if !isCommand {
		w.write("\t")
		w.writeFlagValueInfo(flag)
	}
	w.write("\n")
}

func (w *helpWriter) writeFlagValueInfo(flag *Flag) {
	w.write("(")
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

func (w *helpWriter) verbose() bool {
	return w.maxVerboseLevel < 0 || w.currVerboseLevel < w.maxVerboseLevel
}

func (w *helpWriter) writeCommand(f *FlagSet) {
	var childIndent = w.nextIndent(w.indent)
	if w.isTop {
		w.writeTopCommandInfo(w.indent, f)
	} else {
		w.writeChildInfo(w.indent, &f.self, true)
	}
	if !w.isTop && !w.verbose() {
		return
	}
	if w.isTop {
		if len(f.self.versionLines) > 0 {
			w.writeln()
			w.writeln(w.indent, "Version:")
			w.writeLines(childIndent, f.self.versionLines)
		}
		if len(f.self.descLines) > 0 {
			w.writeln()
			w.writeln(w.indent, "Description:")
			w.writeLines(childIndent, f.self.descLines)
		}
	}
	if len(f.flags) > 0 {
		if w.isTop {
			w.writeln()
			w.writeln(w.indent, "Flags:")
		}
		for i := range f.flags {
			flag := &f.flags[i]

			w.writeChildInfo(childIndent, flag, false)
			if len(flag.descLines) > 0 {
				w.writeLines(w.nextIndent(childIndent), flag.descLines)
			}
		}
	}

	if len(f.subsets) > 0 {
		if w.isTop {
			w.writeln()
			w.writeln(w.indent, "Commands:")
		}
		for i := range f.subsets {
			set := &f.subsets[i]

			nw := helpWriter{
				buf:              w.buf,
				indent:           childIndent,
				currVerboseLevel: w.currVerboseLevel,
				maxVerboseLevel:  w.maxVerboseLevel,
			}
			if !w.isTop {
				nw.currVerboseLevel++
			}
			nw.writeCommand(set)
		}
	}
}
