package main

import (
	"context"
	"net"

	"github.com/netsec-ethz/scion-apps/pkg/pan"
)

type SCIONGateway struct {
	listenConn    pan.ListenConn
	sendConn      pan.Conn
	sendSelector  pan.Selector
	ackListenConn pan.ListenConn
	ackSendConn   pan.Conn
	ackSelector   pan.Selector
	EgressChan    chan []byte
	IngressChan   chan []byte
	ACKChan       chan []byte
}

func NewSCIONGateway(ctx context.Context, sendAddress *string, port *uint,
	egressChan chan []byte, ingressChan chan []byte,
	ackChan chan []byte) (*SCIONGateway, error) {

	sendAddr, err := pan.ResolveUDPAddr(*sendAddress)
	if err != nil {
		return nil, err
	}
	listenConn, err := pan.ListenUDP(ctx, &net.UDPAddr{Port: int(*port)}, nil)
	if err != nil {
		return nil, err
	}

	sendSelector := &pan.DefaultSelector{}
	sendConn, err := pan.DialUDP(ctx, nil, sendAddr, nil, sendSelector)
	if err != nil {
		return nil, err
	}

	if ackChan == nil {
		ackChan = make(chan []byte, 10)
	}

	ackAddr := pan.UDPAddr{
		IA:   sendAddr.IA,
		IP:   sendAddr.IP,
		Port: sendAddr.Port + 100,
	}
	ackSelector := &pan.DefaultSelector{}
	ackSendConn, err := pan.DialUDP(ctx, nil, ackAddr, nil, ackSelector)
	if err != nil {
		return nil, err
	}

	ackListenConn, err := pan.ListenUDP(ctx, &net.UDPAddr{Port: int(*port) + 100}, nil)
	if err != nil {
		return nil, err
	}

	sg := &SCIONGateway{
		listenConn:    listenConn,
		sendConn:      sendConn,
		sendSelector:  sendSelector,
		ackListenConn: ackListenConn,
		ackSelector:   ackSelector,
		ackSendConn:   ackSendConn,
		EgressChan:    egressChan,
		IngressChan:   ingressChan,
		ACKChan:       ackChan,
	}
	return sg, nil
}

func (g *SCIONGateway) Run() {
	go func() {
		defer g.listenConn.Close()
		buffer := make([]byte, MAX_UDP_BUFFER_SIZE)
		for {
			n, _, err := g.listenConn.ReadFrom(buffer)
			if err != nil {
				doneChan <- err
				return
			}
			buf := make([]byte, n)
			copy(buf, buffer[:n])
			g.EgressChan <- buf
		}
	}()

	go func() {
		defer g.sendConn.Close()
		for {
			buf := <-g.IngressChan
			// selector.SetPath(frag.path)
			_, err := g.sendConn.Write(buf)
			if err != nil {
				doneChan <- err
				return
			}
		}
	}()
}

func (g *SCIONGateway) RunACKSend() {
	if g.ACKChan == nil {
		g.ackSendConn.Close()
		return
	}
	go func() {
		defer g.ackSendConn.Close()
		for {
			buf := <-g.ACKChan
			_, err := g.ackSendConn.Write(buf)
			if err != nil {
				doneChan <- err
				return
			}
		}
	}()
}

func (g *SCIONGateway) RunACKReceive() {
	if g.ACKChan == nil {
		g.ackListenConn.Close()
		return
	}
	go func() {
		defer g.ackListenConn.Close()
		buffer := make([]byte, MAX_UDP_BUFFER_SIZE)
		for {
			n, _, err := g.ackListenConn.ReadFrom(buffer)
			if err != nil {
				doneChan <- err
				return
			}
			buf := make([]byte, n)
			copy(buf, buffer[:n])
			g.ACKChan <- buf
		}
	}()
}
