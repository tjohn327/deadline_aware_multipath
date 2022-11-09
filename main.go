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
	errChan        = make(chan error, 1)
	send_telemetry = true
	dataCount      = 20
	parityCount    = 2
	pktSize        = 1202
	udpSendAddr    = "127.0.0.1:5688"
	udpRecvAddr    = "127.0.0.1:5688"
	deadline       = time.Duration(200 * time.Millisecond)
	TEST_UDP       = true
	cs             *csvutil
	t0             = time.Now().UnixMilli()
	timedata       timeData
	delayms        = 33
	mode           = 3 // 0 udp 1 retr 2 fec 3 fec+retr
	runDuration    = time.Duration(10 * time.Second)
	DATA_COUNT     = 159
	MAX_PARITY     = 30
	// t0             = time.Now().UnixMilli()
)

func main() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)

	filename := flag.String("f", "entries", "filename for csv")
	configFile := flag.String("c", "", "location of the config file")
	delay := flag.Int("d", 33, "delay in ms")
	duration := flag.Int("t", 10, "duration in seconds")
	modeFlag := flag.Int("m", 3, "mode")
	flag.Parse()

	mode = *modeFlag

	runDuration = time.Duration(*duration) * time.Second
	delayms = *delay
	cs = createCSVUtil("data.csv")
	go cs.run()

	flag.Parse()
	if *configFile == "" {
		check(fmt.Errorf("config file required"))
	}
	var cfg Config
	if _, err := toml.DecodeFile(*configFile, &cfg); err != nil {
		check(err)
	}
	udpRecvAddr = cfg.UdpRecv
	udpSendAddr = cfg.UdpSend
	pktSize = int(cfg.FragSize)
	dataCount = int(cfg.DataCount)
	parityCount = int(cfg.ParityCount)
	deadline = cfg.Deadline.Duration

	timedata = NewTimeData(*filename, int(deadline.Milliseconds()))
	go timedata.run()

	t0 = time.Now().UnixMilli()

	go RunReceiver(&cfg)
	time.Sleep(100 * time.Millisecond)
	go RunSender(&cfg)

	select {
	case <-sigCh:
		log.Println("\nterminating")
		return
	case err := <-errChan:
		check(err)
	case <-time.After(runDuration):
		errChan <- fmt.Errorf("timeout")
		timedata.PrintAvg()
		timedata.SaveCSV()
		time.Sleep(1 * time.Second)
		return
	}
}

