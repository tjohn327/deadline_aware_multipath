package main

import (
	"context"
	"crypto/md5"
	"encoding/binary"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"runtime"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/vishvananda/netns"
)

var (
	errChan           = make(chan error, 1)
	send_telemetry    = true
	dataCount         = 20
	parityCount       = 0
	pktSize           = 1202
	udpSendAddr       = "127.0.0.1:5688"
	udpRecvAddr       = "127.0.0.1:5688"
	deadline          = time.Duration(200 * time.Millisecond)
	TEST_UDP          = true
	cs                *csvutil
	t0                = time.Now().UnixMilli()
	timedata          timeData
	delayms           = 20
	mode              = 3 // 0 udp 1 retr 2 fec 3 fec+retr
	runDuration       = time.Duration(10 * time.Second)
	DATA_COUNT        = 30
	MAX_PARITY        = 6
	receiveParityChan = make(chan TimeEntry, 2000)
	receiveRetrChan   = make(chan TimeEntry, 2000)
	streamHash        = make(map[int][16]byte)
	numStreams        = 2
	csvloss           = 0.0
	pktloss           = make(chan float64, 1000)
	run               = true

	// t0             = time.Now().UnixMilli()
)

func main() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)

	filename := flag.String("f", "entries.csv", "filename for csv")
	configFile := flag.String("c", "_example/sender.toml", "location of the config file")
	delay := flag.Int("d", 33, "delay in ms")
	duration := flag.Int("t", 10, "duration in seconds")
	modeFlag := flag.Int("m", 3, "mode")
	flag.Parse()

	mode = *modeFlag

	if mode == 4 {
		numStreams = 1
	}

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
	timedata.receiveParityChan = receiveParityChan
	timedata.receiveRetrChan = receiveRetrChan
	go timedata.run()

	t0 = time.Now().UnixMilli()

	go RunReceiver(&cfg)
	time.Sleep(500 * time.Millisecond)
	go RunSender(&cfg)

	time.Sleep(1 * time.Second)
	if mode == 4 {
		runLoss4()
	} else {
		runLoss1()
		runLoss2()
	}

	select {
	case <-sigCh:
		log.Println("\nterminating")
		run = false
		return
	case err := <-errChan:
		check(err)
		run = false
	case <-time.After(runDuration):
		errChan <- fmt.Errorf("timeout")
		run = false
		// timedata.PrintAvg()

		time.Sleep(2 * time.Second)
		timedata.SaveCSV()
		if mode != 4 {
			reset()
		}
		return
	}
}

func RunSender(cfg *Config) {
	log.Println("starting sender")
	// ingressChan := make(chan []byte, 2000)
	scheduler, err := NewScheduler(context.Background(), cfg, numStreams)
	check(err)
	manager, err := NewManager(cfg, scheduler, int(cfg.FragSize))
	check(err)
	// rd, err := NewReedSolomon(dataCount, parityCount)
	check(err)
	scheduler.Run()
	manager.Run()

	// test_image, err := ioutil.ReadFile("testdata/test_image.jpg")
	// check(err)

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
		// csvchan := cs.getInChan()

		decoders := make([]*ReedSolomon, MAX_PARITY+1)
		for i := 0; i <= MAX_PARITY; i++ {
			decoders[i], _ = NewReedSolomon(dataCount, i)
		}
		time.Sleep(1 * time.Millisecond)
		// t0 = time.Now().UnixMilli()
		var conn net.Conn
		if mode == 4 {
			conn, err = net.Dial("tcp", "10.1.0.2:5555")
			check(err)
		}

		//create n byte slice and fill random data
		frameSize := int(cfg.FragSize) * dataCount
		data := make([][]byte, numStreams)
		for i := 0; i < numStreams; i++ {
			data[i] = make([]byte, frameSize)
			for j := 0; j < frameSize; j++ {
				data[i][j] = byte(rand.Intn(256))
			}
		}
		// calculate hash of data
		for i := 0; i < numStreams; i++ {
			streamHash[i] = md5.Sum(data[i])
		}

		delay := time.Duration(time.Millisecond * time.Duration(delayms))
		for {
			// data := <-ingressChan
			t1 := time.Now().UnixMilli()

			for i := 0; i < numStreams; i++ {

				if mode == 4 {
					data := data[i]
					go func() {
						blockHeader := make([]byte, 4)
						binary.BigEndian.PutUint16(blockHeader[:2], uint16(blockID))
						binary.BigEndian.PutUint16(blockHeader[2:], uint16(i))
						timedata.sendChan <- TimeEntry{id: blockID, in: t1 - t0, streamID: i, loss: mainloss}
						buf := append(blockHeader, data...)
						_, err := conn.Write(buf)
						checkNonFatal(err)

					}()
				} else {

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
					split, err := Split(data[i], manager.fragSize)
					// split := &SplitData{
					// 	fragSize:      pktSize,
					// 	padlen:        0,
					// 	fragmentCount: dataCount,
					// 	data:          pkts,
					// }
					_parityCount := manager.GetParityCount()
					// receiveParityChan <- TimeEntry{id: blockID, parity: _parityCount}
					// _parityCount := 2
					if !doEncode {
						_parityCount = 0
					}
					if _parityCount > MAX_PARITY {
						_parityCount = MAX_PARITY
					}
					encodedData, err := decoders[_parityCount].Encode(split, _parityCount)
					checkNonFatal(err)
					retransmit := true
					if mode == 0 || mode == 2 {
						retransmit = false
					}
					if i == 0 {
						retransmit = true
					} else {
						retransmit = true
					}
					db := NewDataBlockFromEncodedDataOption(encodedData, blockID, retransmit, i)
					// if send_telemetry {
					// 	blockid := fmt.Sprintf(`{"blockid":"%d"}`, db.blockID)
					// 	log.Println("send", blockid)
					// 	timestamp_conn.Write([]byte(blockid))
					// }
					// entry := fmt.Sprintf("%d %d,send,%d,%d\n", db.streamID, db.blockID, t1-t0, db.parityCount)
					// fmt.Print(entry)
					// csvchan <- entry
					scheduler.Send(db)
					timedata.sendChan <- TimeEntry{id: db.blockID, in: t1 - t0, parity: db.parityCount, streamID: i}
				}
				time.Sleep(1 * time.Millisecond)
			}

			if blockID > 65534 {
				blockID = 0
			} else {
				blockID++
			}

			time.Sleep(delay)
			if run == false {
				break
			}
		}
	}()

	// // time.Sleep(1 * time.Second)
	// go func() {
	// 	// delay := time.Duration(int64(float64(cfg.Deadline.Duration)))
	// 	delay := time.Duration(time.Millisecond * time.Duration(delayms))
	// 	if mode == 4 {
	// 		blockID := 0
	// 		// csvchan := cs.getInChan()
	// 		for {
	// 			buf := make([]byte, len(test_image))
	// 			copy(buf, test_image)
	// 			blockHeader := make([]byte, 2)
	// 			binary.BigEndian.PutUint16(blockHeader, uint16(blockID))
	// 			buf = append(blockHeader, buf...)
	// 			t1 := time.Now().UnixMilli()

	// 			// entry := fmt.Sprintf("%d,send,%d,%d\n", t1-t0, blockID, 0)
	// 			// csvchan <- entry
	// 			timedata.sendChan <- TimeEntry{id: blockID, in: t1 - t0, parity: 0}
	// 			ingressChan <- buf
	// 			if blockID > 65534 {
	// 				blockID = 0
	// 			} else {
	// 				blockID++
	// 			}
	// 			time.Sleep(delay)
	// 		}
	// 	} else {

	// 		// for {
	// 		// 	buf := make([]byte, len(test_image))
	// 		// 	copy(buf, test_image)
	// 		// 	ingressChan <- buf
	// 		// 	time.Sleep(delay)
	// 		// }
	// 	}
	// }()

}

