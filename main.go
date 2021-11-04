package main

import (
	"context"
	"crypto/md5"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/signal"
	"time"

	"github.com/BurntSushi/toml"
)

var (
	errChan        = make(chan error, 1)
	send_telemetry = true
)

func main() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)

	configFile := flag.String("c", "", "location of the config file")
	flag.Parse()
	if *configFile == "" {
		check(fmt.Errorf("config file required"))
	}
	var cfg Config
	if _, err := toml.DecodeFile(*configFile, &cfg); err != nil {
		check(err)
	}
	switch cfg.GwType {
	case "sender":
		RunSender(&cfg)
	case "receiver":
		RunReceiver(&cfg)
	default:
		check(fmt.Errorf("invalid type"))
	}

	select {
	case <-sigCh:
		log.Println("\nterminating")
		return
	case err := <-errChan:
		check(err)
	}
}

func RunSender(cfg *Config) {
	log.Println("starting sender")
	ingressChan := make(chan []byte, 10)
	scheduler, err := NewScheduler(context.Background(), cfg)
	check(err)
	manager, err := NewManager(cfg, scheduler, int(cfg.FragSize))
	check(err)
	rd, err := NewReedSolomon(10, manager.parityCount)
	check(err)
	scheduler.Run()
	manager.Run()

	test_image, err := ioutil.ReadFile("testdata/test_image.jpg")
	check(err)

	var timestamp_conn net.Conn
	if send_telemetry {
		timestamp_conn, _ = net.Dial("udp", "192.168.1.151:19000")
	}

	go func() {
		blockID := 0
		for {
			data := <-ingressChan
			split, err := Split(data, manager.fragSize)
			checkNonFatal(err)
			encodedData, err := rd.Encode(split, manager.parityCount)
			checkNonFatal(err)
			db := NewDataBlockFromEncodedData(encodedData, blockID)
			if send_telemetry {
				blockid := fmt.Sprintf(`{"blockid":"%d"}`, db.blockID)
				timestamp_conn.Write([]byte(blockid))
			}
			scheduler.Send(db)
			if blockID > 65534 {
				blockID = 0
			} else {
				blockID++
			}
		}
	}()

	time.Sleep(5 * time.Second)
	go func() {
		delay := time.Duration(int64(float64(cfg.Deadline.Duration) * 1.2))
		for {
			buf := make([]byte, len(test_image))
			copy(buf, test_image)
			ingressChan <- buf
			time.Sleep(delay)
		}
	}()

}

func RunReceiver(cfg *Config) {
	log.Println("starting receiver")

	test_image_hash := [16]byte{105, 162, 92, 131, 191, 110, 214, 187, 153, 225, 26, 200, 95, 97, 227, 55}

	egressChan := make(chan []byte, 10)
	receiver, err := NewReceiver(context.Background(), cfg)
	check(err)
	rd, err := NewReedSolomon(10, 2)
	check(err)
	receiver.Run()

	var timestamp_conn net.Conn
	if send_telemetry {
		timestamp_conn, _ = net.Dial("udp", "192.168.1.151:19100")
	}
	go func() {
		prevBlock := -1
		for {
			db := <-receiver.egressChan
			if send_telemetry {
				blockid := fmt.Sprintf(`{"blockid":"%d"}`, db.blockID)
				timestamp_conn.Write([]byte(blockid))
			}
			if db.blockID != prevBlock+1 {
				if db.blockID == prevBlock+2 {
					log.Printf("block missed: %d (1)\n", prevBlock+1)
				} else {
					lastBlock := db.blockID - 1
					num := lastBlock - prevBlock
					log.Printf("blocks missed: %d-%d (%d)\n", prevBlock+1, lastBlock, num)
				}
			}
			prevBlock = db.blockID
			encodedData, err := db.GetEncodedData()
			checkNonFatal(err)
			data, err := rd.Decode(encodedData)
			checkNonFatal(err)
			egressChan <- data
		}
	}()
	go func() {
		for {
			data := <-egressChan
			hash := md5.Sum(data)
			if hash != test_image_hash {
				log.Println("hash mismatch")
			}
		}
	}()
}

func checkNonFatal(e error) {
	if e != nil {
		log.Println(e)
	}
}

func check(e error) {
	if e != nil {
		log.Fatal(e)
	}
}