func RunSender(cfg *Config) {
	log.Println("starting sender")
	ingressChan := make(chan []byte, 500)
	scheduler, err := NewScheduler(context.Background(), cfg)
	check(err)
	manager, err := NewManager(cfg, scheduler, int(cfg.FragSize))
	check(err)
	// rd, err := NewReedSolomon(dataCount, parityCount)
	check(err)
	scheduler.Run()
	manager.Run()

	test_image, err := ioutil.ReadFile("testdata/test_image.jpg")
	check(err)

	// var timestamp_conn net.Conn
	// if send_telemetry {
	// 	timestamp_conn, _ = net.Dial("udp", "192.168.1.151:19000")
	// }

	// udpConn, err := net.ListenPacket("udp", udpRecvAddr)
	// check(err)
	// //receive udp packets
	// go func() {
	// 	defer udpConn.Close()
	// 	for {
	// 		buf := make([]byte, pktSize)
	// 		n, _, err := udpConn.ReadFrom(buf[2:])
	// 		if err != nil {
	// 			continue
	// 		}
	// 		// handleRTPPacket(buf[2 : n+2])

	// 		if n < pktSize-2 {
	// 			for i := n + 2; i < pktSize; i++ {
	// 				buf[i] = 0x00
	// 			}
	// 		}
	// 		padlen := uint16(pktSize - n - 2)
	// 		binary.BigEndian.PutUint16(buf[:2], padlen)
	// 		ingressChan <- buf
	// 	}
	// }()
	// time.Sleep(1 * time.Second)
	go func() {
		blockID := 0
		csvchan := cs.getInChan()
		decoders := make(map[int]*ReedSolomon)
		for i := 0; i <= MAX_PARITY; i++ {
			decoders[i], _ = NewReedSolomon(dataCount, i)
		}
		time.Sleep(1 * time.Millisecond)
		// t0 = time.Now().UnixMilli()
		for {
			data := <-ingressChan
			t1 := time.Now().UnixMilli()

			doEncode := false
			if mode > 1 {
				doEncode = true
			}

			// pkts := make([][]byte, dataCount)
			// for i := 0; i < dataCount; i++ {
			// 	pkts[i] = <-ingressChan
			// 	// h, err := parseRTPH264Header(pkts[i][2:])
			// 	// check(err)
			// 	// if h.IsIDR {
			// 	// 	doEncode = true
			// 	// }
			// }
			split, err := Split(data, manager.fragSize)
			// split := &SplitData{
			// 	fragSize:      pktSize,
			// 	padlen:        0,
			// 	fragmentCount: dataCount,
			// 	data:          pkts,
			// }
			_parityCount := manager.parityCount
			// _parityCount := 2
			if !doEncode {
				_parityCount = 0
			}
			encodedData, err := decoders[_parityCount].Encode(split, _parityCount)
			checkNonFatal(err)
			retransmit := true
			if mode == 0 || mode == 2 {
				retransmit = false
			}
			db := NewDataBlockFromEncodedDataOption(encodedData, blockID, retransmit)
			// if send_telemetry {
			// 	blockid := fmt.Sprintf(`{"blockid":"%d"}`, db.blockID)
			// 	log.Println("send", blockid)
			// 	timestamp_conn.Write([]byte(blockid))
			// }
			entry := fmt.Sprintf("%d,send,%d,%d\n", t1-t0, db.blockID, db.parityCount)
			csvchan <- entry
			scheduler.Send(db)
			timedata.sendChan <- timeEntry{id: db.blockID, in: t1 - t0}
			if blockID > 65534 {
				blockID = 0
			} else {
				blockID++
			}
		}
	}()

	// time.Sleep(1 * time.Second)
	go func() {
		// delay := time.Duration(int64(float64(cfg.Deadline.Duration)))
		delay := time.Duration(time.Millisecond * time.Duration(delayms))
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

	egressChan := make(chan []byte, 500)
	receiver, err := NewReceiver(context.Background(), cfg)
	check(err)
	// rd, err := NewReedSolomon(dataCount, parityCount)
	check(err)
	receiver.Run()

	// var timestamp_conn net.Conn
	// if send_telemetry {
	// 	timestamp_conn, _ = net.Dial("udp", "192.168.1.151:19100")
	// }
	go func() {
		csvchan := cs.getInChan()
		decoders := make(map[int]*ReedSolomon)
		for i := 0; i <= MAX_PARITY; i++ {
			decoders[i], _ = NewReedSolomon(dataCount, i)
		}
		// prevBlock := -1
		for {
			db := <-receiver.egressChan
			if !db.canDecode {
				fmt.Println("cant decode", db.blockID)
				fmt.Println(db.blockID, db.currentCount, db.fragmentCount, db.canDecode)
				continue
			}

			// if db.blockID != prevBlock+1 {
			// 	if db.blockID == prevBlock+2 {
			// 		log.Printf("block missed: %d (1)\n", prevBlock+1)
			// 	} else {
			// 		lastBlock := db.blockID - 1
			// 		num := lastBlock - prevBlock
			// 		log.Printf("blocks missed: %d-%d (%d)\n", prevBlock+1, lastBlock, num)
			// 	}
			// }
			// if db.blockID < prevBlock {
			// 	continue
			// }
			// prevBlock = db.blockID
			// log.Printf("received block %d\n", db.blockID)
			encodedData, err := db.GetEncodedData()
			checkNonFatal(err)
			data, err := decoders[db.parityCount].Decode(encodedData)
			checkNonFatal(err)
			egressChan <- data
			t1 := time.Now().UnixMilli()
			timedata.receiveChan <- timeEntry{id: db.blockID, out: t1 - t0, parity: db.parityCount, retr: db.retrCount}
			entry := fmt.Sprintf("%d,receive,%d,%d\n", t1-t0, db.blockID, db.parityCount)
			csvchan <- entry
			// if send_telemetry {
			// 	blockid := fmt.Sprintf(`{"blockid":"%d"}`, db.blockID)
			// 	log.Println(blockid)
			// 	timestamp_conn.Write([]byte(blockid))
			// }
			// data, err := rd.DecodeRTP(encodedData)
			// checkNonFatal(err)
			// for i := range data {
			// 	padlen := binary.BigEndian.Uint16(data[i][:2])
			// 	if padlen > 0 {
			// 		end := len(data[i]) - int(padlen)
			// 		if end > 2 {
			// 			egressChan <- data[i][2:end]
			// 		} else {
			// 			egressChan <- data[i][2:]
			// 		}
			// 	} else {
			// 		egressChan <- data[i][2:]
			// 	}

			// }
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

	// // send out udp packets
	// go func() {
	// 	udpConn, err := net.Dial("udp", udpSendAddr)
	// 	check(err)
	// 	defer udpConn.Close()
	// 	for {
	// 		data := <-egressChan
	// 		udpConn.Write(data)
	// 		// checkNonFatal(err)
	// 	}
	// }()
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
