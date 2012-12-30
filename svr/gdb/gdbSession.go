package gdb

import (
	"errors"
	"fmt"
	"github.com/tiffon/nvlv/svr/cmn"
	"io"
	"os/exec"
	"strings"
	"sync"
	"time"
)

var GdbBinPath string = "/_assets/tools/gdb/gdb-7.5/build/gdb/gdb"

var RuntimeGdbPy string = "/usr/local/go/src/pkg/runtime/runtime-gdb.py"

var ErrIsStarted = errors.New("is started")

var ErrIsKilled = errors.New("is killed")

// type GdbState int

// const (
// 	NOT_STARTED GdbState = iota
// 	RUNNING
// 	STOPPED
// )

type Msg struct {
	Records []*Record
	Raw     string
}

type Ssn struct {
	cmdToken    int
	started     bool
	killed      bool
	stateMtx    *sync.Mutex
	execOutFile string
	gdb         *cmn.CmdWrapper
	tail        *cmn.CmdWrapper
	// Single line strings that are part of current GDB MI output, which is a multiline string. 
	// The output is processed when the closing token is encountered.
	// recentOutput []string
	// won't need backlog until have reactive programming and need a place to store unrelated
	// output records
	// outputBacklog [][]*Record
	errOutput      chan error
	gdbOutput      chan *Msg
	inferiorOutput chan string
	input          chan []string
}

func NewSsn(execOutFile string, onErr chan error) *Ssn {
	var tail *cmn.CmdWrapper = nil
	if len(execOutFile) > 0 {
		tail = cmn.NewCmdWrapper(exec.Command("tail", "-f", execOutFile))
	}
	return &Ssn{
		499,
		false,
		false,
		&sync.Mutex{},
		execOutFile,
		cmn.NewCmdWrapper(exec.Command("bash")),
		tail,
		// make([]string, 0, 11),
		// make([][]*Record, 0),
		onErr,
		make(chan *Msg),
		make(chan string),
		make(chan []string),
	}
}

func (ssn *Ssn) NewCmdToken() string {
	ssn.cmdToken++
	// return string(ssn.cmdToken)
	return fmt.Sprintf("%d", ssn.cmdToken)
}

func (ssn *Ssn) IsStarted() bool {
	return ssn.started
}

func (ssn *Ssn) IsKilled() bool {
	return ssn.killed
}

func (ssn *Ssn) ErrOutput() <-chan error {
	return ssn.errOutput
}

func (ssn *Ssn) GdbOutput() <-chan *Msg {
	return ssn.gdbOutput
}

func (ssn *Ssn) InferiorOutput() <-chan string {
	return ssn.inferiorOutput
}

func (ssn *Ssn) Input() chan<- []string {
	return ssn.input
}

func (ssn *Ssn) Start(fileExec string, args ...string) error {
	ssn.stateMtx.Lock()
	defer ssn.stateMtx.Unlock()

	if ssn.started {
		return ErrIsStarted
	}
	if ssn.killed {
		return ErrIsKilled
	}
	ssn.started = true

	err := ssn.gdb.Start()
	if err != nil {
		return err
	}

	cmdStr := GdbBinPath + " --interpreter mi " + fileExec
	gin := ssn.gdb.InChan()
	gin <- cmdStr
	gin <- "source " + RuntimeGdbPy
	for _, arg := range args {
		gin <- arg
	}

	go ssn.readTail()
	go ssn.gdbIoLoop()

	return nil
}

func (ssn *Ssn) Run() {
	go func(ssn *Ssn) {
		ssn.input <- []string{"run > " + ssn.execOutFile}
	}(ssn)
}

// Issues a command to the GDB and returns the response. A token is generated and 
// prepended to the cmd which how the response is identified. This is a synchronous 
// call that blocks while processing output from the GDB. 
func (ssn *Ssn) GetResponse(cmd string, d time.Duration) (idx int, resp *Msg, skipped [][]*Record, err error) {

	skipped = make([][]*Record, 0)
	token := ssn.NewCmdToken()
	cmd = token + cmd
	timeout := time.After(d)
	out := ssn.GdbOutput()
	ssn.Input() <- []string{cmd}

	for {
		select {

		case r := <-out:
			if i, ok := HasToken(r.Records, token); ok {
				return i, r, skipped, err
			} else {
				skipped = append(skipped, r.Records)
			}

		case <-timeout:
			err = errors.New("gdb: timeout")
			return
		}
	}
	panic("unreachable")
}

func (ssn *Ssn) GetFrameVars(threadId, frameLvl string, timeout time.Duration) (fVars []interface{}, skipped [][]*Record, err error) {

	cmd := fmt.Sprintf("-stack-list-variables --thread %s --frame %s --all-values", threadId, frameLvl)
	i, resp, skipped, err := ssn.GetResponse(cmd, timeout)
	if err != nil {
		return
	}

	fVars, ok := resp.Records[i].Data["variables"].([]interface{})
	if !ok {
		err = fmt.Errorf("unknown frame-vars type: %t", resp.Records[i].Data["variables"])
	}
	return
}

func (ssn *Ssn) gdbIoLoop() {

	getInput := ssn.input
	sendInput := ssn.gdb.InChan()
	getOutput := ssn.gdb.OutChan()
	sendOutput := ssn.gdbOutput
	getErr := ssn.gdb.ErrChan()
	recentMsgs := make([]string, 0, 11)

	var err error = nil
	for err == nil {
		select {
		case xs, ok := <-getInput:
			if !ok {
				err = errors.New("input channel closed")
				break
			}
			for _, s := range xs {
				sendInput <- s
			}

		case m := <-getOutput:
			if m.Err != nil {
				err = m.Err
				break
			}
			recentMsgs = append(recentMsgs, m.Msg)
			if strings.Contains(m.Msg, "(gdb)") {
				rawGdbOut := strings.Join(recentMsgs, "")
				for i := range recentMsgs {
					recentMsgs[i] = ""
				}
				recentMsgs = recentMsgs[:0]
				sendOutput <- &Msg{ParseGdbOutput(rawGdbOut), rawGdbOut}
			}

		case m := <-getErr:
			if len(m.Msg) > 0 {
				err = errors.New("tail: " + m.Msg)
			} else {
				err = m.Err
			}
		}
	}
	if err != nil && err != io.EOF {
		ssn.handleErr(err)
	}
}

func (ssn *Ssn) readTail() {

	ssn.stateMtx.Lock()
	getOutput := ssn.tail.OutChan()
	getError := ssn.tail.ErrChan()
	sendOutput := ssn.inferiorOutput
	ssn.tail.Start()
	ssn.stateMtx.Unlock()

	var err error = nil
	for err == nil {
		select {
		case m := <-getOutput:
			if m.Err != nil {
				err = m.Err
			} else {
				sendOutput <- m.Msg
			}

		case m := <-getError:
			if len(m.Msg) > 0 {
				err = errors.New("tail: " + m.Msg)
			} else {
				err = m.Err
			}
		}
	}
	if err != nil && err != io.EOF {
		ssn.handleErr(err)
	}
}

func (ssn *Ssn) handleErr(err error) {
	ssn.Kill()
	ssn.errOutput <- err
}

func (ssn *Ssn) Kill() {
	ssn.stateMtx.Lock()
	if ssn.gdb.IsStarted() && !ssn.gdb.IsKilled() {
		ssn.gdb.KillRelease()
	}
	if ssn.tail.IsStarted() && !ssn.tail.IsKilled() {
		ssn.tail.KillRelease()
	}
	ssn.killed = true
	ssn.stateMtx.Unlock()
}
