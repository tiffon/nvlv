package gdb

import (
	"fmt"
	"strings"
)

const (
	NATURE_RESULT       = '^'
	NATURE_EXEC_OUT     = '*'
	NATURE_STATUS       = '+'
	NATURE_NOTIFY       = '='
	NATURE_CONSOLE_STRM = '~'
	NATURE_TARGET_STRM  = '@'
	NATURE_LOG_STRM     = '&'
)

// Holds state for the parsing process.
type parser struct {
	s   string // the source text
	i   int    // the current position
	len int    // string should not change
}

// Parses output from the gdb/mi interface into an array of 
// *Record structs. It is assumed earch record will be on its
// own line. If a line starts with "(gdb)" it is ignored. If there is
// an error while parsing a line, the record for that line is the 
// result of parsing prior to the error and the ParseError value
// on the record is set to error message.
//
// output ==> ( out-of-band-record )* [ result-record ] "(gdb)" nl
//
// Format details: http://sourceware.org/gdb/onlinedocs/gdb/GDB_002fMI-Output-Syntax.html#GDB_002fMI-Output-Syntax
func ParseGdbOutput(s string) []*Record {

	lines := strings.Split(s, "\n")
	records := make([]*Record, 0, len(lines))

	for _, raw := range lines {

		raw = strings.TrimSpace(raw)
		if len(raw) == 0 {
			continue
		}
		if len(raw) > 4 && raw[0:5] == "(gdb)" {
			continue
		}

		r, err := ParseGdbRecord(raw)
		if err != nil {
			// let the client know
			if r == nil {
				r = &Record{}
			}
			r.ParseError = err.Error()
			r.ErrorData = raw
		}
		records = append(records, r)
	}

	return records
}

// Parses a line of output from gdb/mi interface.
func ParseGdbRecord(s string) (*Record, error) {

	p := &parser{s, 0, len(s)}
	msg := new(Record)
	msg.Token = p.parseDigits()

	if p.i >= p.len {
		return msg, p.errEOF("record", "record nature character (^,*,+,=,~,@ or &")
	}

	switch p.s[p.i] {

	case '^', '*', '+', '=':
		msg.Nature = p.s[p.i]
		p.i++
		msg.Class = p.parseWord()

		if len(msg.Class) < 1 {
			return msg, p.err("result-record or async-record", "result-class or async-class")
		}
		if p.i >= p.len {
			return msg, nil
		}
		if !p.consumeComma() {
			return msg, p.err("result-record or async-record", "','")
		}

		var err error
		msg.Data, err = p.parseRecordResults()
		return msg, err

	case '~', '@', '&':
		msg.Nature = p.s[p.i]
		p.i++

		if p.i >= p.len {
			p.errEOF("stream", "\"")
		}
		if p.s[p.i] != '"' {
			return msg, p.err("stream", "'\"'")
		}

		p.i++

		if p.i >= p.len {
			return msg, p.errEOF("stream", "a character or terminating '\"'")
		}
		if p.s[p.len-1] != '"' {
			return msg, p.err("stream", "terminating '\"'")
		}

		msg.Stream = p.s[p.i : p.len-1]
		return msg, nil

	default:
		return msg, p.err("record", "record nature character")
	}

	panic("unreachable")
}

// Parses one or more name / value pairs and expects to reach the
// end of the string being parsed. Pratically, this is 'result' portion, if
// present, for result records or async records.
//
// result ( "," result )*
func (p *parser) parseRecordResults() (map[string]interface{}, error) {

	data := make(map[string]interface{})

	for {
		nm, v, err := p.parseResult()
		if err != nil {
			return data, err
		}

		data[nm] = v
		if p.i >= p.len {
			return data, nil
		}

		if !p.consumeComma() {
			return data, p.err("msg results", ",")
		}
	}
	panic("unreachable")
}

