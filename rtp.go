package main

import (
	"fmt"
)

type ErrRTPH264HeaderTooShort struct{}

func (e ErrRTPH264HeaderTooShort) Error() string {
	return "RTPH264HeaderTooShort"
}

type rtpHeader struct {
	Version        uint8
	Padding        uint8
	Extension      uint8
	CSRCCount      uint8
	Marker         uint8
	PayloadType    uint8
	SequenceNumber uint16
	Timestamp      uint32
	SSRC           uint32
	FragmentType   uint8
	NalType        uint8
	StartBit       uint8
	IsIDR          bool
}

//handle rtp packet
func handleRTPPacket(buf []byte) {
	//parse rtp header
	h, err := parseRTPH264Header(buf)
	if err != nil {
		println(err)
		return
	}
	//print rtp header
	// printRTPH264Header(h)
	//fmt.Println("ISIDr:", h.IsIDR, "packet length:", len(buf), "start bit:", h.StartBit)
	fmt.Println(h.IsIDR, h.NalType, h.FragmentType, h.StartBit, len(buf))
}

//parse rtp h264 header
func parseRTPH264Header(buf []byte) (rtpHeader, error) {
	var h rtpHeader
	if len(buf) < 14 {
		return h, &ErrRTPH264HeaderTooShort{}
	}
	h.Version = (buf[0] & 0xC0) >> 6
	h.Padding = (buf[0] & 0x20) >> 5
	h.Extension = (buf[0] & 0x10) >> 4
	h.CSRCCount = buf[0] & 0xF
	h.Marker = (buf[1] & 0x80) >> 7
	h.PayloadType = buf[1] & 0x7F
	h.SequenceNumber = uint16(buf[2])<<8 | uint16(buf[3])
	h.Timestamp = uint32(buf[4])<<24 | uint32(buf[5])<<16 | uint32(buf[6])<<8 | uint32(buf[7])
	h.SSRC = uint32(buf[8])<<24 | uint32(buf[9])<<16 | uint32(buf[10])<<8 | uint32(buf[11])
	h.FragmentType = buf[12] & 0x1F
	h.NalType = buf[13] & 0x1F
	h.StartBit = (buf[13] & 0x80)

	//check for IDR frame
	if ((h.FragmentType == 28 || h.FragmentType == 29) && (h.NalType == 5)) || h.FragmentType == 5 {
		h.IsIDR = true
	} else {
		h.IsIDR = false
	}

	return h, nil
}

//pretty print rtp h264 header
func printRTPH264Header(h rtpHeader) {
	println("Version:", h.Version)
	println("Padding:", h.Padding)
	println("Extension:", h.Extension)
	println("CSRCCount:", h.CSRCCount)
	println("Marker:", h.Marker)
	println("PayloadType:", h.PayloadType)
	println("SequenceNumber:", h.SequenceNumber)
	println("Timestamp:", h.Timestamp)
	println("SSRC:", h.SSRC)
	println("FragmentType:", h.FragmentType)
	println("NalType:", h.NalType)
	println("StartBit:", h.StartBit)
	println("IsIDR:", h.IsIDR)
}
