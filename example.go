package main

// import (
// 	"bytes"
// 	"darm"
// 	"log"
// 	"os"
// 	"os/signal"
// )

// func main() {
// 	sigCh := make(chan os.Signal, 1)
// 	signal.Notify(sigCh, os.Interrupt)

// 	var (
// 		b0 bytes.Buffer
// 		b1 bytes.Buffer
// 	)
// 	p := make([]byte, 5000)

// 	gw := darm.NewGateway(15000)
// 	gw.Start()

// 	conn, err := gw.Dial("19-ffaa:1:eba,127.0.0.1:15000",
// 		"50ms", 2, 1200)
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	sendStreams, err := conn.GetSendStreams()
// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	receiveStreams, err := conn.GetReceiveStreams()
// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	go func() {
// 		for {
// 			b0.Read(p)
// 			sendStreams[0].Write(p)
// 		}
// 	}()

// 	go func() {
// 		for {
// 			p := make([]byte, 5000)
// 			b1.Read(p)
// 			sendStreams[1].Write(p)
// 		}
// 	}()

// 	go func() {
// 		for {
// 			p := make([]byte, 5000)
// 			receiveStreams[1].Read(p)
// 		}
// 	}()

// 	<-sigCh
// }
