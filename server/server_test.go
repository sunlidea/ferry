package server

import (
	"encoding/json"
	"github.com/sunlidea/ferry/message"
	"testing"
)

type Arith int

func (t *Arith) Add(A, B int) (int, error) {
	return A + B, nil
}

func (t *Arith) Mul(A, B int) (int, error) {
	return A * B, nil
}

// Test server register
func TestServer_Register(t *testing.T) {
	s := NewServer()
	err := s.Register(new(Arith))
	if err != nil {
		t.Fatalf("TestServer_Register|Register|Fail|%v", err.Error())
		return
	}
}

// Test server handle request
func TestServer_handleRequest(t *testing.T) {
	s := NewServer()
	err := s.Register(new(Arith))
	if err != nil {
		t.Fatalf("TestServer_handleRequest|Register|Fail|%v", err.Error())
		return
	}

	a := 3
	b := 5
	argA, _ := json.Marshal(a)
	argB, _ := json.Marshal(b)
	rawReq := &message.RawRequest{
		Path:   "Arith",
		Method: "Add",
		Args:   []json.RawMessage{argA, argB},
	}

	replys, err := s.handleRequest(rawReq)
	if err != nil || len(replys) != 2 {
		t.Fatalf("TestServer_handleRequest|handleRequest|Fail|%v|%+v",
			err.Error(), replys)
		return
	}

	if replys[1] != nil {
		t.Fatalf("TestServer_handleRequest|Add|Fail|%+v",
			replys[1])
	}

	if _, ok := replys[0].(int); !ok {
		t.Fatalf("TestServer_handleRequest|interface convert|Fail|%+v",
			replys[0])
		return
	} else if replys[0].(int) != 3+5 {
		t.Fatalf("TestServer_handleRequest|result|Fail|%+v",
			replys[0])
		return
	}
}
