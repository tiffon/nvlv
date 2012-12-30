// This is just an executable to run in the GDB, for testing purposes.

package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"time"
)

func checkErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func readLoop(rdr *bufio.Reader, cmd *exec.Cmd) {
	fmt.Println("read loop started")
	b, err := rdr.ReadByte()
	for err == nil {
		fmt.Printf("%c", b)
		b, err = rdr.ReadByte()
	}
	if err != io.EOF {
		log.Fatal(err)
	}

	fmt.Println("read loop ended")
}

func main() {

	fmt.Println("first thing in main")
	cmd := exec.Command("bash")
	fmt.Println("cmd created")

	inPipe, err := cmd.StdinPipe()
	if err != nil {
		log.Fatal(err)
	}

	outPipe, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}

	errPipe, err := cmd.StderrPipe()
	if err != nil {
		log.Fatal(err)
	}

	input := bufio.NewReader(os.Stdin)
	rdr := bufio.NewReader(outPipe)

	go io.Copy(os.Stderr, errPipe)

	fmt.Println("reading from the pipe")

	err = cmd.Start()
	defer cmd.Wait()

	// go io.Copy(os.Stderr, inPipe)

	// inPipe.Write([]byte("bash\n")): help

	// inPipe.Write([]byte("ls -l\n"))
	// inPipe.Write([]byte("pfffwd\n"))

	// go func() {
	// 	time.Sleep(6 * time.Second)
	// 	fmt.Println("exiting")
	// 	inPipe.Write([]byte("exit\n"))116
	// }()
	// fmt.Println("sleeper is going")

	// go func() {
	// 	// inPipe.Write([]byte("pwd\n"))
	// 	for {
	// 		readLoop(rdr)
	// 		fmt.Print(">> ")
	// 		line, err := input.ReadString('\n')
	// 		checkErr(err)
	// 		inPipe.Write([]byte(line + "\n"))
	// 	}
	// }()
	// fmt.Println("loop is going")

	go func() {
		for {
			//fmt.Print(">> ")
			// os.Stdout.WriteString(">> ")
			fmt.Print(">> ")
			line, err := input.ReadString('\n')
			checkErr(err)
			fmt.Println("line:" + line)
			inPipe.Write([]byte(line + "\n"))
		}
	}()

	go readLoop(rdr, cmd)

	// for testing debugger, create some go routines that just sleep and count
	for i := 0; i < 10; i++ {
		go func() {
			for count := 0; count < 1000; count++ {
				timer := time.NewTimer(1e9)
				select {
				case <-timer.C:
				}
				timer.Stop()
			}
		}()
	}
	outsideFunc("arg of awesomness")
	return
}

func outsideFunc(aweArg string) {
	localness := true
	fmt.Println("we are outside!!", aweArg, localness)
	onlyArg(true)
}

func onlyArg(arg bool) {
	fmt.Println("only arg:", arg)
	onlyLocal()
}

func onlyLocal() {
	local := true
	fmt.Println("local: ", local)
}
