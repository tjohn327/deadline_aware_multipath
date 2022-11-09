package main

// import (
// 	"encoding/binary"
// 	"fmt"
// 	"time"
// )

// type Packet struct {
// 	Type          int
// 	StreamID      int
// 	FrameID       int
// 	SubFrameID    int
// 	SubFrameCount int
// 	FragID        int
// 	Fragcount     int
// 	ParityCount   int
// 	PadLength     int
// 	InTimestamp   time.Time
// 	OutTimestamp  time.Time
// 	Acked         bool
// 	Data          []byte
// }

// func decode_pkt(buf []byte) (Packet, error) {
// 	if len(buf) < 12 {
// 		return Packet{}, fmt.Errorf("Invalid packet")
// 	}
// 	// header := make([]byte, headerSize)
// 	// header[0] = byte(0)
// 	// binary.BigEndian.PutUint16(header[1:3], uint16(db.blockID))
// 	// header[3] = byte(uint8(fragID))
// 	// header[4] = byte(uint8(db.fragmentCount))
// 	// header[5] = byte(uint8(db.parityCount))
// 	// binary.BigEndian.PutUint16(header[6:8], uint16(db.padlen))
// 	pkt_type := uint8(buf[0])
// 	streamID := buf[1]
// 	frameID := binary.BigEndian.Uint16(buf[2:4])
// 	header_2 := binary.BigEndian.Uint32(buf[4:8])
// 	subFrameID := uint8(header_2 >> 27)
// 	tmp := header_2 << 5
// 	tmp = tmp >> 27
// 	subFrameCount := uint8(tmp)
// 	tmp = header_2 << 10
// 	tmp = tmp >> 26
// 	fragID := uint8(tmp)
// 	tmp = header_2 << 18
// 	tmp = tmp >> 26
// 	fragCount := uint8(tmp)
// 	tmp = header_2 << 26
// 	tmp = tmp >> 27
// 	parityCount := uint8(tmp)

// 	inTime := binary.BigEndian.Uint16(buf[8:10])
// 	timestamp := uint64(time.Now())
// 	timestamp = timestamp & 0xFFFFFFFFFFFF0000
// 	timestamp = timestamp | uint64(inTime)
// 	inTimestamp := time.Unix(0, timestamp*(time.Millisecond))
// 	padlen := binary.BigEndian.Uint16(buf[10:12])
// 	pkt := Packet{
// 		Type:          int(pkt_type),
// 		StreamID:      int(streamID),
// 		FrameID:       int(frameID),
// 		SubFrameID:    int(subFrameID),
// 		SubFrameCount: int(subFrameCount),
// 		FragID:        int(fragID),
// 		Fragcount:     int(fragCount),
// 		ParityCount:   int(parityCount),
// 		PadLength:     int(padlen),
// 		InTimestamp:   inTimestamp,
// 		Data:          buf,
// 	}
// 	if pkt_type == 1 {
// 		timestamp = timestamp & 0xFFFFFFFFFFFF0000
// 		timestamp = timestamp | uint64(padlen)
// 		outTimestamp := time.Unix(0, padlen*uint64(time.Millisecond))
// 		pkt.OutTimestamp = outTimestamp
// 	}

// 	return pkt, nil
// }
