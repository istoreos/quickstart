package message

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"net"
)

const (
	MsgTypePing = 1
	MsgTypePong = 2

	MsgTypeDhnsNewConn = 10
)

type Message struct {
	//ID   int         `json:"id"`
	Type int         `json:"type"`
	Msg  interface{} `json:"msg"`
}

type MessagePingPong struct {
	Msg string `json:"msg"`
}

type MessageDhnsNewConn struct {
	ConnID string `json:"connId"`
}

func ReadMessage(c net.Conn, data []byte) (int, int, error) {
	_, err := io.ReadFull(c, data[:4])
	if err != nil {
		return 0, 0, err
	}
	size := int(binary.LittleEndian.Uint16(data[:2]))
	dataType := int(binary.LittleEndian.Uint16(data[2:4]))
	if size > len(data)-4 {
		return 0, 0, errors.New("size to max")
	}
	if size > 4 {
		_, err = io.ReadFull(c, data[:size-4])
		if err != nil {
			return 0, dataType, err
		}
	}
	return size - 4, dataType, nil
}

func WriteMessage(c net.Conn, dataType uint16, data interface{}) error {
	header := make([]byte, 4)
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	binary.LittleEndian.PutUint16(header, uint16(len(jsonData)+4))
	binary.LittleEndian.PutUint16(header[2:4], dataType)
	_, err = c.Write(header)
	if err != nil {
		return err
	}
	_, err = c.Write(jsonData)
	return err
}
