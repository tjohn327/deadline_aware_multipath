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

type ScionSender struct {
	sendConn           pan.Conn
	sendSelector       pan.Selector
	retransmitConn     pan.Conn
	retransmitSelector pan.Selector
	ackListenConn      pan.ListenConn
	ingressChan        chan []byte
	ackChan            chan []byte
	retransmitChan     chan []byte
}

type ScionReceiver struct {
	listenConn  pan.ListenConn
	ackSendConn pan.Conn
	ackSelector pan.Selector
	egressChan  chan []byte
	ackChan     chan []byte
}

func NewScionSender(ctx context.Context, sendAddress *string, port *uint,
	sendSelector pan.Selector, retransmitSelector pan.Selector, ingressChan chan []byte,
	ackChan chan []byte, retransmitChan chan []byte) (*ScionSender, error) {
	sendAddr, err := pan.ResolveUDPAddr(*sendAddress)
	if err != nil {
		return nil, err
	}
	sendConn, err := pan.DialUDP(ctx, nil, sendAddr, nil, sendSelector)
	if err != nil {
		return nil, err
	}
	var retransmitConn pan.Conn
	if retransmitChan != nil {
		retransmitConn, err = pan.DialUDP(ctx, nil, sendAddr, nil, retransmitSelector)
		if err != nil {
			return nil, err
		}
	}
	var ackListenConn pan.ListenConn
	if ackChan != nil {
		ackListenConn, err = pan.ListenUDP(ctx, &net.UDPAddr{Port: int(*port) + 100}, nil)
		if err != nil {
			return nil, err
		}
	}
	sender := &ScionSender{
		sendConn:           sendConn,
		sendSelector:       sendSelector,
		retransmitConn:     retransmitConn,
		retransmitSelector: retransmitSelector,
		ackListenConn:      ackListenConn,
		ingressChan:        ingressChan,
		ackChan:            ackChan,
		retransmitChan:     retransmitChan,
	}
	go func() {
		defer sender.sendConn.Close()
		for {
			buf := <-sender.ingressChan
			// selector.SetPath(frag.path)
			_, err := sender.sendConn.Write(buf)
			if err != nil {
				ctx.Done()
				return
			}
		}
	}()

	if ackChan != nil {
		go func() {
			defer sender.ackListenConn.Close()
			buffer := make([]byte, MAX_UDP_BUFFER_SIZE)
			for {
				n, _, err := sender.ackListenConn.ReadFrom(buffer)
				if err != nil {
					ctx.Done()
					return
				}
				buf := make([]byte, n)
				copy(buf, buffer[:n])
				sender.ackChan <- buf
			}
		}()
	}
	if retransmitChan != nil {
		go func() {
			defer sender.retransmitConn.Close()
			for {
				buf := <-sender.retransmitChan
				// selector.SetPath(frag.path)
				_, err := sender.retransmitConn.Write(buf)
				if err != nil {
					ctx.Done()
					return
				}
			}
		}()
	}

	return sender, nil
}

func NewScionReceiver(ctx context.Context, sendAddress *string, port *uint,
	sendSelector pan.Selector, egressChan chan []byte,
	ackChan chan []byte) (*ScionReceiver, error) {
	sendAddr, err := pan.ResolveUDPAddr(*sendAddress)
	if err != nil {
		return nil, err
	}
	listenConn, err := pan.ListenUDP(ctx, &net.UDPAddr{Port: int(*port)}, nil)
	if err != nil {
		return nil, err
	}
	ackAddr := pan.UDPAddr{
		IA:   sendAddr.IA,
		IP:   sendAddr.IP,
		Port: sendAddr.Port + 100,
	}
	// ackSelector := &pan.DefaultSelector{}
	var ackSendConn pan.Conn
	if ackChan != nil {
		ackSendConn, err = pan.DialUDP(ctx, nil, ackAddr, nil, sendSelector)
		if err != nil {
			return nil, err
		}
	}
	receiver := &ScionReceiver{
		listenConn:  listenConn,
		ackSelector: sendSelector,
		ackSendConn: ackSendConn,
		egressChan:  egressChan,
		ackChan:     ackChan,
	}

	go func() {
		defer receiver.listenConn.Close()
		buffer := make([]byte, MAX_UDP_BUFFER_SIZE)
		for {
			n, _, err := receiver.listenConn.ReadFrom(buffer)
			if err != nil {
				ctx.Done()
				return
			}
			buf := make([]byte, n)
			copy(buf, buffer[:n])
			receiver.egressChan <- buf
		}
	}()

	if ackChan != nil {
		go func() {
			defer receiver.ackSendConn.Close()
			for {
				buf := <-receiver.ackChan
				_, err := receiver.ackSendConn.Write(buf)
				if err != nil {
					doneChan <- err
					return
				}
			}
		}()
	}

	return receiver, nil
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