// Returns a name / value pair and advances the current position
// past the last character of the name / value pair.
//
// result ==> variable "=" value 
func (p *parser) parseResult() (name string, value interface{}, err error) {

	name = p.parseWord()

	if len(name) == 0 {
		return "", nil, p.err("result", "name")
	}

	if !p.consumeEq() {
		return "", nil, p.err("result", "=")
	}

	value, err = p.parseValue()
	return name, value, err
}

// Returns a either a string, tuple or list and advances the current
// position past the last character of the value. The type of value
// returned is determined by the first character of the value: 
// '"', '{' or '[', for string, tuple and list, respectively.
//
// value ==> const | tuple | list 
// const ==> c-string 
func (p *parser) parseValue() (interface{}, error) {

	switch p.s[p.i] {

	case '"':
		v, err := p.parseString()
		return v, err

	case '{':
		v, err := p.parseTuple()
		return v, err

	case '[':
		v, err := p.parseList()
		return v, err
	}

	return nil, p.err("value", ` '"', '{' or ']'`)
}

// Returns the string from the current character, which should be
// a leading '"', to a terminating '"' and advances the current
// position past the terminating '"'. Ignores any character 
// preceded by the '\\' character.
//
// const ==> c-string 
func (p *parser) parseString() (string, error) {

	if p.s[p.i] != '"' {
		return "", p.err("string", "\"")
	}

	p.i++
	start, i := p.i, p.i

	for {
		switch {
		case i >= p.len:
			return "", p.errEOF("string", "a character or terminating '\"'")

		case p.s[i] == '\\':
			i += 2
			continue

		case p.s[i] == '"':
			p.i = i + 1
			return p.s[start:i], nil

		default:
			i++
		}
	}
	p.i = i
	return p.s[start:i], p.err("string", "\"")
}

// Parses a tuple and advances the current position past the last
// character of the tuple.
//
// tuple ==> "{}" | "{" result ( "," result )* "}" 
func (p *parser) parseTuple() (map[string]interface{}, error) {

	if p.s[p.i] != '{' {
		return nil, p.err("tuple", "{")
	}

	p.i++
	tuple := make(map[string]interface{})

	for {

		nm, v, err := p.parseResult()
		if err != nil {
			return tuple, err
		}

		tuple[nm] = v

		if p.i >= p.len {
			return tuple, p.errEOF("tuple", "comma or '}'")
		}

		if !p.consumeComma() {
			break
		}
	}

	if p.s[p.i] != '}' {
		return tuple, p.err("tuple", "}")
	}

	p.i++
	return tuple, nil
}

// Parses a list and advances the current position past the last
// character of the list.
//
// list ==> "[]" | "[" value ( "," value )* "]" | "[" result ( "," result )* "]" 
func (p *parser) parseList() ([]interface{}, error) {

	if p.s[p.i] != '[' {
		return nil, p.err("list", "[")
	}

	p.i++
	list := make([]interface{}, 0)
	c := p.s[p.i]

	switch {
	// list of values
	case c == '"' || c == '[' || c == '{':
		for {
			v, err := p.parseValue()
			if err != nil {
				return list, err
			}
			list = append(list, v)
			if p.i >= p.len {
				return list, p.errEOF("list", "comma or ']'")
			}
			if !p.consumeComma() {
				break
			}
		}

	// list of results
	case 'a' <= c && c <= 'z' || c == '-':
		for {
			nm, v, err := p.parseResult()
			if err != nil {
				return list, err
			}
			nv := NamedValue{nm, v}
			list = append(list, nv)

			if p.i >= p.len {
				return list, p.errEOF("list", "comma or ']'")
			}
			if !p.consumeComma() {
				break
			}
		}
	}

	if p.s[p.i] != ']' {
		return list, p.err("list", "]")
	}

	p.i++
	return list, nil
}

