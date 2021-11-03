package main

import (
	"context"
	"crypto/md5"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/BurntSushi/toml"
)

var (
	errChan = make(chan error, 1)
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
		break
	case "receiver":
		RunReceiver(&cfg)
		break
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
	manager, err := NewManager(cfg, scheduler, 1000)
	check(err)
	rd, err := NewReedSolomon(10, manager.parityCount)
	check(err)
	scheduler.Run()
	manager.Run()

	test_image, err := ioutil.ReadFile("testdata/test_image.jpg")
	check(err)

	go func() {
		blockID := 0
		for {
			data := <-ingressChan
			split, err := Split(data, manager.fragSize)
			checkNonFatal(err)
			encodedData, err := rd.Encode(split, manager.parityCount)
			checkNonFatal(err)
			db := NewDataBlockFromEncodedData(encodedData, blockID)
			scheduler.Send(db)
			if blockID > 65534 {
				blockID = 0
			} else {
				blockID++
			}
		}
	}()

	time.Sleep(10 * time.Second)
	go func() {
		for {
			buf := make([]byte, len(test_image))
			copy(buf, test_image)
			ingressChan <- buf
			time.Sleep(cfg.Deadline.Duration)
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
	go func() {
		prevBlock := -1
		for {
			db := <-receiver.egressChan
			if db.blockID != prevBlock+1 {
				log.Printf("block missed: %d\n", prevBlock+1)
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
