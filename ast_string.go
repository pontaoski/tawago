package main

import "fmt"

func typeToString(t *Type) string {
	if t == nil {
		return ""
	}

	switch v := (*t).(type) {
	case Ident:
		return v.Name
	}

	panic("unhandled")
}

func (f Func) String() string {
	var args []string
	for _, arg := range f.Arguments {
		args = append(args, typeToString(&arg.Kind))
	}
	return fmt.Sprintf("func() %s;", typeToString(f.Returns))
}