// Returns portion of string that is sequential letters
// or '-', starting from current position.
func (p *parser) parseWord() string {

	start, i := p.i, p.i

	for i < p.len && ('a' <= p.s[i] && p.s[i] <= 'z' || 'A' <= p.s[i] && p.s[i] <= 'Z' || p.s[i] == '-' || p.s[i] == '_') {
		i++
	}
	if i == start {
		return ""
	}
	p.i = i
	return p.s[start:i]

}

// Returns portion of string that is sequential digits, 
// starting from current position.
func (p *parser) parseDigits() string {

	start, i := p.i, p.i
	max := len(p.s)

	for i < max && '0' <= p.s[i] && p.s[i] <= '9' {
		i++
	}
	if i == start {
		return ""
	}
	p.i = i
	return p.s[start:i]
}

// Returns true and advances the position if current character is a ','.
func (p *parser) consumeComma() bool {

	if p.i < len(p.s) && p.s[p.i] == ',' {
		p.i++
		return true
	}
	return false
}

// Returns true and advances the position if current character is a '='.
func (p *parser) consumeEq() bool {

	if p.i < len(p.s) && p.s[p.i] == '=' {
		p.i++
		return true
	}
	return false
}

// Returns an informative based on the element type being parsed, 
// what type of string was expected, and the current position and
// character of the parser.
func (p *parser) err(elmType, expected string) error {
	return fmt.Errorf("malformed %s, expected %s, found %c at %d", elmType, expected, p.s[p.i], p.i)
}

// Returns an informative based on the element type being parsed 
// and what type of string was expected.
func (p *parser) errEOF(elmType, expected string) error {
	return fmt.Errorf("malformed %s, expected %s, found EOF", elmType, expected)
}

