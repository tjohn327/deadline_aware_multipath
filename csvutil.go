package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"sort"
	"time"
)

var mainloss = 0.0

type TimeEntry struct {
	id       int
	in       int64
	out      int64
	diff     int64
	parity   int
	retr     int
	loss     float64
	path1    int
	path2    int
	streamID int
}

var (
	lastBandwidthMeasurement time.Time
	lastBandwidthsIN         map[string]int
	lastBandwidthsOUT        map[string]int
)

type timeData struct {
	filename          string
	deadline          int
	sendChan          chan TimeEntry
	ReceiveChan       chan TimeEntry
	receiveLossChan   chan TimeEntry
	receiveParityChan chan TimeEntry
	receiveRetrChan   chan TimeEntry
	entries           map[int]map[int]TimeEntry
	startTime         time.Time
}

func NewTimeData(filename string, deadline int) timeData {
	start := time.Now()
	return timeData{
		filename:    filename,
		deadline:    deadline,
		sendChan:    make(chan TimeEntry, 2000),
		ReceiveChan: make(chan TimeEntry, 2000),
		entries:     make(map[int]map[int]TimeEntry),
		startTime:   start,
	}
}

func (td *timeData) run() {
	lastBandwidthMeasurement = time.Now()
	ifaces := []string{"veth0", "veth2"}
	lastBandwidthsIN = make(map[string]int)
	lastBandwidthsOUT = make(map[string]int)
	for _, iface := range ifaces {
		lastBandwidthsIN[iface] = 0
		lastBandwidthsOUT[iface] = 0
	}
	getBandwidthUsage(ifaces)
	for {
		select {
		case entry := <-td.sendChan:
			if _, ok := td.entries[entry.streamID]; !ok {
				td.entries[entry.streamID] = make(map[int]TimeEntry)
				td.entries[entry.streamID][entry.id] = entry
			} else {
				td.entries[entry.streamID][entry.id] = entry
			}
		case entry := <-td.ReceiveChan:
			if v, ok := td.entries[entry.streamID][entry.id]; ok {
				v.out = entry.out
				if v.out == 0 {
					v.diff = 1
				} else {
					v.diff = entry.out - v.in
				}
				if v.diff == 0 {
					v.diff = 1
				}
				bws := getBandwidthUsage(ifaces)
				v.path1 = int(bws[0])
				v.path2 = int(bws[1])
				td.entries[entry.streamID][entry.id] = v

				fmt.Println("bw", v.path1, v.path2)
			}
		case entry := <-td.receiveParityChan:
			if v, ok := td.entries[entry.streamID][entry.id]; ok {
				v.parity = entry.parity
				td.entries[entry.streamID][entry.id] = v
			}
		case entry := <-td.receiveRetrChan:
			if v, ok := td.entries[entry.streamID][entry.id]; ok {
				v.retr = entry.retr
				// v.loss = entry.loss
				// v.loss = mainloss
				// v.loss = int(csvloss)
				// v.loss = int(csvloss * 100)
				td.entries[entry.streamID][entry.id] = v
			}
			// case <-errChan:
			// 	return
		}
	}
}

func stringToFloat64(s string) float64 {
	var result float64
	fmt.Sscanf(s, "%f", &result)
	return result
}

// get the bandwidth usage in bytes per second for the given interface
func getBandwidthUsage(ifaces []string) []float64 {

	var bandwidthUsage []float64
	// get the current time
	now := time.Now()
	// get the time since the last measurement
	elapsed := now.Sub(lastBandwidthMeasurement)
	// update the last measurement time
	lastBandwidthMeasurement = now
	for _, iface := range ifaces {
		// get the bytes sent and received since the last measurement
		out, err := exec.Command("cat", "/sys/class/net/"+iface+"/statistics/tx_bytes").Output()
		if err != nil {
			panic(err)
		}
		in, err := exec.Command("cat", "/sys/class/net/"+iface+"/statistics/rx_bytes").Output()
		if err != nil {
			panic(err)
		}
		// calculate the bandwidth usage in bytes per second
		outBytes := stringToFloat64(string(out))
		outB := outBytes - float64(lastBandwidthsOUT[iface])
		lastBandwidthsOUT[iface] = int(outBytes)
		inBytes := stringToFloat64(string(in))
		inB := inBytes - float64(lastBandwidthsIN[iface])
		lastBandwidthsIN[iface] = int(inBytes)
		bandwidthUsage = append(bandwidthUsage, ((outB+inB)/elapsed.Seconds())/1000)
	}
	return bandwidthUsage
}

func (t *timeData) AddIn(streamID int, id int, in int64) {
	t.entries[streamID][id] = TimeEntry{streamID: streamID, id: id, in: in, out: 0, diff: 0}
}

func (t *timeData) AddOut(streamID int, id int, out int64) {
	entry := t.entries[streamID][id]
	entry.out = out
	entry.diff = out - entry.in
	if entry.diff == 0 {
		entry.diff = 1
	}
	t.entries[streamID][id] = entry
}

