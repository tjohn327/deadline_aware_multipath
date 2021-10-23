package main

type schedulerType string

const (
	Sender   schedulerType = "sender"
	Receiver               = "receiver"
)

type Scheduler struct {
	sender   ScionSender
	receiver ScionReceiver
}

func NewScheduler(cfg *Config) {
	if cfg.gwType == "sender" {

	}

}

func setupSender(cfg *Config) {

	sender := NewScionSender()
}