var testMsgs []string = []string{
	`=thread-group-added,id="i1"
~"GNU gdb (GDB) 7.5\n"
~"Copyright (C) 2012 Free Software Foundation, Inc.\n"
~"License GPLv3+: GNU GPL version 3 or later <http://gnu.org/licenses/gpl.html>\nThis is free software: you are free to change and redistribute it.\nThere is NO WARRANTY, to the extent permitted by law.  Type \"show copying\"\nand \"show warranty\" for details.\n"
~"This GDB was configured as \"x86_64-apple-darwin12.1.0\".\nFor bug reporting instructions, please see:\n"
~"<http://www.gnu.org/software/gdb/bugs/>...\n"
~"Reading symbols from /_projects/_self/go/projects/src/bitbucket.org/_joe/nvlv/cmd/dev_0..."
~"done.\n"
(gdb) `,
	`&"source /usr/local/go/src/pkg/runtime/runtime-gdb.py\n"
&"Loading Go Runtime support.\n"
^done
(gdb) `,
	`0^done,bkpt={number="1",type="breakpoint",disp="keep",enabled="y",addr="0x00000000000023c5",func="main.main",file="/_projects/_self/go/projects/src/bitbucket.org/_joe/nvlv/cmd/dev_0.go",fullname="/_projects/_self/go/projects/src/bitbucket.org/_joe/nvlv/cmd/dev_0.go",line="33",times="0",original-location="main.main"}
(gdb) `,
	`1^done,bkpt={number="2",type="breakpoint",disp="keep",enabled="y",addr="0x0000000000002a81",func="main.outsideFunc",file="/_projects/_self/go/projects/src/bitbucket.org/_joe/nvlv/cmd/dev_0.go",fullname="/_projects/_self/go/projects/src/bitbucket.org/_joe/nvlv/cmd/dev_0.go",line="120",times="0",original-location="main.outsideFunc"}
(gdb) `,
	`=thread-group-started,id="i1",pid="6425"
=thread-created,id="1",group-id="i1"
2^running
*running,thread-id="all"
(gdb) `,
	`=thread-created,id="2",group-id="i1"
~"[New Thread 0x1903 of process 6425]\n"
=breakpoint-modified,bkpt={number="1",type="breakpoint",disp="keep",enabled="y",addr="0x00000000000023c5",func="main.main",file="/_projects/_self/go/projects/src/bitbucket.org/_joe/nvlv/cmd/dev_0.go",fullname="/_projects/_self/go/projects/src/bitbucket.org/_joe/nvlv/cmd/dev_0.go",line="33",times="1",original-location="main.main"}
~"[Switching to Thread 0x1903 of process 6425]\n"
*stopped,reason="breakpoint-hit",disp="keep",bkptno="1",frame={addr="0x00000000000023c5",func="main.main",args=[],file="/_projects/_self/go/projects/src/bitbucket.org/_joe/nvlv/cmd/dev_0.go",fullname="/_projects/_self/go/projects/src/bitbucket.org/_joe/nvlv/cmd/dev_0.go",line="33"},thread-id="2",stopped-threads="all"
(gdb) `,
	`3^running
*running,thread-id="2"
(gdb) `,
	`*running,thread-id="all"
*stopped,reason="end-stepping-range",frame={addr="0x00000000000023e7",func="main.main",args=[],file="/_projects/_self/go/projects/src/bitbucket.org/_joe/nvlv/cmd/dev_0.go",fullname="/_projects/_self/go/projects/src/bitbucket.org/_joe/nvlv/cmd/dev_0.go",line="35"},thread-id="2",stopped-threads="all"
(gdb) `,
	`^done,threads=[{id="2",target-id="Thread 0x1903 of process 6425",frame={level="0",addr="0x00000000000023e7",func="main.main",args=[],file="/_projects/_self/go/projects/src/bitbucket.org/_joe/nvlv/cmd/dev_0.go",fullname="/_projects/_self/go/projects/src/bitbucket.org/_joe/nvlv/cmd/dev_0.go",line="35"},state="stopped"},{id="1",target-id="Thread 0x1703 of process 6425",frame={level="0",addr="0x0000000000018c21",func="runtime.mach_semaphore_timedwait",args=[],file="/private/tmp/bindist454984655/go/src/pkg/runtime/sys_darwin_amd64.s",line="321"},state="stopped"}],current-thread-id="2"
(gdb) `,
	`4^done,stack=[frame={level="0",addr="0x00000000000023e7",func="main.main",file="/_projects/_self/go/projects/src/bitbucket.org/_joe/nvlv/cmd/dev_0.go",fullname="/_projects/_self/go/projects/src/bitbucket.org/_joe/nvlv/cmd/dev_0.go",line="35"},frame={level="1",addr="0x000000000000f210",func="runtime.main",file="/private/tmp/bindist454984655/go/src/pkg/runtime/proc.c",line="244"},frame={level="2",addr="0x000000000000f2b3",func="schedunlock",file="/private/tmp/bindist454984655/go/src/pkg/runtime/proc.c",line="267"},frame={level="3",addr="0x0000000000000000",func="??"}]
(gdb) `,
}

var testRecords map[string]string = map[string]string{
	"rawResultDone": `99^done,stack=[frame={level="0",addr="0x00000000000023c5",func="main.main",file="/_projects/_self/go/projects/src/bitbucket.org/_joe/nvlv/cmd/dev_0.go",fullname="/_projects/_self/go/projects/src/bitbucket.org/_joe/nvlv/cmd/dev_0.go",line="33"},frame={level="1",addr="0x000000000000f210",func="runtime.main",file="/private/tmp/bindist454984655/go/src/pkg/runtime/proc.c",line="244"},frame={level="2",addr="0x000000000000f2b3",func="schedunlock",file="/private/tmp/bindist454984655/go/src/pkg/runtime/proc.c",line="267"},frame={level="3",addr="0x0000000000000000",func="??"}]`,

	"rawResultStopped": `*stopped,reason="breakpoint-hit",disp="keep",bkptno="1",frame={addr="0x00000000000023c5",func="main.main",args=[],file="/_projects/_self/go/projects/src/bitbucket.org/_joe/nvlv/cmd/dev_0.go",fullname="/_projects/_self/go/projects/src/bitbucket.org/_joe/nvlv/cmd/dev_0.go",line="33"},thread-id="2",stopped-threads="all"`,
}