func (t *timeData) PrintAvg() {
	sent := 0
	recv := 0
	keys := make([]int, 0, len(t.entries))
	for k := range t.entries {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	for _, k := range keys {
		ids := make([]int, 0, len(t.entries[k]))
		for id := range t.entries[k] {
			ids = append(ids, id)
		}
		sort.Ints(ids)
		for _, id := range ids {
			entry := t.entries[k][id]
			if entry.id > 100 && entry.id < keys[len(ids)-10] {
				sent += 1
				if entry.diff <= int64(t.deadline) && entry.diff > 0 {
					recv += 1
				}
			}
		}
	}
	fmt.Printf("Average Sent: %d, Recv: %d, Avg: %f\n", sent, recv, float32(recv*100)/float32(sent))
}

func (t *timeData) SaveCSV() {

	keys := make([]int, 0, len(t.entries))
	for k := range t.entries {
		keys = append(keys, k)
	}
	sort.Ints(keys)

	for _, k := range keys {

		filename := fmt.Sprintf("%s_%d.csv", t.filename, k)
		file, err := os.Create(filename)
		if err != nil {
			panic(err)
		}
		defer file.Close()
		writer := bufio.NewWriter(file)
		defer writer.Flush()
		if err != nil {
			panic(err)
		}
		defer file.Close()
		ids := make([]int, 0, len(t.entries[k]))
		for id := range t.entries[k] {
			ids = append(ids, id)
		}
		sort.Ints(ids)
		for _, id := range ids {
			entry := t.entries[k][id]

			// 	continue
			// }
			writer.WriteString(fmt.Sprintf("%d,%d,%d,%d,%d,%d,%d,%f,%f,%d,%d\n",
				entry.streamID, entry.id, entry.in, entry.out, entry.diff, (entry.parity*100)/DATA_COUNT,
				(entry.retr*100)/DATA_COUNT, entry.loss,
				float32(entry.parity+entry.retr)*100/float32(DATA_COUNT),
				entry.path1, entry.path2))

		}
	}
}

type csvutil struct {
	in     chan string
	file   *os.File
	writer *bufio.Writer
}

func createCSVUtil(filename string) *csvutil {
	c := new(csvutil)
	c.in = make(chan string, 200)
	var err error
	c.file, err = os.Create(filename)
	if err != nil {
		panic(err)
	}
	c.writer = bufio.NewWriter(c.file)
	return c
}

func (c *csvutil) run() {
	defer c.writer.Flush()
	defer c.file.Close()
	for {
		select {
		case s := <-c.in:
			c.writer.WriteString(s)
			c.writer.Flush()
		case <-errChan:
			return
		}
	}
}

func (c *csvutil) getInChan() chan string {
	return c.in
}

var test *[]byte

// get random number between 0 and 0.5 from Zipf distribution with 2 decimal places
func getZipf() float64 {
	zipf := rand.NewZipf(rand.New(rand.NewSource(time.Now().UnixNano())), 1.5, 1, 100)
	return float64(zipf.Uint64()) / 100
}

func runLoss1() {

	go func() {
		startTime := time.Now()
		for {
			loss := getZipf()
			fmt.Println("Loss: ", loss)
			command := fmt.Sprintf("sudo tcset veth2 --delay 60ms --rate 100mbps --change --loss %f%%", loss)
			cmd := exec.Command("bash", "-c", command)
			if loss < 0.45 {
				cmd.Run()
				mainloss = loss
				time.Sleep(200 * time.Millisecond)
			} else {
				command := fmt.Sprintf("sudo tcset veth2 --delay 60ms --rate 100mbps --change --loss %f%%", 100.0)
				cmd := exec.Command("bash", "-c", command)
				cmd.Run()
				mainloss = 100.0
				time.Sleep(2 * time.Second)
			}
			if time.Since(startTime) > 40*time.Second {
				reset()
				return
			}
		}
	}()
}

func runLoss2() {
	go func() {
		startTime := time.Now()
		for {
			loss := getZipf()
			command := fmt.Sprintf("sudo tcset veth0 --delay 60ms --rate 100mbps --change --loss %f%%", loss)
			cmd := exec.Command("bash", "-c", command)
			cmd.Run()
			time.Sleep(200 * time.Millisecond)
			if time.Since(startTime) > 40*time.Second {
				reset()
				return
			}
		}
	}()
}

func runLoss4() {
	go func() {
		losses := []float64{0.0, 0.01, 0.05, 0.5, 1.0, 2.0, 5.0, 10.0}
		for i := 0; i < len(losses); i++ {
			inf := "veth2"
			if mode >= 4 {
				inf = "veth0"
			}
			command := fmt.Sprintf("sudo tcset %s --delay 60ms --rate 100mbps --change --loss %f%%", inf, losses[i])
			mainloss = losses[i]
			cmd := exec.Command("bash", "-c", command)
			err := cmd.Run()
			if err != nil {
				fmt.Println(err)
			}
			fmt.Printf("Loss: %f\n", losses[i])
			time.Sleep(5 * time.Second)
		}
		reset()
	}()
}

func reset() {
	command := fmt.Sprintf("sudo tcset veth2 --delay 60ms --rate 100mbps --change --loss %f%%", 0.0)
	cmd := exec.Command("bash", "-c", command)
	cmd.Run()
	mainloss = 0.0
	command = fmt.Sprintf("sudo tcset veth0 --delay 60ms --rate 100mbps --change --loss %f%%", 0.0)
	cmd = exec.Command("bash", "-c", command)
	cmd.Run()

}
