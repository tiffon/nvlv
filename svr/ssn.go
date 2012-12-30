package svr

import (
	"code.google.com/p/go.net/websocket"
	"fmt"
	"github.com/tiffon/nvlv/svr/cmn"
	"github.com/tiffon/nvlv/svr/gdb"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
)

type nvlvSsn struct {
	dir           string
	ws            *websocket.Conn
	shMsgBody     *clientBody
	exeMsgBody    *clientBody
	gdbMsgBody    *clientBody
	cmdMsgBody    *clientBody
	shCmd         *cmn.CmdWrapper
	gdbSsn        *gdb.Ssn
	gdbErr        <-chan error
	gdbExecOut    string
	lastGdbRecs   []*gdb.Record
	msgFromClient chan *clientMsg
	ssnErr        chan error
}

func newNvlvSsn(ws *websocket.Conn) (*nvlvSsn, error) {
	var err error
	ssn := &nvlvSsn{}

	ssn.dir, ssn.gdbExecOut, err = getSsnSpace()
	if err != nil {
		log.Println("Err: Unable to create session storage locations: ", err)
		return nil, err
	}

	ssn.ws = ws
	ssn.ssnErr = make(chan error)
	return ssn, nil
}

func (ssn *nvlvSsn) SsnErr() <-chan error {
	return ssn.ssnErr
}

func (ssn *nvlvSsn) Run() error {

	var err error

	ssn.shMsgBody = &clientBody{
		"sh",
		make(map[string]interface{}),
	}
	ssn.exeMsgBody = &clientBody{
		"exe",
		make(map[string]interface{}),
	}
	ssn.gdbMsgBody = &clientBody{
		"gdb",
		make(map[string]interface{}),
	}
	ssn.cmdMsgBody = &clientBody{
		"cmd",
		make(map[string]interface{}),
	}

	gdbErrChan := make(chan error)
	ssn.gdbErr = gdbErrChan
	ssn.gdbSsn = gdb.NewSsn(ssn.gdbExecOut, gdbErrChan)
	gdbInferiorOut := ssn.gdbSsn.InferiorOutput()
	gdbOutput := ssn.gdbSsn.GdbOutput()

	ssn.shCmd = cmn.NewCmdWrapper(exec.Command("bash"))
	if err = ssn.shCmd.Start(); err != nil {
		log.Println("Err: Unable to start shell command for nvlv session: ", err)
		return err
	}

	ssn.msgFromClient = make(chan *clientMsg)
	go recvJsonLoop(ssn.ws, ssn.msgFromClient)

	for {
		select {

		case m := <-ssn.shCmd.OutChan():
			if isReadErr(m.Err, "sh out stream") {
				err = m.Err
				goto endConn
			}
			ssn.shMsgBody.sendMsg(ssn.ws, m.Msg)

		case m := <-ssn.shCmd.ErrChan():
			if isReadErr(m.Err, "sh err stream") {
				err = m.Err
				goto endConn
			}
			ssn.shMsgBody.sendErr(ssn.ws, m.Msg)

		case err = <-gdbErrChan:
			log.Println("gbd err: ", err)
			ssn.gdbMsgBody.sendErr(ssn.ws, err.Error())
			goto endConn

		case s := <-gdbInferiorOut:
			ssn.exeMsgBody.sendMsg(ssn.ws, s)

		case gdbMsg := <-gdbOutput:
			ssn.lastGdbRecs = gdbMsg.Records
			ssn.gdbMsgBody.sendData(ssn.ws, gdbMsg.Records, "raw", gdbMsg.Raw)

		case m := <-ssn.msgFromClient:
			if isReadErr(m.err, "client") {
				err = m.err
				goto endConn
			}
			log.Println("msg from client:", m.data)
			err = handleMsg(ssn, m)
		}
	}

endConn:

	log.Println("Killing cmds")

	if ssn.shCmd.IsStarted() && !ssn.shCmd.IsKilled() {
		ssn.shCmd.InChan() <- "exit"
		ssn.shCmd.KillRelease()
	}
	ssn.gdbSsn.Kill()

	return err
}

