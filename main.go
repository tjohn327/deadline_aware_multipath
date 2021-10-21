package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"
)

var (
	doneChan = make(chan error, 1)
)

func main() {
	remoteAddr := flag.String("remote", "", "[SIM] Remote control's SCION Address (e.g. 17-ffaa:1:1,[127.0.0.1]:12345)")
	simAddr := flag.String("sim", "", "[Remote Control] Simulator's SCION Address (e.g. 17-ffaa:1:1,[127.0.0.1]:12345)")
	port := flag.Uint("port", 11550, "local SCION port to listen on")
	listenAddr := flag.String("listen", "", "UDP address to listen on  (e.g. 127.0.0.1:12345)")
	sendAddr := flag.String("send", "", "UDP address to send on  (e.g. 127.0.0.1:12345)")
	flag.Parse()

	if (len(*remoteAddr) > 0) == (len(*simAddr) > 0) {
		check(fmt.Errorf("either specify -remote for remote side or -sim for simulator side"))
	}
	ctx := context.Background()
	if len(*remoteAddr) > 0 {
		RunSim(ctx, remoteAddr, listenAddr, sendAddr, port)
	} else {
		RunRemote(ctx, simAddr, listenAddr, sendAddr, port)
	}

	var err error
	select {
	case <-ctx.Done():
		fmt.Println("cancelled")
		err = ctx.Err()
	case err = <-doneChan:
	}
	check(err)
}

func RunRemote(ctx context.Context, simAdress *string, listenAddr *string, sendAddr *string, port *uint) {
	udp, err := NewUDPProxy(listenAddr, sendAddr)
	check(err)
	udp.Run()
	vidReceiveChan := make(chan []byte, 10)
	gw, err := NewSCIONGateway(ctx, simAdress, port, vidReceiveChan, udp.EgressChan, nil)
	check(err)
	gw.Run()
	gw.RunACKSend()

	vid := NewDataQueue(vidReceiveChan, nil, gw.ACKChan, true)
	vid.RunIngress()

	go func() {
		for {
			frame := <-vid.egressChan
			for _, v := range frame.fragments {
				udp.IngressChan <- v.data
			}
		}
	}()
}

func RunSim(ctx context.Context, remoteAddr *string, listenAddr *string, sendAddr *string, port *uint) {
	udp, err := NewUDPProxy(listenAddr, sendAddr)
	check(err)
	udp.Run()
	vidSendChan := make(chan []byte, 10)
	gw, err := NewSCIONGateway(ctx, remoteAddr, port, udp.IngressChan, vidSendChan, nil)
	check(err)
	gw.Run()
	gw.RunACKReceive()

	vid := NewDataQueue(udp.EgressChan, nil, nil, false)
	vid.RunIngress()
	// vid.RunTrim()
	unack_vid := NewDataQueue(nil, nil, gw.ACKChan, false)
	// unack_vid.RunTrim()
	unack_vid.RunACK()

	go func() {
		for {
			select {
			case frame := <-vid.egressChan:
				unack_vid.InsertFrame(frame)
				// fmt.Println("Frag OUT", frame.frameID)
				for _, v := range frame.fragments {
					vidSendChan <- v.data
					// fmt.Println("Frag OUT", v.frameID, v.fragmentID)
				}
			case <-time.After(20 * time.Millisecond):
				if unack_vid.len > 0 {
					frame := unack_vid.DiscardFrame()
					fmt.Println("Reinject", frame.blockID, frame.len, unack_vid.len)
					for _, v := range frame.fragments {
						vidSendChan <- v.data
						// fmt.Println("Frag OUT", v.frameID, v.fragmentID)
					}
				}
			}

		}
	}()
}

func check(e error) {
	if e != nil {
		fmt.Fprintln(os.Stderr, "Fatal error:", e)
		os.Exit(1)
	}
}
