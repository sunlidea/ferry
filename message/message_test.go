package message

import (
	"bytes"
	"encoding/json"
	"github.com/google/go-cmp/cmp"
	"testing"
)

// Test Encode and Decode message Header
func TestHeader_EncodeAndDecode(t *testing.T) {
	header := Header{
		Version:      0,
		MessageType:  MsgTypeRequest,
		CompressType: NoneCompress,
		SeqID:        1,
		Extension:    0,
		BodyLength:   10,
	}

	hrd := EncodeHeader(header)

	h, err := DecodeHeader(bytes.NewReader(hrd))
	if err != nil {
		t.Errorf("DecodeHeader(%v) Fail", h)
		return
	}

	diff := cmp.Diff(header, h)
	if diff != "" {
		t.Fatalf(diff)
	}
}

// Test message encode and receive
func TestMessage_EncodeAndRecv(t *testing.T) {

	// create request
	name := "leo"
	age := 18
	req := Request{
		Path:   "Student",
		Method: "Register",
		Args:   []interface{}{name, age},
	}
	reqData, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("TestMessage_EncodeAndRecv|Marshal|Fail|%v", err)
		return
	}
	// create message
	msg := Message{
		Header: &Header{
			Version:      0,
			MessageType:  MsgTypeRequest,
			CompressType: NoneCompress,
			SeqID:        1,
			Extension:    0,
			BodyLength:   uint32(len(reqData)),
		},
		Data: reqData,
	}
	// encode message
	msgData := msg.Encode()

	// test receive message
	recvMsg, err := RecvMessage(bytes.NewReader(msgData))
	if err != nil {
		t.Fatalf("TestMessage_EncodeAndRecv|RecvMessage|Fail|%v", err)
		return
	}

	// compare
	diff := cmp.Diff(msg, *recvMsg)
	if diff != "" {
		t.Fatalf(diff)
	}
}
