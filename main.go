package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

var (
	doneChan = make(chan error, 1)
)

func main() {
	configFile := flag.String("c", "", "location of the config file")
	flag.Parse()
	var config Config
	if _, err := toml.DecodeFile(*configFile, &config); err != nil {
		check(err)
	}
	//TODO implement main
}

func check(e error) {
	if e != nil {
		fmt.Fprintln(os.Stderr, "Fatal error:", e)
		os.Exit(1)
	}
}
