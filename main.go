package main

import (
	"context"
	"crypto/md5"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"flag"
	"fmt"
	"log"
	"math/big"

	mrand "math/rand"
	"net"
	"os"
	"os/signal"
	"runtime"
	"time"

	quic "github.com/Abdoueck632/mp-quic"
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
	Timedata          timeData
	delayms           = 20
	mode              = 3 // 0 udp 1 retr 2 fec 3 fec+retr 4 mptcp 5 mpquic
	runDuration       = time.Duration(10 * time.Second)
	DATA_COUNT        = 20
	MAX_PARITY        = DATA_COUNT / 5
	receiveParityChan = make(chan TimeEntry, 2000)
	receiveRetrChan   = make(chan TimeEntry, 2000)
	streamHash        = make(map[int][16]byte)
	numStreams        = 1
	csvloss           = 0.0
	pktloss           = make(chan float64, 1000)
	run               = true

	// t0             = time.Now().UnixMilli()
)

func generateTLSConfig() *tls.Config {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		panic(err)
	}
	template := x509.Certificate{SerialNumber: big.NewInt(1)}
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		panic(err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		panic(err)
	}
	return &tls.Config{Certificates: []tls.Certificate{tlsCert}}
}

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
	// utils.SetLogLevel(utils.LogLevelInfo)

	// utils.SetLogLevel(utils.LogLevelDebug)

	if mode >= 4 {
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

	Timedata = NewTimeData(*filename, int(deadline.Milliseconds()))
	Timedata.receiveParityChan = receiveParityChan
	Timedata.receiveRetrChan = receiveRetrChan
	go Timedata.run()

	t0 = time.Now().UnixMilli()

	go RunReceiver(&cfg)
	time.Sleep(500 * time.Millisecond)
	go RunSender(&cfg)

	time.Sleep(1 * time.Second)
	if mode == 4 || true {
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
		Timedata.SaveCSV()
		if mode != 4 {
			reset()
		}
		return
	}
}

