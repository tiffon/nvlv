package main

import (
	"flag"
	"github.com/tiffon/nvlv/svr"
)

// Port to listen on
var WebSockPort string

// Local dir for session related data
var SessionStorageDir string

func main() {
	flag.StringVar(&WebSockPort, "websocket-port", ":12345", "nvlv server websocket port number")
	flag.StringVar(&SessionStorageDir, "session-dir", "/data/nvlv", "local dir for session related data")
	flag.Parse()
	svr.Start(SessionStorageDir, WebSockPort)
}