func RunReceiver(cfg *Config) {
	log.Println("starting receiver")

	// test_image_hash := [16]byte{105, 162, 92, 131, 191, 110, 214, 187, 153, 225, 26, 200, 95, 97, 227, 55}

	// egressChan := make(chan []byte, 500)

	if mode == 4 {
		go func() {
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
			listener, _ := net.Listen("tcp", "10.1.0.2:5555")
			conn, err := listener.Accept()
			checkNonFatal(err)
			for {
				buf := make([]byte, 2000000)
				n, err := conn.Read(buf)
				checkNonFatal(err)
				blockID := int(binary.BigEndian.Uint16(buf[:2]))
				streamID := int(binary.BigEndian.Uint16(buf[2:4]))
				data := buf[4:n]
				fmt.Println("received", n, mainloss, blockID)
				t1 := time.Now().UnixMilli()
				hash := md5.Sum(data)
				if hash != streamHash[streamID] {
					log.Println("hash mismatch")
					timedata.receiveChan <- TimeEntry{id: blockID, out: 0, streamID: streamID, loss: mainloss}
				} else {
					timedata.receiveChan <- TimeEntry{id: blockID, out: t1 - t0, streamID: streamID, loss: mainloss}
				}
				if run == false {
					break
				}
			}
		}()

	} else {
		receiver, err := NewReceiver(context.Background(), cfg, numStreams)
		check(err)
		// rd, err := NewReedSolomon(dataCount, parityCount)
		check(err)
		receiver.Run()

		// var timestamp_conn net.Conn
		// if send_telemetry {
		// 	timestamp_conn, _ = net.Dial("udp", "192.168.1.151:19100")
		// }
		go func() {
			// csvchan := cs.getInChan()
			decoders := make([]*ReedSolomon, MAX_PARITY+1)
			for i := 0; i <= MAX_PARITY; i++ {
				decoders[i], _ = NewReedSolomon(DATA_COUNT, i)
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
				// egressChan <- data
				hash := md5.Sum(data)
				if hash == streamHash[db.streamID] {
					t1 := time.Now().UnixMilli()
					timedata.receiveChan <- TimeEntry{id: db.blockID, out: t1 - t0, streamID: db.streamID}

					// entry := fmt.Sprintf("%d %d,receive,%d,%d\n", db.streamID, db.blockID, t1-t0, db.parityCount)
					// fmt.Println(entry)
					// csvchan <- entry
				} else {
					fmt.Println("hash mismatch")
				}

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
				if run == false {
					break
				}
			}
		}()
	}

	// go func() {
	// 	for {
	// 		csvchan := cs.getInChan()
	// 		data := <-egressChan
	// 		t1 := time.Now().UnixMilli()
	// 		if mode == 4 {
	// 			blockID := int(binary.BigEndian.Uint16(data[:2]))

	// 			hash := md5.Sum(data[2:])
	// 			d := t1 - t0
	// 			if hash != test_image_hash {
	// 				log.Println("hash mismatch")
	// 				d = 0
	// 			}
	// 			timedata.receiveChan <- TimeEntry{id: blockID, out: d}
	// 			entry := fmt.Sprintf("%d,receive,%d,%d\n", d, blockID, 0)
	// 			csvchan <- entry
	// 		} else {
	// 			hash := md5.Sum(data[2:])
	// 			if hash != test_image_hash {
	// 				log.Println("hash mismatch")
	// 			}
	// 		}
	// 	}
	// }()

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
