package svr

import (
	"code.google.com/p/go.net/websocket"
	"github.com/tiffon/nvlv/svr/cmn"
	"log"
)

func connHandler(ws *websocket.Conn) {

	ssn, err := newNvlvSsn(ws)
	if err != nil {
		s := "error creating nvlv session: " + err.Error()
		log.Println(s)
		websocket.Message.Send(ws, s)
		return
	}

	err = ssn.Run()
	if err != nil {
		s := "error starting nvlv session: " + err.Error()
		log.Println(s)
		websocket.Message.Send(ws, s)
	}
}

func recvJsonLoop(ws *websocket.Conn, resutlChan chan *clientMsg) {
	for {
		msg := new(clientMsg)
		msg.err = websocket.JSON.Receive(ws, &msg.data)
		resutlChan <- msg
	}
}

type clientMsg struct {
	data *clientBody
	err  error
}

type clientBody struct {
	Ctx  string
	Data map[string]interface{}
}

func (c *clientBody) sendErr(ws *websocket.Conn, msg interface{}, keyValPairs ...interface{}) {

	sendOnWS(c, ws, "err", msg, keyValPairs...)
}

func (c *clientBody) sendMsg(ws *websocket.Conn, msg interface{}, keyValPairs ...interface{}) {

	sendOnWS(c, ws, "msg", msg, keyValPairs...)
}

func (c *clientBody) sendData(ws *websocket.Conn, data interface{}, keyValPairs ...interface{}) {

	sendOnWS(c, ws, "data", data, keyValPairs...)
}

func (c *clientBody) send(ws *websocket.Conn, keyvals ...interface{}) error {

	keys, err := cmn.AppendKVPs(c.Data, keyvals)
	if err != nil {
		log.Println("Error appending extra key and values in send Err: ", err)
		return err
	}
	websocket.JSON.Send(ws, c)

	if keys != nil {
		for _, key := range keys {
			delete(c.Data, key)
		}
	}
	return nil
}

func sendOnWS(c *clientBody, ws *websocket.Conn, key string, data interface{}, keyValPairs ...interface{}) {

	xtraKeys, err := cmn.AppendKVPs(c.Data, keyValPairs)
	if err != nil {
		log.Println("Error appending extra key and values in send Err: ", err)
	}
	c.Data[key] = data
	websocket.JSON.Send(ws, c)

	delete(c.Data, key)
	if xtraKeys != nil {
		for _, key := range xtraKeys {
			delete(c.Data, key)
		}
	}
}
