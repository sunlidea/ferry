package message

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
)

// MessageType represents the message type for RPC
type MessageType byte

const (
	MsgTypeRequest MessageType = iota
	MsgTypeResponse
)

// CompressType represents compress method for the message
type CompressType byte

const (
	//without compress
	NoneCompress CompressType = iota
)

// Header represents message header
type Header struct {
	Version      byte
	MessageType  MessageType
	CompressType CompressType
	SeqID        uint64
	Extension    uint32
	BodyLength   uint32
}

const (
	headerLen = 1 + 1 + 1 + 8 + 4 + 4
)

// EncodeHeader encodes the message header
func EncodeHeader(h Header) []byte {
	data := make([]byte, headerLen)
	data[0] = h.Version
	data[1] = byte(h.MessageType)
	data[2] = byte(h.CompressType)
	binary.BigEndian.PutUint64(data[3:], h.SeqID)
	binary.BigEndian.PutUint32(data[11:], h.Extension)
	binary.BigEndian.PutUint32(data[15:], h.BodyLength)

	return data
}

// DecodeHeader reads and creates message header from reader stream
func DecodeHeader(r *bytes.Reader) (*Header, error) {

	//TODO 注意小对象h重用
	h := Header{}

	err := binary.Read(r, binary.BigEndian, &(h.Version))
	if err != nil {
		return nil, err
	}

	err = binary.Read(r, binary.BigEndian, &(h.MessageType))
	if err != nil {
		return nil, err
	}

	err = binary.Read(r, binary.BigEndian, &(h.CompressType))
	if err != nil {
		return nil, err
	}

	err = binary.Read(r, binary.BigEndian, &(h.SeqID))
	if err != nil {
		return nil, err
	}

	err = binary.Read(r, binary.BigEndian, &(h.Extension))
	if err != nil {
		return nil, err
	}

	err = binary.Read(r, binary.BigEndian, &(h.BodyLength))
	if err != nil {
		return nil, err
	}

	return &h, nil
}

// Message defines the basic message struct for RPC call
type Message struct {
	*Header
	Data []byte
}

// Encode writes the message to []byte
func (m *Message) Encode() []byte {
	buf := make([]byte, 0, headerLen+m.BodyLength)
	b := bytes.NewBuffer(buf)
	b.Write(EncodeHeader(*m.Header))
	b.Write(m.Data)

	return b.Bytes()
}

// RecvMessage reads and wraps the message from the reader
func RecvMessage(r io.Reader) (*Message, error) {
	//TODO reuse message object

	buff := make([]byte, headerLen)
	_, err := io.ReadFull(r, buff)
	if err != nil {
		return nil, err
	}

	header, err := DecodeHeader(bytes.NewReader(buff))
	if err != nil {
		return nil, err
	}

	bodyData := make([]byte, header.BodyLength)
	_, err = io.ReadFull(r, bodyData)
	if err != nil {
		return nil, err
	}

	return &Message{
		Header: header,
		Data:   bodyData,
	}, nil
}

// Request represents the basic request struct for RPC call
type Request struct {
	Path   string        `json:"path"`   //ServicePath
	Method string        `json:"method"` //Method
	Args   []interface{} `json:"args"`   //in args
}

// RawRequest represents the raw request struct for RPC call
type RawRequest struct {
	Path   string            `json:"path"`   //ServicePath
	Method string            `json:"method"` //Method
	Args   []json.RawMessage `json:"args"`   //in args
}

// Response represents the basic response struct for RPC call
type Response struct {
	ErrCode uint          `json:"code"`
	Error   string        `json:"error"`
	Result  []interface{} `json:"result"` //out args
}

// RawResponse represents the raw response struct for RPC call
type RawResponse struct {
	ErrCode uint              `json:"code"`
	Error   string            `json:"error"`
	Result  []json.RawMessage `json:"result"` //out args
}

// DecodeRequest gets the RPC request body
func (m *Message) DecodeRequest() (*RawRequest, error) {
	if m == nil || m.MessageType != MsgTypeRequest || len(m.Data) <= 0 {
		// invalid input
		return nil, fmt.Errorf("invalid message")
	}

	var req RawRequest
	err := json.Unmarshal(m.Data, &req)
	if err != nil {
		return nil, err
	}

	return &req, nil
}

// DecodeResponse gets the RPC response body
func (m *Message) DecodeResponse() (*RawResponse, error) {
	if m == nil || m.MessageType != MsgTypeResponse || len(m.Data) <= 0 {
		// invalid input
		return nil, fmt.Errorf("invalid message")
	}

	var resp RawResponse
	err := json.Unmarshal(m.Data, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}