/* output format: http://sourceware.org/gdb/onlinedocs/gdb/GDB_002fMI-Output-Syntax.html#GDB_002fMI-Output-Syntax

output ==>
( out-of-band-record )* [ result-record ] "(gdb)" nl 
result-record ==>
[ token ] "^" result-class ( "," result )* nl 
out-of-band-record ==>
async-record | stream-record 
async-record ==>
exec-async-output | status-async-output | notify-async-output 
exec-async-output ==>
[ token ] "*" async-output 
status-async-output ==>
[ token ] "+" async-output 
notify-async-output ==>
[ token ] "=" async-output 
async-output ==>
async-class ( "," result )* nl 
result-class ==>
"done" | "running" | "connected" | "error" | "exit" 
async-class ==>
"stopped" | others (where others will be added depending on the needs—this is still in development). 
result ==>
variable "=" value 
variable ==>
string 
value ==>
const | tuple | list 
const ==>
c-string 
tuple ==>
"{}" | "{" result ( "," result )* "}" 
list ==>
"[]" | "[" value ( "," value )* "]" | "[" result ( "," result )* "]" 
stream-record ==>
console-stream-output | target-stream-output | log-stream-output 
console-stream-output ==>
"~" c-string 
target-stream-output ==>
"@" c-string 
log-stream-output ==>
"&" c-string 
nl ==>
CR | CR-LF 
token ==>
any sequence of digits.
Notes:

All output sequences end in a single line containing a period.
The token is from the corresponding request. Note that for all async output, while the token is allowed by the grammar and may be output by future versions of gdb for select async output messages, it is generally omitted. Frontends should treat all async output as reporting general changes in the state of the target and there should be no need to associate async output to any prior command.
status-async-output contains on-going status information about the progress of a slow operation. It can be discarded. All status output is prefixed by `+'.
exec-async-output contains asynchronous state change on the target (stopped, started, disappeared). All async output is prefixed by `*'.
notify-async-output contains supplementary information that the client should handle (e.g., a new breakpoint information). All notify output is prefixed by `='.
console-stream-output is output that should be displayed as is in the console. It is the textual response to a CLI command. All the console output is prefixed by `~'.
target-stream-output is the output produced by the target program. All the target output is prefixed by `@'.
log-stream-output is output text coming from gdb's internals, for instance messages that should be displayed as part of an error log. All the log output is prefixed by `&'.
New gdb/mi commands should only output lists containing values.
*/

