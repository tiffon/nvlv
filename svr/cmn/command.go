package cmn

import (
	"bufio"
	"errors"
	"io"
	"os/exec"
)

type CmdMsg struct {
	Msg string
	Err error
}

type CmdWrapper struct {
	cmd     *exec.Cmd
	inchan  chan string
	outchan chan *CmdMsg
	errchan chan *CmdMsg
	started bool
	killed  bool
}

func NewCmdWrapper(cmd *exec.Cmd) *CmdWrapper {
	return &CmdWrapper{
		cmd,
		make(chan string),
		make(chan *CmdMsg),
		make(chan *CmdMsg),
		false,
		false,
	}
}

func (c *CmdWrapper) Cmd() *exec.Cmd {
	return c.cmd
}

func (c *CmdWrapper) InChan() chan<- string {
	return c.inchan
}

func (c *CmdWrapper) OutChan() <-chan *CmdMsg {
	return c.outchan
}

func (c *CmdWrapper) ErrChan() <-chan *CmdMsg {
	return c.errchan
}

func (c *CmdWrapper) IsStarted() bool {
	return c.started
}

func (c *CmdWrapper) IsKilled() bool {
	return c.killed
}

func (c *CmdWrapper) Start() error {
	if c.started {
		return errors.New("Cmd already started")
	}
	var (
		inPipe  io.WriteCloser
		outPipe io.ReadCloser
		errPipe io.ReadCloser
		err     error
	)
	if inPipe, err = c.cmd.StdinPipe(); err != nil {
		return err
	}
	if outPipe, err = c.cmd.StdoutPipe(); err != nil {
		return err
	}
	if errPipe, err = c.cmd.StderrPipe(); err != nil {
		return err
	}
	err = c.cmd.Start()
	if err != nil {
		return err
	}
	go pipeReadLoop(outPipe, c.outchan)
	go pipeReadLoop(errPipe, c.errchan)
	go func() {
		for {
			select {
			case s := <-c.inchan:
				inPipe.Write([]byte(s + "\n"))
			}
		}
	}()
	c.started = true
	return nil
}

func (c *CmdWrapper) KillRelease() {
	c.cmd.Process.Kill()
	c.cmd.Process.Release()
	c.killed = true
}

func pipeReadLoop(pipe io.ReadCloser, resutlChan chan *CmdMsg) {
	rdr := bufio.NewReader(pipe)
	line, err := rdr.ReadString('\n')
	for err == nil {
		resutlChan <- &CmdMsg{line, nil}
		line, err = rdr.ReadString('\n')
	}
	resutlChan <- &CmdMsg{line, err}
}