func handleMsg(ssn *nvlvSsn, msg *clientMsg) error {

	switch msg.data.Ctx {

	case "sh":
		if s, ok := msg.data.Data["cmd"].(string); ok {
			ssn.shCmd.InChan() <- s
		} else {
			ssn.shMsgBody.sendErr(ssn.ws, `unable to cast Data["cmd"] to string`)
		}

	case "gdb":
		if !ssn.gdbSsn.IsStarted() {
			ssn.gdbMsgBody.sendErr(ssn.ws, `The gdb process is not started.`)
			break
		}
		if ssn.gdbSsn.IsKilled() {
			ssn.gdbMsgBody.sendErr(ssn.ws, `The gdb process has been killed and must be restarted.`)
			break
		}
		if s, ok := msg.data.Data["cmd"].(string); ok {
			ssn.gdbSsn.Input() <- []string{s}
		} else {
			s = fmt.Sprintf("Unrecognized cmd value: %v\n", msg.data.Data["cmd"])
			ssn.gdbMsgBody.sendErr(ssn.ws, s)
		}

	case "cmd":
		log.Println("have a command")
		adminCmd, ok := msg.data.Data["cmd"].(string)
		if !ok {
			s := fmt.Sprintf("Unrecognized cmd value: %v\n", msg.data.Data["cmd"])
			ssn.cmdMsgBody.sendErr(ssn.ws, s)
			break
		}
		ssn.cmdMsgBody.sendMsg(ssn.ws, adminCmd)

		switch adminCmd {

		case "-gdb-start":
			log.Println("-gdb-start cmd")

			var execFile string
			if args, ok := msg.data.Data["args"]; ok {

				switch v := args.(type) {
				case []interface{}:
					for _, elm := range v {
						execFile += fmt.Sprintf("%v ", elm)
					}
					execFile = strings.TrimSpace(execFile)

				default:
					execFile = fmt.Sprintf("%v", args)
				}
			}

			if err := ssn.gdbSsn.Start(execFile); err != nil {
				s := fmt.Sprintf("Err: Unable to start gdb ssn: %s", err.Error())
				ssn.cmdMsgBody.sendErr(ssn.ws, s)
				log.Println(s)
			}
			ssn.cmdMsgBody.sendMsg(ssn.ws, "gdb process started")

		case "-gdb-run":
			ssn.gdbSsn.Run()
			ssn.cmdMsgBody.sendMsg(ssn.ws, "gdb run called")

		case "-gdb-get-threads-frames":
			threadInfo, _, err := getThreadsWithBt(ssn)
			if err != nil {
				ssn.cmdMsgBody.sendErr(ssn.ws, err.Error(), "threadInfo", threadInfo)
				return nil
			}
			ssn.cmdMsgBody.send(ssn.ws, "-gdb-get-threads-frames", threadInfo)

		case "-see-files":

			names, ok := msg.data.Data["args"]
			if !ok {
				ssn.cmdMsgBody.sendErr(ssn.ws, "-see-files", "argument error: 'args' not found")
				return nil
			}
			files := make(map[string]string)

			if nmList, ok := names.([]interface{}); ok {

				for _, v := range nmList {
					var nm string
					if nm, ok = v.(string); !ok {
						nm = fmt.Sprintf("%v", v)
					}

					if tx, err := readFile(nm); err != nil {
						files[nm] = fmt.Sprintf("Error reading %s: %v", nm, err)
					} else {
						files[nm] = tx
					}
				}
				ssn.cmdMsgBody.send(ssn.ws, "-see-files", files)
				return nil
			}

			nm, ok := names.(string)
			if !ok {
				nm = fmt.Sprintf("%v", names)
			}
			if tx, err := readFile(nm); err != nil {
				files[nm] = fmt.Sprintf("Error reading %s: %v", nm, err)
			} else {
				files[nm] = tx
			}
			ssn.cmdMsgBody.send(ssn.ws, "-see-files", files)
			return nil
		}
	}
	return nil
}

func readFile(name string) (contents string, err error) {

	f, err := os.Open(name)
	if err != nil {
		return
	}
	if bts, err := ioutil.ReadAll(f); err == nil {
		contents = string(bts)
	}
	return
}

func getThreadsWithBt(ssn *nvlvSsn) (threadsBt []map[string]interface{}, skipped [][]*gdb.Record, err error) {

	gdbSsn := ssn.gdbSsn
	timeout := 15 * time.Second
	i, resp, rSkip, err := gdbSsn.GetResponse("-thread-info", timeout)
	if err != nil {
		return
	}
	if len(rSkip) > 0 {
		skipped = append(skipped, rSkip...)
	}

	// now that have the list of threads collect stack info for each
	threads, ok := resp.Records[i].Data["threads"].([]interface{})
	if !ok {
		err = fmt.Errorf("unknown 'threads' type: %t", resp.Records[i].Data["threads"])
		return
	}
	threadsBt = make([]map[string]interface{}, 0, len(threads))

	// get a backtrace for each thread
	for _, th := range threads {

		thread, ok := th.(map[string]interface{})
		if !ok {
			err = fmt.Errorf("unknown thread type: %t", th)
			return
		}
		threadsBt = append(threadsBt, thread)
		id, ok := thread["id"].(string)
		if !ok {
			err = fmt.Errorf("unknown id type: %t", thread["id"])
			return
		}
		i, resp, rSkip, err = gdbSsn.GetResponse("-stack-list-frames --thread "+id, timeout)
		if len(rSkip) > 0 {
			skipped = append(skipped, rSkip...)
		}
		if err != nil {
			return
		}
		v := resp.Records[i].Data["stack"]
		thread["stack"] = v

		// get the args for each call in the stack...
		// stack info from backtrace
		stackData, ok := v.([]interface{})
		if !ok {
			err = fmt.Errorf("unknown stack type: %t", v)
			return
		}

		for _, frm := range stackData {

			// drill down to frame
			frmNV, ok := frm.(gdb.NamedValue)
			if !ok {
				err = fmt.Errorf("unknown frame type: %t", frm)
				return
			}
			frame, ok := frmNV.Data.(map[string]interface{})
			if !ok {
				err = fmt.Errorf("unknown frame value type: %t", frmNV.Data)
				return
			}

			lvl, ok := frame["level"].(string)
			if !ok {
				err = fmt.Errorf("unable to find or cast frame level: %v", frame)
				continue
			}

			// get the variables for this frame
			fVars, rSkip, vErr := gdbSsn.GetFrameVars(id, lvl, timeout)
			if len(rSkip) > 0 {
				skipped = append(skipped, rSkip...)
			}
			if vErr != nil {
				err = vErr
				return
			}
			frame["variables"] = fVars
		}
	}
	return threadsBt, skipped, err
}

func isReadErr(err error, src string) bool {
	if err == nil {
		return false
	}
	if err != io.EOF {
		log.Println(err)
	}
	log.Println("close source", src)
	return true
}
