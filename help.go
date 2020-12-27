package flag

import (
	"fmt"
	"strings"
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

func (w *helpWriter) writeTopCommandInfo(currIndent string, f *FlagSet, normal, positional []*Flag) {
	var arglist string
	switch f.self.Arglist {
	case "-":
	default:
		arglist = f.self.Arglist
	case "":
		var sb strings.Builder

		flagCount, cmdCount := len(normal), len(f.subsets)
		if flagCount != 0 {
			if cmdCount != 0 {
				sb.WriteString("[FLAG|COMMAND]...")
			} else {
				sb.WriteString("[FLAG]...")
			}
		} else {
			if cmdCount != 0 {
				sb.WriteString("[COMMAND]...")
			}
		}
		if len(positional) > 0 || f.self.ArgsPtr != nil {
			for _, p := range positional {
				if sb.Len() > 0 {
					sb.WriteString(" ")
				}
				sb.WriteString("[")
				sb.WriteString(p.Arglist)
				sb.WriteString("]")
			}
			if f.self.ArgsPtr != nil {
				sb.WriteString(" [ARG]...")
			}
		}
		arglist = sb.String()
	}
	if f.self.Usage != "" {
		w.writeln(currIndent, f.self.Usage)
		w.writeln()
	}
	w.writeln(currIndent, "Usage: ", f.self.Names+" "+arglist)
}

func (w *helpWriter) writeChildInfo(currIndent string, flag *Flag, isCommand bool) {
	w.write(currIndent)
	var info string
	if !isCommand {
		if flag.Names == flagNamePositional {
			info = flagNamePositional + flag.Arglist
		} else if flag.Arglist != "" {
			if isBoolPtr(flag.Ptr) {
				info = flag.Names + "=" + flag.Arglist
			} else {
				info = flag.Names + " " + flag.Arglist
			}
		} else {
			info = flag.Names
		}
	} else {
		info = flag.Names
	}
	w.write(info)
	if flag.Usage != "" {
		w.write("\t", flag.Usage)
	} else {
		w.write("\t")
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

func (w *helpWriter) writeCommand(f *FlagSet) {
	var childIndent = w.nextIndent(w.indent)

	var normalFlags, positionalFlags []*Flag
	for i := range f.flags {
		f := &f.flags[i]
		if f.Names == "@" {
			positionalFlags = append(positionalFlags, f)
		} else {
			normalFlags = append(normalFlags, f)
		}
	}
	if w.isTop {
		w.writeTopCommandInfo(w.indent, f, normalFlags, positionalFlags)
	} else {
		w.writeChildInfo(w.indent, &f.self, true)
	}
	if !w.isTop {
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
		w.writeln()
		w.writeln(w.indent, "Flags:")
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

			w.writeChildInfo(childIndent, &set.self, true)
		}
	}
}
