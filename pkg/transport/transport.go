package transport

import (
	"encoding/json"
	"sync/atomic"

	"github.com/outrigdev/outrig/pkg/global"
)

const (
	PacketTypeLog = "log"
)

type PacketType struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

func SendPacket(pk *PacketType) (bool, error) {
	if atomic.LoadInt32(&global.OutrigEnabled) == 0 {
		return false, nil
	}
	client := global.ClientPtr.Load()
	if client == nil {
		return false, nil
	}
	barr, err := json.Marshal(pk)
	if err != nil {
		return false, err
	}
	barr = append(barr, '\n')
	_, err = client.Conn.Write(barr)
	if err != nil {
		atomic.AddInt64(&global.TransportErrors, 1) // this will force a disconnect later
		return false, nil
	}
	atomic.AddInt64(&global.TransportPacketsSent, 1)
	return true, nil
}
