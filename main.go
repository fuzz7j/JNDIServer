package main

import (
	hex2 "encoding/hex"
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

var (
	res      []string
	port     int
	exitChan = make(chan int)
	wg       sync.WaitGroup
)

func main() {
	CommandParse()
	res = append(res, "running")
	go Listen()
	code := <-exitChan
	os.Exit(code)
}

func CommandParse() {
	flag.IntVar(&port, "p", 8000, "JNDI Server Listen Port")
	flag.Parse()
}

func ChooseMode(b byte, conn net.Conn) {
	switch b {
	case 0x47:
		HttpServer(conn)
	case 0x30:
		LDAPServer(conn)
	case 0x4a:
		RMIServer(conn)
	default:
		conn.Close()
		break
	}
}

func Listen() {
	var bytes []byte
	ln, err := net.Listen("tcp", fmt.Sprintf(":%v", port))
	fmt.Printf("Author: fuzz7j (https://github.com/fuzz7j/JNDIServer) \nListen on 0.0.0.0:%v\n", port)

	if err != nil {
		fmt.Println(err.Error())
		exitChan <- 1
	}
	defer ln.Close()

	for {
		conn, _ := ln.Accept()
		if err != nil {
			fmt.Println(err.Error())
			continue
		}
		fb, err := ReadTagAndLength(conn, &bytes)
		if err == nil {
			go ChooseMode(fb, conn)
		}
	}
}

func ReadTagAndLength(conn net.Conn, bytes *[]byte) (fb byte, err error) {
	var b byte
	b, err = ReadBytes(conn, bytes, 1)
	if err != nil {
		return
	}
	return b, err
}

func ReadBytes(conn net.Conn, bytes *[]byte, length int) (b byte, err error) {
	newbytes := make([]byte, length)
	n, err := conn.Read(newbytes)
	if n != length {
		fmt.Errorf("%d bytes read instead of %d", n, length)
	} else if err != nil {
		return
	}
	*bytes = append(*bytes, newbytes...)
	b = (*bytes)[len(*bytes)-1]
	return
}

func HttpServer(conn net.Conn) {
	rev := make([]byte, 1024)
	_, err := conn.Read(rev)
	defer conn.Close()
	if err == nil {
		session := strings.Replace(strings.Split(string(rev), " ")[1], "/session/", "", -1)
		IsSave := CheckSave(session)
		fmt.Printf("%v HTTP Query \"%v\" From %v\n", time.Now().Format("2006-01-02 15:04:05"), strings.TrimSpace(session), conn.RemoteAddr())
		if IsSave {
			conn.Write([]byte("True\n"))
		} else {
			conn.Write([]byte("False\n"))
		}
	}
}

func LDAPServer(conn net.Conn) {
	rev := make([]byte, 1024)
	data, _ := hex2.DecodeString("300c02010161070a010004000400\n")
	_, err := conn.Read(rev)
	defer conn.Close()
	if err == nil {
		conn.Write(data)
		_, err := conn.Read(rev)
		if err == nil {
			res = append(res, string(rev[8:30]))
			fmt.Printf("%v LDAP Query \"%v\" From %v\n",time.Now().Format("2006-01-02 15:04:05"), strings.TrimSpace(string(rev[8:30])), conn.RemoteAddr())
		}
	}
}

func RMIServer(conn net.Conn) {
	rev := make([]byte, 1024)
	data, _ := hex2.DecodeString("4e00096c6f63616c686f73740000ff47\n")
	_, err := conn.Read(rev)
	defer conn.Close()
	if err == nil {
		conn.Write(data)
		_, err := conn.Read(rev)
		if err == nil {
			conn.Read(rev)
			res = append(res, string(rev[43:67]))
			fmt.Printf("%v RMI Query \"%v\" From %v\n", time.Now().Format("2006-01-02 15:04:05"), strings.TrimSpace(string(rev[43:67])), conn.RemoteAddr())
		}
	}
}

func CheckSave(session string) (flag bool) {
	session = strings.TrimSpace(session)
	flag = false
	for _, i := range res {
		if strings.Contains(i, session) {
			flag = true
		}
	}
	return flag
}