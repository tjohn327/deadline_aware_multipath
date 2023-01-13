package main

// gilbert-elliot loss model for udp packet loss simulation using netem for n seconds
// usage: go run lossmodel.go -n 10 -p 0.1 -s 1 -t 1 -i 1000
// -n: number of seconds to run the simulation
// -p: packet loss probability
// -s: seed for random number generator
// -t: number of threads to run
// -i: interval between packets in microseconds

// import (
// 	"flag"
// 	"fmt"
// 	"math/rand"
// 	"os"
// 	"os/exec"
// 	"strconv"
// 	"sync"
// 	"time"
// )

// var (
// 	// number of seconds to run the simulation
// 	n = flag.Int("n", 10, "number of seconds to run the simulation")
// 	// packet loss probability
// 	p = flag.Float64("p", 0.1, "packet loss probability")
// 	// seed for random number generator
// 	s = flag.Int64("s", 1, "seed for random number generator")
// 	// number of threads to run
// 	t = flag.Int("t", 1, "number of threads to run")
// 	// interval between packets in microseconds
// 	i = flag.Int("i", 1000, "interval between packets in microseconds")
// )

// func main() {
// 	flag.Parse()
// 	rand.Seed(*s)
// 	// start netem
// 	cmd := exec.Command
// 	cmd("sudo", "tc", "qdisc", "add", "dev", "lo", "root", "netem", "loss", "0%").Run()
// 	cmd("sudo", "tc", "qdisc", "change", "dev", "lo", "root", "netem", "loss", "0%").Run()
// 	// run simulation
// 	var wg sync.WaitGroup
// 	for j := 0; j < *t; j++ {
// 		wg.Add(1)
// 		go func() {
// 			defer wg.Done()
// 			// start netem
// 			cmd("sudo", "tc", "qdisc", "change", "dev", "lo", "root", "netem", "loss", "0%").Run()
// 			// run simulation
// 			start := time.Now()
// 			for time.Since(start) < time.Duration(*n)*time.Second {
// 				// generate packet loss
// 				loss := rand.Float64() < *p
// 				// set netem loss
// 				if loss {
// 					cmd
// 					("sudo
