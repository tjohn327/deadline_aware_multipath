// // songgao/water water based ip gateway
package main

// import (
// 	"flag"
// 	"fmt"
// 	"log"
// 	"net"
// 	"os"
// 	"os/signal"
// 	"runtime"
// 	"syscall"
// 	"time"

// 	"github.com/songgao/water"
// )

// var (
// 	ifaceName string
// 	iface     *water.Interface
// )

// func init() {
// 	flag.StringVar(&ifaceName, "iface", "tun0", "name of the interface")
// }

// func main() {
// 	flag.Parse()
// 	iface, err := water.New(water.Config{
// 		DeviceType: water.TUN,
// 	})
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	fmt.Printf("Interface created: %s\n", iface.Name())

// 	iface.Name = ifaceName
// 	if err := iface.Configure(water.Config{
// 		DeviceType: water.TUN,
// 	}); err != nil {
// 		log.Fatal(err)
// 	}
// 	fmt.Printf("Interface configured: %s\n", iface.Name())

// 	if err := iface.Up(); err != nil {
// 		log.Fatal(err)
// 	}
// 	fmt.Printf("Interface up: %s\n", iface.Name())

// 	// Set up a goroutine to read from the interface.
// 	go func() {
// 		buf := make([]byte, 1500)
// 		for {
// 			n, err := iface.Read(buf)
// 			if err != nil {
// 				log.Fatal(err)
// 			}
// 			fmt.Printf("Read %d bytes from interface %s\n", n, iface.Name())
// 		}
// 	}()

// 	// Set up a goroutine to write to the interface.
// 	go func() {
// 		for {
// 			_, err := iface.Write([]byte("Hello from TUN!"))
// 			if err != nil {
// 				log.Fatal(err)
// 			}
// 			time.Sleep(1 * time.Second)
// 		}
// 	}

// 	// Set up a goroutine to handle signals.
// 	go func() {
// 		c := make(chan os.Signal, 1)
// 		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
// 		<-c
// 		iface.Close()
// 		os.Exit(0)
// 	}

// 	// Block forever.
// 	select {}

// }
