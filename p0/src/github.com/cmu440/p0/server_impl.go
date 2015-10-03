// Implementation of a MultiEchoServer. Students should write their code in this file.

package p0

import (
	"bufio"
	"fmt"
	"net"
	"strconv"
)

type Client struct {
	clientConn net.Conn
	clientChan chan string
}

type multiEchoServer struct {
	msgchan  chan string
	addchan  chan Client
	rmchan   chan net.Conn
	quitchan chan bool
	countReq chan bool
	countRcv chan int
	ln       net.Listener
	clients  map[net.Conn](chan string)
}

// Creates and returns (but does not start) a new MultiEchoServer.
func New() MultiEchoServer {

	server := multiEchoServer{
		msgchan:  make(chan string),
		addchan:  make(chan Client),
		rmchan:   make(chan net.Conn),
		quitchan: make(chan bool),
		countReq: make(chan bool),
		countRcv: make(chan int),
		clients:  make(map[net.Conn](chan string))}
	return &server
}

// Start net listener and accept clients
func (mes *multiEchoServer) Start(port int) error {

	ln, err := net.Listen("tcp", ":"+strconv.Itoa(port))

	if err != nil {
		fmt.Println("Couldn't listen: ", err)
	}
	mes.ln = ln

	// handle message on server
	go handleMessage(mes)

	// accept clients in a goroutine to avoid blocking
	go acceptClients(mes)
	return nil
}

func (mes *multiEchoServer) Close() {

	close(mes.quitchan)
	mes.ln.Close()

	return
}

func (mes *multiEchoServer) Count() int {

	go func() { mes.countReq <- true }()
	for {
		select {
		case count := <-mes.countRcv:
			return count
		}
	}
	return -1
}

func handleMessage(mes *multiEchoServer) {

	for {
		select {

		case <-mes.quitchan: // close all the clients and close goroutine
			for conn, _ := range mes.clients {
				err := conn.Close()
				if err != nil {
					fmt.Println("Couldn't close conn: ", err)
					continue
				}
			}
			return

		case <-mes.countReq: // count length of map
			mes.countRcv <- len(mes.clients)

		case msg := <-mes.msgchan: // double echo msg
			s := msg
			if last := len(s) - 1; last >= 0 && s[last] == '\n' {
				s = s[:last]
			}

			processedMsg := s + s + "\n"

			for _, clientChan := range mes.clients {
				select {
				case clientChan <- processedMsg:
				default:
					continue
				}
			}

		case client := <-mes.addchan: // add clients
			mes.clients[client.clientConn] = client.clientChan

		case conn := <-mes.rmchan: // remove clients
			delete(mes.clients, conn)
		}
	}
}

// accept connections
func acceptClients(mes *multiEchoServer) {
	for {
		select {
		case <-mes.quitchan:
			return
		default:
			conn, err := mes.ln.Accept()
			if err != nil {
				fmt.Println("Couldn't accept: ", err)
				continue
			}
			go handleConnection(mes, conn)
		}
	}
}

func handleConnection(mes *multiEchoServer, conn net.Conn) {
	// Register user
	clientCh := make(chan string, 75)
	mes.addchan <- Client{conn, clientCh}

	go readLinesInto(mes, conn)
	writeLinesFrom(clientCh, conn)
}

// read msg from conn and add to msg channel of server
func readLinesInto(mes *multiEchoServer, conn net.Conn) {
	b := bufio.NewReader(conn)
	for {
		line, err := b.ReadString('\n')
		if err != nil {
			mes.rmchan <- conn
		}
		mes.msgchan <- line
	}
}

// write the msg in clients' channel into conne
func writeLinesFrom(clientChan <-chan string, conn net.Conn) {
	for msg := range clientChan {
		conn.Write([]byte(msg))
	}
}
