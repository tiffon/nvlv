package gdb

// Used to represent a name / value pair in a list, for example:
// [frame={details="abridged"}]
type NamedValue struct {
	Name string
	Data interface{}
}

// A parsed line of output from the GDB that should not be modified 
// for any reason.
type Record struct {
	Token      string
	Nature     byte
	Class      string
	Data       map[string]interface{}
	Stream     string
	ParseError string
	ErrorData  interface{}
}

func ExecState(recs []*Record) (class, reason string, found bool) {
	for _, r := range recs {
		if r.Nature == NATURE_EXEC_OUT {
			class = r.Class
			reason, _ = r.Data["reason"].(string)
			found = true
			return
		}
	}
	return
}

func HasToken(recs []*Record, token string) (index int, found bool) {
	for i, r := range recs {
		if r.Token == token {
			return i, true
		}
	}
	return
}
