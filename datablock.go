package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"sync"
	"time"
)

type DataBlock struct {
	complete            bool
	canDecode           bool
	blockID             int
	fragmentCount       int
	parityCount         int
	padlen              int
	streamID            int
	currentCount        int
	unackedCount        int
	retransmissionCount int
	restransmit         bool
	inTime              time.Time
	mutex               sync.Mutex
	fragments           []*DataFragment
}

func newDataBlock(blockID int, fragmentCount int, streamID int, retransmit bool) *DataBlock {
	return &DataBlock{
		blockID:       blockID,
		fragmentCount: fragmentCount,
		streamID:      streamID,
		restransmit:   retransmit,
		fragments:     make([]*DataFragment, fragmentCount),
	}
}

func NewDataBlockFromEncodedData(ed *EncodedData, blockID int, streamID int) *DataBlock {
	db := newDataBlock(blockID, ed.dataCount+ed.parityCount, streamID, false)
	db.padlen = ed.padlen
	db.parityCount = ed.parityCount
	common_header := CreateHeader(db, 0)
	for i := 0; i < db.fragmentCount; i++ {
		UpdateFragID(common_header, i)
		header := make([]byte, headerSize)
		// isIDR := true
		// h, err := parseRTPH264Header(ed.data[i][2:])
		// if err != nil {
		// 	if h.IsIDR {
		// 		isIDR = true
		// 		header[0] = IDR
		// 	}
		// }
		isParity := false
		copy(header, common_header)
		if ed.parityCount > 0 {
			if i >= ed.dataCount {
				isParity = true
			}
		}

		fragment := &DataFragment{
			fragType:      int(header[0]),
			blockID:       db.blockID,
			fragmentID:    i,
			fragmentCount: db.fragmentCount,
			streamID:      db.streamID,
			parityCount:   db.parityCount,
			padlen:        db.padlen,
			isParity:      isParity,
			data:          append(header, ed.data[i]...),
		}
		db.fragments[i] = fragment
	}
	db.currentCount = db.fragmentCount
	db.unackedCount = db.fragmentCount - db.parityCount
	db.complete = true
	db.canDecode = true
	return db
}

func NewDataBlockFromEncodedDataOption(ed *EncodedData, blockID int, retransmit bool, streamID int) *DataBlock {
	db := newDataBlock(blockID, ed.dataCount+ed.parityCount, streamID, retransmit)
	db.padlen = ed.padlen
	db.parityCount = ed.parityCount
	common_header := CreateHeader(db, 0)
	for i := 0; i < db.fragmentCount; i++ {
		UpdateFragID(common_header, i)
		header := make([]byte, headerSize)
		isParity := false
		copy(header, common_header)
		if i >= ed.dataCount {
			isParity = true
		}
		fragment := &DataFragment{
			fragType:      int(header[0]),
			blockID:       db.blockID,
			fragmentID:    i,
			fragmentCount: db.fragmentCount,
			parityCount:   db.parityCount,
			streamID:      db.streamID,
			padlen:        db.padlen,
			retransmit:    retransmit,
			isParity:      isParity,
			data:          append(header, ed.data[i]...),
		}
		db.fragments[i] = fragment
	}
	db.currentCount = db.fragmentCount
	db.unackedCount = db.fragmentCount - db.parityCount
	db.complete = true
	db.canDecode = true
	return db
}

func NewDataBlockFromFragment(frag *DataFragment) *DataBlock {
	db := &DataBlock{
		blockID:       frag.blockID,
		fragmentCount: frag.fragmentCount,
		unackedCount:  frag.fragmentCount,
		parityCount:   frag.parityCount,
		streamID:      frag.streamID,
		padlen:        frag.padlen,
		inTime:        time.Now(),
		fragments:     make([]*DataFragment, frag.fragmentCount),
	}
	_, err := db.InsertFragment(frag)
	if err != nil {
		log.Println(err)
	}
	return db
}

func (db *DataBlock) AcknowledgeFragment(f *DataFragment) {
	if db.blockID != f.blockID {
		return
	}
	if f.fragmentID < db.fragmentCount {
		if !db.fragments[f.fragmentID].acked {
			db.fragments[f.fragmentID].acked = true
			db.unackedCount--
		}
	}

}