/* LONGER SESSION:


=thread-group-added,id="i1"
~"GNU gdb (GDB) 7.5\n"
~"Copyright (C) 2012 Free Software Foundation, Inc.\n"
~"License GPLv3+: GNU GPL version 3 or later <http://gnu.org/licenses/gpl.html>\nThis is free software: you are free to change and redistribute it.\nThere is NO WARRANTY, to the extent permitted by law.  Type \"show copying\"\nand \"show warranty\" for details.\n"
~"This GDB was configured as \"x86_64-apple-darwin12.1.0\".\nFor bug reporting instructions, please see:\n"
~"<http://www.gnu.org/software/gdb/bugs/>...\n"
~"Reading symbols from /_projects/_self/go/projects/src/bitbucket.org/_joe/nvlv/cmd/dev_0..."
~"done.\n"
(gdb) 
&"source /usr/local/go/src/pkg/runtime/runtime-gdb.py\n"
&"Loading Go Runtime support.\n"
^done
(gdb) 
0^done,bkpt={number="1",type="breakpoint",disp="keep",enabled="y",addr="0x00000000000023c5",func="main.main",file="/_projects/_self/go/projects/src/bitbucket.org/_joe/nvlv/cmd/dev_0.go",fullname="/_projects/_self/go/projects/src/bitbucket.org/_joe/nvlv/cmd/dev_0.go",line="33",times="0",original-location="main.main"}
(gdb) 
1^done,bkpt={number="2",type="breakpoint",disp="keep",enabled="y",addr="0x0000000000002a81",func="main.outsideFunc",file="/_projects/_self/go/projects/src/bitbucket.org/_joe/nvlv/cmd/dev_0.go",fullname="/_projects/_self/go/projects/src/bitbucket.org/_joe/nvlv/cmd/dev_0.go",line="120",times="0",original-location="main.outsideFunc"}
(gdb) 
=thread-group-started,id="i1",pid="6425"
=thread-created,id="1",group-id="i1"
2^running
*running,thread-id="all"
(gdb) 
=thread-created,id="2",group-id="i1"
~"[New Thread 0x1903 of process 6425]\n"
=breakpoint-modified,bkpt={number="1",type="breakpoint",disp="keep",enabled="y",addr="0x00000000000023c5",func="main.main",file="/_projects/_self/go/projects/src/bitbucket.org/_joe/nvlv/cmd/dev_0.go",fullname="/_projects/_self/go/projects/src/bitbucket.org/_joe/nvlv/cmd/dev_0.go",line="33",times="1",original-location="main.main"}
~"[Switching to Thread 0x1903 of process 6425]\n"
*stopped,reason="breakpoint-hit",disp="keep",bkptno="1",frame={addr="0x00000000000023c5",func="main.main",args=[],file="/_projects/_self/go/projects/src/bitbucket.org/_joe/nvlv/cmd/dev_0.go",fullname="/_projects/_self/go/projects/src/bitbucket.org/_joe/nvlv/cmd/dev_0.go",line="33"},thread-id="2",stopped-threads="all"
(gdb) 
3^running
*running,thread-id="2"
(gdb) 
*running,thread-id="all"
*stopped,reason="end-stepping-range",frame={addr="0x00000000000023e7",func="main.main",args=[],file="/_projects/_self/go/projects/src/bitbucket.org/_joe/nvlv/cmd/dev_0.go",fullname="/_projects/_self/go/projects/src/bitbucket.org/_joe/nvlv/cmd/dev_0.go",line="35"},thread-id="2",stopped-threads="all"
(gdb) 
^done,threads=[{id="2",target-id="Thread 0x1903 of process 6425",frame={level="0",addr="0x00000000000023e7",func="main.main",args=[],file="/_projects/_self/go/projects/src/bitbucket.org/_joe/nvlv/cmd/dev_0.go",fullname="/_projects/_self/go/projects/src/bitbucket.org/_joe/nvlv/cmd/dev_0.go",line="35"},state="stopped"},{id="1",target-id="Thread 0x1703 of process 6425",frame={level="0",addr="0x0000000000018c21",func="runtime.mach_semaphore_timedwait",args=[],file="/private/tmp/bindist454984655/go/src/pkg/runtime/sys_darwin_amd64.s",line="321"},state="stopped"}],current-thread-id="2"
(gdb) 
4^done,stack=[frame={level="0",addr="0x00000000000023e7",func="main.main",file="/_projects/_self/go/projects/src/bitbucket.org/_joe/nvlv/cmd/dev_0.go",fullname="/_projects/_self/go/projects/src/bitbucket.org/_joe/nvlv/cmd/dev_0.go",line="35"},frame={level="1",addr="0x000000000000f210",func="runtime.main",file="/private/tmp/bindist454984655/go/src/pkg/runtime/proc.c",line="244"},frame={level="2",addr="0x000000000000f2b3",func="schedunlock",file="/private/tmp/bindist454984655/go/src/pkg/runtime/proc.c",line="267"},frame={level="3",addr="0x0000000000000000",func="??"}]
(gdb) 


*/
