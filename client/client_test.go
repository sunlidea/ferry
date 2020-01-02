package client

import (
	"encoding/json"
	"fmt"
	"github.com/sunlidea/ferry/message"
	"reflect"
	"sync"
	"testing"
)

type ArithProxy struct {
	Add func(A, B int) (int, error)
	Mul func(A, B int) (int, error)
}

// test reflect
func TestReflect(t *testing.T) {
	var s ArithProxy

	rtype := reflect.TypeOf(s)
	if rtype.Kind() == reflect.Ptr {
		rtype = rtype.Elem()
	}
	instance := reflect.New(rtype)

	elem := instance.Elem()
	for i := 0; i < elem.NumField(); i++ {
		field := elem.Field(i)

		// check
		numOut := field.Type().NumOut()
		if numOut < 1 {
			panic(fmt.Sprintf("method %s field %s param num out %d invalid", rtype.Name(), field.Type().Name(), numOut))
		}
		if !field.Type().Out(numOut - 1).Implements(typeOfError) {
			panic(fmt.Sprintf("method %s field %s last out param not implements error", rtype.Name(), field.Type().Name()))
		}

		fmt.Println(field.CanSet(), field.Type().Kind(), field.Type().Name(), rtype.Field(i).Name)
	}
}

// test handleResponse
func TestClient_handleResponse(t *testing.T) {

	client := &Client{
		mutex:    sync.Mutex{},
		seq:      0,
		pending:  make(map[uint64]*Call),
		closing:  false,
		shutdown: false,
	}
	call := &Call{
		ServiceName: "Arith",
		MethodName:  "Add",
		Args:        []interface{}{1, 2},
		Done:        make(chan *Call, 1),
	}
	client.pending[client.seq] = call

	resp := message.Response{
		Result: []interface{}{3, nil},
	}
	respData, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("TestClient_handleResponse|Marshal|Fail|%v", err)
		return
	}
	msg := &message.Message{
		Header: &message.Header{
			MessageType:  message.MsgTypeResponse,
			CompressType: message.NoneCompress,
			SeqID:        client.seq,
			BodyLength:   uint32(len(respData)),
		},
		Data: respData,
	}

	client.handleResponse(msg)

	respCall := <-call.Done
	if len(respCall.Replys) != 2 {
		t.Fatalf("TestClient_handleResponse|call.Replys|Fail|%d|%+v", len(respCall.Replys), len(respCall.Replys))
		return
	}

	type reply struct {
		err error
		num int
	}
	var r reply
	err = json.Unmarshal(respCall.Replys[1], &r.err)
	if err != nil {
		t.Fatalf("TestClient_handleResponse|error|Fail|%v|%+v", err, respCall.Replys[1])
		return
	}
	err = json.Unmarshal(respCall.Replys[0], &r.num)
	if err != nil || r.num != 3 {
		t.Fatalf("TestClient_handleResponse|num|Fail|%v|%+v", err, respCall.Replys[0])
		return
	}
}
