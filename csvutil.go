package main

import (
	"bufio"
	"fmt"
	"os"
	"sort"
)

type timeEntry struct {
	id     int
	in     int64
	out    int64
	diff   int64
	parity int
	retr   int
}

type timeData struct {
	filename    string
	deadline    int
	sendChan    chan timeEntry
	receiveChan chan timeEntry
	entries     map[int]timeEntry
}

func NewTimeData(filename string, deadline int) timeData {
	return timeData{
		filename:    filename,
		deadline:    deadline,
		sendChan:    make(chan timeEntry, 200),
		receiveChan: make(chan timeEntry, 200),
		entries:     make(map[int]timeEntry),
	}
}

func (td *timeData) run() {
	for {
		select {
		case entry := <-td.sendChan:
			td.entries[entry.id] = entry
		case entry := <-td.receiveChan:
			if v, ok := td.entries[entry.id]; ok {
				v.out = entry.out
				v.diff = entry.out - v.in
				v.parity = entry.parity
				v.retr = entry.retr
				if v.diff == 0 {
					v.diff = 1
				}
				td.entries[entry.id] = v
			}
		case <-errChan:
			return
		}
	}
}

func (t *timeData) AddIn(id int, in int64) {
	t.entries[id] = timeEntry{id: id, in: in, out: 0, diff: 0}
}

func (t *timeData) AddOut(id int, out int64) {
	entry := t.entries[id]
	entry.out = out
	entry.diff = out - entry.in
	if entry.diff == 0 {
		entry.diff = 1
	}
	t.entries[id] = entry
}

func (t *timeData) PrintAvg() {
	sent := 0
	recv := 0
	keys := make([]int, 0, len(t.entries))
	for k := range t.entries {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	for _, entry := range t.entries {
		if entry.id > 100 && entry.id < keys[len(keys)-10] {
			sent += 1
			if entry.diff <= int64(t.deadline) && entry.diff > 0 {
				recv += 1
			}
		}
	}
	fmt.Printf("Average Sent: %d, Recv: %d, Avg: %f\n", sent, recv, float32(recv*100)/float32(sent))
}

func (t *timeData) SaveCSV() {
	file, err := os.Create(t.filename)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	keys := make([]int, 0, len(t.entries))
	for k := range t.entries {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	writer := bufio.NewWriter(file)
	defer writer.Flush()
	for _, k := range keys {
		entry := t.entries[k]
		writer.WriteString(fmt.Sprintf("%d,%d,%d,%d,%d,%d\n", entry.id, entry.in, entry.out, entry.diff, entry.parity, entry.retr))
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
