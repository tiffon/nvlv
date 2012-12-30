package svr

import (
	"bufio"
	"code.google.com/p/go.net/websocket"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

var listener net.Listener
var ssnBaseDir string
var HandlerPath string = "/nvlv"

func Start(ssnDir, port string) {

	ssnBaseDir = ssnDir

	fmt.Println("Handler path: ", HandlerPath)
	fmt.Println("Handler port: ", port)
	fmt.Println("Session dir:  ", ssnDir)

	http.Handle(HandlerPath, websocket.Handler(connHandler))

	// need this or will shadow listener
	var err error

	listener, err = net.Listen("tcp", port)
	if err != nil {
		log.Fatal("net.Listen err: ", err)
	}

	go svrInputLoop()
	go svrSignalTrap()

	err = http.Serve(listener, nil)

	if err != nil {
		log.Fatal("http.Serve err: ", err)
	}
}

func svrInputLoop() {

	fmt.Println("Enter 'exit' to end... ")

	input := bufio.NewReader(os.Stdin)

	for {
		line, err := input.ReadString('\n')

		if err != nil {
			log.Fatal("svrInputLoop err: ", err)
		}

		if line == "exit\n" || line == "quit\n" {
			fmt.Print("\n\033[33mExiting...\033[0m\n")
			listener.Close()
			break
		}
	}
}

func svrSignalTrap() {
	var interrupted = make(chan os.Signal)
	signal.Notify(interrupted, syscall.SIGQUIT, syscall.SIGINT)
	<-interrupted
	fmt.Printf("\n\033[33mExiting...\033[0m\n")
	listener.Close()
}
