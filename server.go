// CoAP Client and Server in Go
package coap

import (
	"log"
	"net"
	"time"
)

const maxPktLen = 1500

// Handle CoAP messages.
type RequestHandler interface {
	// Handle the message and optionally return a response message.
	Handle(l *net.UDPConn, a *net.UDPAddr, m Message) *Message
}

type funcHandler func(l *net.UDPConn, a *net.UDPAddr, m Message) *Message

func (f funcHandler) Handle(l *net.UDPConn, a *net.UDPAddr, m Message) *Message {
	return f(l, a, m)
}

// Build a handler from a function.
func FuncHandler(f func(l *net.UDPConn, a *net.UDPAddr, m Message) *Message) RequestHandler {
	return funcHandler(f)
}

func handlePacket(l *net.UDPConn, data []byte, u *net.UDPAddr,
	rh RequestHandler) {

	msg, err := parseMessage(data)
	if err != nil {
		log.Printf("Error parsing %v", err)
		return
	}

	rv := rh.Handle(l, u, msg)
	if rv != nil {
		log.Printf("Transmitting %#v", rv)
		Transmit(l, u, *rv)
	}
}

// Transmit a message.
func Transmit(l *net.UDPConn, a *net.UDPAddr, m Message) error {
	d, err := encodeMessage(m)
	if err != nil {
		return err
	}

	if a == nil {
		_, err = l.Write(d)
	} else {
		_, err = l.WriteTo(d, a)
	}
	return err
}

// Receive a message.
func Receive(l *net.UDPConn) (Message, error) {
	l.SetReadDeadline(time.Now().Add(RESPONSE_TIMEOUT))

	data := make([]byte, maxPktLen)
	nr, _, err := l.ReadFromUDP(data)
	if err != nil {
		return Message{}, err
	}
	return parseMessage(data[:nr])
}

// Bind to the given address and serve requests forever.
func ListenAndServe(n, addr string, rh RequestHandler) error {
	uaddr, err := net.ResolveUDPAddr(n, addr)
	if err != nil {
		return err
	}

	l, err := net.ListenUDP(n, uaddr)
	if err != nil {
		return err
	}

	for {
		buf := make([]byte, maxPktLen)

		nr, addr, err := l.ReadFromUDP(buf)
		if err == nil {
			go handlePacket(l, buf[:nr], addr, rh)
		}
	}
}