var blockIDR = 0

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
		var stream quic.Stream
		recvaddr := "10.1.0.2:5555"
		if mode == 4 {
			conn, err = net.Dial("tcp", recvaddr)
			check(err)

		} else if mode == 5 {
			quicConfig := &quic.Config{
				CreatePaths: true,
			}

			session, err := quic.DialAddr(recvaddr, &tls.Config{InsecureSkipVerify: true}, quicConfig)
			check(err)
			stream, err = session.OpenStreamSync()
			check(err)
		}

		//create n byte slice and fill random data
		frameSize := int(cfg.FragSize) * dataCount
		data := make([][]byte, numStreams)
		for i := 0; i < numStreams; i++ {
			data[i] = make([]byte, frameSize)
			for j := 0; j < frameSize; j++ {
				data[i][j] = byte(mrand.Intn(256))
			}
		}
		// calculate hash of data
		for i := 0; i < numStreams; i++ {
			streamHash[i] = md5.Sum(data[i])
		}
		time.Sleep(500 * time.Millisecond)

		delay := time.Duration(time.Millisecond * time.Duration(delayms))
		for {
			// data := <-ingressChan
			tsent := time.Now()
			t1 := time.Now().UnixMilli()

			for i := 0; i < numStreams; i++ {

				doEncode := false
				if mode > 1 && mode < 4 {
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
				checkNonFatal(err)
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
				} else {
					// _parityCount = MAX_PARITY
				}
				if _parityCount > MAX_PARITY {
					_parityCount = MAX_PARITY
				}
				// _parityCount = int(mainloss / 100 * float64(dataCount) * 1.1)
				encodedData, err := decoders[_parityCount].Encode(split, _parityCount)
				checkNonFatal(err)
				retransmit := true
				if mode == 0 || mode == 2 {
					retransmit = false
				}
				// if i == 0 {
				// 	retransmit = true
				// } else {
				// 	retransmit = true
				// }
				db := NewDataBlockFromEncodedDataOption(encodedData, blockID, retransmit, i)
				// if send_telemetry {
				// 	blockid := fmt.Sprintf(`{"blockid":"%d"}`, db.blockID)
				// 	log.Println("send", blockid)
				// 	timestamp_conn.Write([]byte(blockid))
				// }
				// entry := fmt.Sprintf("%d %d,send,%d,%d\n", db.streamID, db.blockID, t1-t0, db.parityCount)
				// fmt.Print(entry)
				// csvchan <- entry
				if mode >= 4 {
					fmt.Println("block", db.blockID, db.parityCount, db.fragmentCount)
					for i := 0; i < len(db.fragments); i++ {
						buf := db.fragments[i].data
						// go func(buf []byte) {
						//
						// fmt.Println("fragsent", db.fragments[i].blockID, db.fragments[i].fragmentID, db.fragments[i].fragmentCount, len(db.fragments[i].data))
						if mode == 4 {
							conn.Write(buf)
							// checkNonFatal(err)
						} else if mode == 5 {
							stream.Write(buf)
							// checkNonFatal(err)
						}
						// }(buf)
						// time.Sleep(5 * time.Microsecond)
					}
				} else {
					scheduler.Send(db)
				}
				Timedata.sendChan <- TimeEntry{id: db.blockID, in: t1 - t0, parity: db.parityCount, streamID: i, loss: mainloss}

			}

			if blockID > 65534 {
				blockID = 0
			} else {
				blockID++
			}
			blockIDR = blockID

			for {
				time.Sleep(1 * time.Millisecond)
				if time.Now().Sub(tsent) > delay {
					break
				}
			}
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
	decoders := make([]*ReedSolomon, MAX_PARITY+1)
	for i := 0; i <= MAX_PARITY; i++ {
		decoders[i], _ = NewReedSolomon(DATA_COUNT, i)
	}

	if mode >= 4 {
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

			recvqueue := NewDataQueue(Receive, nil, nil, nil, nil, 0)
			var conn net.Conn
			var stream quic.Stream
			if mode == 4 {
				listener, _ := net.Listen("tcp", "10.1.0.2:5555")
				conn, err = listener.Accept()
				checkNonFatal(err)
			} else if mode == 5 {
				listener, err := quic.ListenAddr("10.1.0.2:5555", generateTLSConfig(), nil)
				check(err)
				sess, err := listener.Accept()
				check(err)
				stream, err = sess.AcceptStream()
				check(err)
			}

			go func() {
				for {
					buf := make([]byte, 1210)
					var n int
					var err error
					if mode == 4 {
						n, err = conn.Read(buf)
					} else if mode == 5 {
						n, err = stream.Read(buf)
					}
					// checkNonFatal(err)
					if err != nil {
						continue
					}
					frag, err := NewFragmentFromBytes(buf[:n])
					checkNonFatal(err)
					if err != nil || frag.blockID > blockIDR+100 || n != 1210 {
						continue
					}
					// fmt.Println("fragrecv", frag.blockID, frag.fragmentID, frag.fragmentCount, frag.parityCount, len(frag.data))
					//
					if err == nil {
						recvqueue.ingressChan <- frag
					}
					if !run {
						break
					}
				}
			}()
			for {
				db := <-recvqueue.egressChan
				if !db.canDecode {
					fmt.Println("cant decode", db.blockID)
					fmt.Println(db.blockID, db.currentCount, db.fragmentCount, db.canDecode)
					continue
				}

				fmt.Println("recevd", db.blockID, db.currentCount, db.fragmentCount, db.canDecode)

				encodedData, err := db.GetEncodedData()
				checkNonFatal(err)
				data, err := decoders[0].Decode(encodedData)
				checkNonFatal(err)
				// egressChan <- data
				hash := md5.Sum(data)
				if hash == streamHash[db.streamID] {
					t1 := time.Now().UnixMilli()
					Timedata.ReceiveChan <- TimeEntry{id: db.blockID, out: t1 - t0, streamID: db.streamID}

					entry := fmt.Sprintf("%d %d,receive,%d,%d\n", db.streamID, db.blockID, t1-t0, db.parityCount)
					fmt.Println(entry)
					// csvchan <- entry
				} else {
					fmt.Println("hash mismatch")
				}
				if !run {
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
				if err != nil {
					continue
				}
				data, err := decoders[db.parityCount].Decode(encodedData)
				checkNonFatal(err)
				if err != nil {
					continue
				}
				// egressChan <- data
				hash := md5.Sum(data)
				if hash == streamHash[db.streamID] {
					t1 := time.Now().UnixMilli()
					Timedata.ReceiveChan <- TimeEntry{id: db.blockID, out: t1 - t0, streamID: db.streamID}

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
