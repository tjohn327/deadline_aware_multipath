package main

import (
	"net"
)

const (
	MAX_BUFFER_SIZE = 1200
)

type UDPProxy struct {
	conn        *net.UDPConn
	sendAddr    *net.UDPAddr
	EgressChan  chan []byte
	IngressChan chan []byte
}

func NewUDPProxy(listenAddress *string, sendAddress *string) (*UDPProxy, error) {
	raddr, err := net.ResolveUDPAddr("udp", *sendAddress)
	if err != nil {
		return nil, err
	}
	laddr, err := net.ResolveUDPAddr("udp", *listenAddress)
	if err != nil {
		return nil, err
	}
	conn, err := net.ListenUDP("udp", laddr)
	if err != nil {
		return nil, err
	}
	udpProxy := &UDPProxy{
		conn:        conn,
		sendAddr:    raddr,
		EgressChan:  make(chan []byte, 10),
		IngressChan: make(chan []byte, 10),
	}
	return udpProxy, nil
}

func (u *UDPProxy) Run() {
	buffer := make([]byte, MAX_BUFFER_SIZE)
	go func() {
		defer u.conn.Close()
		for {
			n, _, err := u.conn.ReadFromUDP(buffer)
			if err != nil {
				// doneChan <- err
				return
			}
			buf := make([]byte, n)
			copy(buf, buffer[:n])
			u.EgressChan <- buf
		}
	}()

	go func() {
		defer u.conn.Close()
		for {
			buf := <-u.IngressChan
			_, err := u.conn.WriteToUDP(buf, u.sendAddr)
			if err != nil {
				// doneChan <- err
				return
			}
			// if n != len(buf) {
			// 	doneChan <- err
			// 	return
			// }
		}
	}()
}