func (db *DataBlock) InsertFragment(fragment *DataFragment) (bool, error) {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	if fragment.blockID != db.blockID {
		return false, fmt.Errorf("frameID mismatch")
	}
	if fragment.fragmentID >= db.fragmentCount {
		return false, fmt.Errorf("invalid fragment ID")
	}
	if db.fragments[fragment.fragmentID] == nil {
		db.fragments[fragment.fragmentID] = fragment
		db.currentCount++
		if fragment.isRetr {
			db.retransmissionCount++
		}
		if db.currentCount == db.fragmentCount-db.parityCount {
			db.canDecode = true
		}
		if db.fragmentCount == db.currentCount {
			db.canDecode = true
			db.complete = true
			return true, nil
		}
	}
	return false, nil
}

func (db *DataBlock) GetEncodedData() (*EncodedData, error) {
	if !db.canDecode {
		err := fmt.Errorf("too few fragments to successfully decode")
		return nil, err
	}
	ed := &EncodedData{
		dataCount:   db.fragmentCount - db.parityCount,
		parityCount: db.parityCount,
		padlen:      db.padlen,
		data:        make([][]byte, db.fragmentCount),
	}
	for i, v := range db.fragments {
		if v != nil {
			ed.data[i] = v.data[headerSize:]
		} else {
			ed.data[i] = nil
		}
	}
	// ed.fragSize = len(ed.data[0])
	return ed, nil
}

// func (db *DataBlock) removeFragmentByID(fragID int) bool {
// 	for i, v := range db.fragments {
// 		if v.fragmentID == fragID {
// 			db.fragments = append(db.fragments[:i], db.fragments[i+1:]...)
// 			db.currentCount--
// 			return true
// 		}
// 	}
// 	return false
// }

const headerSize = 10

type DataFragment struct {
	isParity        bool
	isIDR           bool
	isRetr          bool
	acked           bool
	retransmit      bool
	fragType        int
	streamID        int
	blockID         int
	fragmentID      int
	fragmentCount   int
	parityCount     int
	padlen          int
	retransmitCount int
	data            []byte
}

// 0                   1                   2                   3
// 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |      Type     |   StreamID    |             FrameID           |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |     FragID    |    FragCount  |  ParityCount  |   PadLength   |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |                              Data                             |
// +                                                               +
// |                                                               |
// +                                                               +
// |                                                               |
// +                                               +-+-+-+-+-+-+-+-+
// |                                               |    Padding    |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+

const (
	NORETRANSMIT = byte(0)
	IDR          = byte(1)
)

func CreateHeader(db *DataBlock, fragID int) []byte {
	header := make([]byte, headerSize)
	header[0] = byte(0)
	binary.BigEndian.PutUint16(header[1:3], uint16(db.blockID))
	header[3] = byte(uint8(fragID))
	header[4] = byte(uint8(db.fragmentCount))
	header[5] = byte(uint8(db.parityCount))
	binary.BigEndian.PutUint16(header[6:8], uint16(db.padlen))
	binary.BigEndian.PutUint16(header[8:10], uint16(db.streamID))

	return header
}
func UpdateFragID(header []byte, fragID int) {
	header[3] = byte(uint8(fragID))
}

func NewFragmentFromBytes(data []byte) (*DataFragment, error) {
	if len(data) < headerSize {
		err := fmt.Errorf("data too small")
		return nil, err
	}
	return &DataFragment{
		fragType:      int(data[0]),
		blockID:       int(binary.BigEndian.Uint16(data[1:3])),
		fragmentID:    int(data[3]),
		fragmentCount: int(data[4]),
		parityCount:   int(data[5]),
		padlen:        int(binary.BigEndian.Uint16(data[6:8])),
		streamID:      int(binary.BigEndian.Uint16(data[8:10])),
		acked:         false,
		data:          data,
	}, nil
}

func (f *DataFragment) GetAckBytes() []byte {
	buf := make([]byte, headerSize)
	copy(buf, f.data[:headerSize])
	return buf
}
