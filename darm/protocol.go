package darm

import "time"

type darm interface {
	Dial(remote_addr string, deadline string,
		streamCount int, maxPktSize int) (Connection, error)
	NewGateway() error
}

type Connection interface {
	GetSendStream(streamID int) (SendStream, error)
	GetSendStreams() ([]SendStream, error)
	GetReceiveStream(streamID int) (ReceiveStream, error)
	GetReceiveStreams() ([]ReceiveStream, error)
}

type SendStream interface {
	Write(data []byte) (int, error)
}

type ReceiveStream interface {
	Read(data []byte) (int, error)
}

// 0                   1                   2                   3
// 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |      Type     |    StreamID   |           FrameID             |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |SubFrmID |SubCount |   FragID      |   FragCount   |ParityCount|
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |        Timestamp              |          PadLength            |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |                              Data                             |
// +                                                               +
// |                                                               |
// +                                                               +
// |                                                               |
// +                                               +-+-+-+-+-+-+-+-+
// |                                               |    Padding    |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+

type Packet struct {
	Type                int
	StreamID            int
	FrameID             int
	SubFrameID          int
	SubFrameCount       int
	FragID              int
	Fragcount           int
	ParityCount         int
	PadLength           int
	InTimestamp         time.Time
	OutTimestamp        time.Time
	RetransmitThreshold time.Time
	Acked               bool
	Data                []byte
}

// func Dial(remote_addr string, deadline string,
// 	streamCount int, maxPktSize int) (Connection, error) {
// 	return nil, nil
// }

// 0                   1                   2                   3
// 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |   Type:ACK    |    StreamID   |           FrameID             |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |SubFrmID |SubCount |   FragID      |   FragCount   |ParityCount|
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |        Timestamp              |        OutTimeStamp           |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |   Type:ACK    |    StreamID   |           FrameID             |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |SubFrmID |SubCount |   FragID      |   FragCount   |ParityCount|
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |        Timestamp              |        OutTimeStamp           |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
