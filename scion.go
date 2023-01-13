package main

import (
	"context"
	"log"
	"net"
	"runtime"
	"time"

	"github.com/netsec-ethz/scion-apps/pkg/pan"
	"github.com/vishvananda/netns"
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
	sendConn           net.Conn
	sendSelector       pan.Selector
	retransmitConn     net.Conn
	retransmitSelector pan.Selector
	ackListenConn      net.Conn
	ingressChan        chan []byte
	ackChan            chan []byte
	retransmitChan     chan []byte
	paritySendConn     net.Conn
	paritySelector     pan.Selector
	ingressChanParity  chan []byte
}

type ScionReceiver struct {
	listenConn       net.Conn
	listenConnRe     net.Conn
	listenConnParity net.Conn
	ackSendConn      net.Conn
	ackSelector      pan.Selector
	egressChan       chan []byte
	reInChan         chan []byte
	ackChan          chan []byte
	parityChan       chan []byte
}

func NewScionSender(ctx context.Context, sendAddress *string, port *uint,
	sendSelector pan.Selector, retransmitSelector pan.Selector,
	ingressChan chan []byte, ackChan chan []byte,
	retransmitChan chan []byte, paritySelector pan.Selector,
	ingressChanParity chan []byte) (*ScionSender, error) {

	if ingressChan == nil {
		ingressChan = make(chan []byte, 500)
	}

	var retransmitConn net.Conn
	var ackListenConn net.Conn
	var sendConn net.Conn

	sendAddr, err := net.ResolveUDPAddr("udp", "10.3.0.2:30001")
	if err != nil {
		return nil, err
	}
	sendAddrRe, err := net.ResolveUDPAddr("udp", "10.1.0.2:30002")
	if err != nil {
		return nil, err
	}
	// sendAddrACK, err := net.ResolveUDPAddr("udp", "10.1.0.2:30003")
	// if err != nil {
	// 	return nil, err
	// }
	sendAddrParity, err := net.ResolveUDPAddr("udp", "10.1.0.2:30004")
	if err != nil {
		return nil, err
	}
	laddr, err := net.ResolveUDPAddr("udp", "10.3.0.1:30001")
	if err != nil {
		return nil, err
	}
	laddrRe, err := net.ResolveUDPAddr("udp", "10.1.0.1:30002")
	if err != nil {
		return nil, err
	}
	laddrACK, err := net.ResolveUDPAddr("udp", "10.1.0.1:30003")
	if err != nil {
		return nil, err
	}
	lAddrParity, err := net.ResolveUDPAddr("udp", "10.1.0.1:30004")
	if err != nil {
		return nil, err
	}
	sendConn, err = net.DialUDP("udp", laddr, sendAddr)
	if err != nil {
		return nil, err
	}

	if retransmitChan != nil {
		retransmitConn, err = net.DialUDP("udp", laddrRe, sendAddrRe)
		if err != nil {
			return nil, err
		}
	}

	if ackChan != nil {
		ackListenConn, err = net.ListenUDP("udp", laddrACK)
		if err != nil {
			return nil, err
		}
	}

	paritySendConn, err := net.DialUDP("udp", lAddrParity, sendAddrParity)
	if err != nil {
		return nil, err
	}
	// if !TEST_UDP {
	// 	sendAddr, err := pan.ResolveUDPAddr(*sendAddress)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// 	sendConn, err = pan.DialUDP(ctx, nil, sendAddr, nil, sendSelector)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// 	if retransmitChan != nil {
	// 		retransmitConn, err = pan.DialUDP(ctx, nil, sendAddr, nil, retransmitSelector)
	// 		if err != nil {
	// 			return nil, err
	// 		}
	// 	}
	// 	if ackChan != nil {
	// 		ackListenConn, err = pan.ListenUDP(ctx, &net.UDPAddr{Port: int(*port) + 100}, nil)
	// 		if err != nil {
	// 			return nil, err
	// 		}
	// 	}

	// 	paritySendConn, err = pan.DialUDP(ctx, nil, sendAddr, nil, paritySelector)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// }

	sender := &ScionSender{
		sendConn:           sendConn,
		sendSelector:       sendSelector,
		retransmitConn:     retransmitConn,
		retransmitSelector: retransmitSelector,
		ackListenConn:      ackListenConn,
		ingressChan:        ingressChan,
		ackChan:            ackChan,
		retransmitChan:     retransmitChan,
		paritySendConn:     paritySendConn,
		ingressChanParity:  ingressChanParity,
		paritySelector:     paritySelector,
	}
	go func() {
		defer sender.sendConn.Close()
		for {
			select {

			case buf := <-sender.ingressChan:
				// selector.SetPath(frag.path)
				// if sender.sendSelector.Path() == nil {
				// 	log.Println("no path available\n")
				// 	continue
				// }
				_, err := sender.sendConn.Write(buf)
				// time.Sleep(time.Microsecond * 1)
				if err != nil {
					// errChan <- err
					log.Println("scion send:", err)
					continue
				}
			case <-time.After(runDuration):
				return
			}

			// log.Println("scion send:", n, len(buf))
		}
	}()

	go func() {
		defer sender.paritySendConn.Close()
		for {
			select {
			case buf := <-sender.ingressChanParity:
				// selector.SetPath(frag.path)
				// if sender.sendSelector.Path() == nil {
				// 	log.Println("no path available\n")
				// 	continue
				// }
				_, err := sender.paritySendConn.Write(buf)
				// time.Sleep(time.Microsecond * 100)
				if err != nil {
					// errChan <- err
					log.Println("scion send:", err)
					continue
				}
			case <-time.After(runDuration):
				return
			}

			// log.Println("scion send:", n, len(buf))
		}
	}()

	if ackChan != nil {
		go func() {
			defer sender.ackListenConn.Close()
			buffer := make([]byte, MAX_BUFFER_SIZE)
			for {
				n, err := sender.ackListenConn.Read(buffer)
				if err != nil {
					// errChan <- err
					log.Println("ack listen:", err)
					continue
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
				// time.Sleep(time.Microsecond * 1)
				if err != nil {
					// errChan <- err
					log.Println("retransmit:", err)
					continue
				}
			}
		}()
	}

	return sender, nil
}

func NewScionReceiver(ctx context.Context, sendAddress *string, port *uint,
	sendSelector pan.Selector, egressChan chan []byte, reInChan chan []byte,
	ackChan chan []byte, parityChan chan []byte) (*ScionReceiver, error) {

	// reInChan := make(chan []byte, 500)
	ns, err := netns.GetFromName("ns2")
	if err != nil {
		checkNonFatal(err)
	}
	defer ns.Close()
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if err := netns.Set(ns); err != nil {
		checkNonFatal(err)
	}

	var listenConnRe net.Conn
	var listenConn net.Conn
	var listenConnParity net.Conn
	var ackSendConn net.Conn

	lAddr, err := net.ResolveUDPAddr("udp", "10.3.0.2:30001")
	if err != nil {
		return nil, err
	}
	lAddrRe, err := net.ResolveUDPAddr("udp", "10.1.0.2:30002")
	if err != nil {
		return nil, err
	}
	lAddrACK, err := net.ResolveUDPAddr("udp", "10.1.0.2:30003")
	if err != nil {
		return nil, err
	}
	sendAddrACK, err := net.ResolveUDPAddr("udp", "10.1.0.1:30003")
	if err != nil {
		return nil, err
	}
	lAddrParity, err := net.ResolveUDPAddr("udp", "10.1.0.2:30004")
	if err != nil {
		return nil, err
	}

	listenConnRe, err = net.ListenUDP("udp", lAddrRe)
	if err != nil {
		return nil, err
	}
	listenConn, err = net.ListenUDP("udp", lAddr)
	if err != nil {
		return nil, err
	}

	listenConnParity, err = net.ListenUDP("udp", lAddrParity)
	if err != nil {
		return nil, err
	}
	ackSendConn, err = net.DialUDP("udp", lAddrACK, sendAddrACK)
	if err != nil {
		return nil, err
	}

	// if !TEST_UDP {

	// sendAddr, err := pan.ResolveUDPAddr(*sendAddress)
	// if err != nil {
	// 	return nil, err
	// }
	// listenConn, err := pan.ListenUDP(ctx, &net.UDPAddr{Port: int(*port)}, nil)
	// if err != nil {
	// 	return nil, err
	// }
	// ackAddr := pan.UDPAddr{
	// 	IA:   sendAddr.IA,
	// 	IP:   sendAddr.IP,
	// 	Port: sendAddr.Port + 100,
	// }
	// // ackSelector := &pan.DefaultSelector{}
	// var ackSendConn pan.Conn
	// if ackChan != nil {
	// 	ackSendConn, err = pan.DialUDP(ctx, nil, ackAddr, nil, sendSelector)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// }
	//}

	if egressChan == nil {
		egressChan = make(chan []byte, 500)
	}
	receiver := &ScionReceiver{
		listenConn:       listenConn,
		listenConnRe:     listenConnRe,
		listenConnParity: listenConnParity,
		ackSelector:      sendSelector,
		ackSendConn:      ackSendConn,
		egressChan:       egressChan,
		ackChan:          ackChan,
		reInChan:         reInChan,
		parityChan:       parityChan,
	}

	// ns.Close()
	// runtime.UnlockOSThread()

	go func() {
		defer receiver.listenConn.Close()
		buffer := make([]byte, MAX_BUFFER_SIZE)
		for {
			n, err := receiver.listenConn.Read(buffer)
			if err != nil {
				log.Println("scion listen:", err)
				continue
			}
			buf := make([]byte, n)
			copy(buf, buffer[:n])
			receiver.egressChan <- buf
		}
	}()

	go func() {
		defer receiver.listenConnParity.Close()
		buffer := make([]byte, MAX_BUFFER_SIZE)
		for {
			n, err := receiver.listenConnParity.Read(buffer)
			if err != nil {
				log.Println("scion listen:", err)
				continue
			}
			buf := make([]byte, n)
			copy(buf, buffer[:n])
			receiver.parityChan <- buf
		}
	}()

	go func() {
		defer receiver.listenConnRe.Close()
		buffer := make([]byte, MAX_BUFFER_SIZE)
		for {
			n, err := receiver.listenConnRe.Read(buffer)
			if err != nil {
				log.Println("scion listen:", err)
				continue
			}
			buf := make([]byte, n)
			copy(buf, buffer[:n])
			receiver.reInChan <- buf
		}
	}()

	if ackChan != nil {
		go func() {
			defer receiver.ackSendConn.Close()
			for {
				buf := <-receiver.ackChan
				_, err := receiver.ackSendConn.Write(buf)
				if err != nil {
					log.Println("ack send", err)
					continue
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
		ackChan = make(chan []byte, 500)
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
		buffer := make([]byte, MAX_BUFFER_SIZE)
		for {
			n, _, err := g.listenConn.ReadFrom(buffer)
			if err != nil {
				// doneChan <- err
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
			// time.Sleep(time.Microsecond)

			if err != nil {
				log.Println("send", err)
				continue
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
				log.Println("ack send", err)
				continue
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
		buffer := make([]byte, MAX_BUFFER_SIZE)
		for {
			n, _, err := g.ackListenConn.ReadFrom(buffer)
			if err != nil {

				return
			}
			buf := make([]byte, n)
			copy(buf, buffer[:n])
			g.ACKChan <- buf
		}
	}()
}
