package client

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/sunlidea/ferry/message"
	"io"
	"log"
	"net"
	"reflect"
	"sync"
)

const (
	// read buffer size
	ReadSize = 1024
)

var typeOfError = reflect.TypeOf((*error)(nil)).Elem()
var ErrShutdown = errors.New("connection is shut down")

// Client represents the RPC client
type Client struct {
	mutex       sync.Mutex
	seq         uint64
	pending     map[uint64]*Call
	closing     bool // user has called Close
	shutdown    bool // server has told us to stop
	definition  interface{}
	serviceName string
	conn        net.Conn
}

// Call represents a RPC call
type Call struct {
	ServiceName string            // The name of the service
	MethodName  string            // The method
	Args        []interface{}     // The arguments to the function
	Replys      []json.RawMessage // The replys from the function
	Error       error             // After completion, the error status.
	Done        chan *Call        // Strobes when call is complete.
}

// Dail the remote service server
func Dail(network, address, serivceName string, definition interface{}) (*Client, error) {
	conn, err := net.Dial(network, address)
	if err != nil {
		return nil, err
	}
	return NewClient(conn, serivceName, definition), nil
}

func NewClient(conn net.Conn, serivceName string, definition interface{}) *Client {

	c := &Client{
		mutex:       sync.Mutex{},
		seq:         0,
		pending:     make(map[uint64]*Call),
		closing:     false,
		shutdown:    false,
		serviceName: serivceName,
		conn:        conn,
	}

	//build dynamic call
	rtype := reflect.TypeOf(definition)
	if rtype.Kind() == reflect.Ptr {
		rtype = rtype.Elem()
	}
	instance := reflect.New(rtype)
	c.definition = instance.Interface()

	//MakeFunc
	elem := instance.Elem()
	for i := 0; i < elem.NumField(); i++ {
		field := elem.Field(i)

		// check out params
		numOut := field.Type().NumOut()
		if numOut < 1 {
			panic(fmt.Sprintf("field %s field %s param num out %d invalid", rtype.Name(), rtype.Field(i).Name, numOut))
		}
		if !field.Type().Out(numOut - 1).Implements(typeOfError) {
			panic(fmt.Sprintf("field %s field %s last out param not implements error", rtype.Name(), rtype.Field(i).Name))
		}

		out := make([]reflect.Type, 0, field.Type().NumOut())
		for j := 0; j < field.Type().NumOut(); j++ {
			out = append(out, field.Type().Out(j))
		}

		name := rtype.Field(i).Name
		fn := func(in []reflect.Value) (results []reflect.Value) {
			return c.rpcInvoke(serivceName, name, in, out)
		}

		v := reflect.MakeFunc(field.Type(), fn)
		field.Set(v)
	}

	// get ready to accept message from the remote server
	go c.clientConn(conn)

	return c
}

// GetService returns a instance which represents the remote service definition
func (c *Client) GetService() interface{} {
	return c.definition
}

// rpcInvoke executes a RPC call
func (c *Client) rpcInvoke(serviceName string, methodName string, in []reflect.Value, out []reflect.Type) (results []reflect.Value) {

	// convert reflect.Value to Interface{}
	args := make([]interface{}, 0, len(in))
	for _, arg := range in {
		args = append(args, arg.Interface())
	}

	// register call
	call := new(Call)
	call.ServiceName = serviceName
	call.MethodName = methodName
	call.Args = args
	//TODO reply
	//call.Replys =
	call.Done = make(chan *Call, 1)

	// create sequence number of the call
	c.mutex.Lock()
	seq := c.seq
	c.seq++
	c.pending[seq] = call
	c.mutex.Unlock()

	// build request
	request := message.Request{
		Path:   serviceName,
		Method: call.MethodName,
		Args:   args,
	}

	// build message
	reqBody, err := json.Marshal(request)
	if err != nil {
		log.Print("ferry.rpcInvoke: Marshal Fail: ", err.Error())
		return nil
	}
	msg := message.Message{
		Header: &message.Header{
			Version:      0,
			MessageType:  message.MsgTypeRequest,
			CompressType: message.NoneCompress,
			BodyLength:   uint32(len(reqBody)),
			//ReqID
			SeqID: seq,
		},
		Data: reqBody,
	}

	//TODO handle error
	_, err = c.conn.Write(msg.Encode())
	if err != nil {
		log.Print("ferry.rpcInvoke: Write Fail: ", err.Error())
		return nil
	}

	// wait for the response
	respCall := <-call.Done

	// convert to reflect.Type
	replys := make([]reflect.Value, 0, len(respCall.Replys))
	for i, r := range respCall.Replys {
		inst := reflect.New(out[i])
		err = json.Unmarshal(r, inst.Interface())
		if err != nil {
			log.Print("ferry.rpcInvoke: Replys  Unmarshal  Fail: ", i, err.Error())
		}
		replys = append(replys, inst.Elem())
	}

	return replys
}

// clientConn reads the message from the conn
func (c *Client) clientConn(conn net.Conn) {

	var err error
	var msg *message.Message
	r := bufio.NewReaderSize(conn, ReadSize)
	for {

		//read message
		msg, err = message.RecvMessage(r)
		if err != nil {
			log.Print("ferry.clientConn: RecvMessage Fail: ", err.Error())
			break
		}

		go c.handleResponse(msg)
	}

	c.mutex.Lock()
	c.shutdown = true
	closing := c.closing
	if err == io.EOF {
		if closing {
			err = ErrShutdown
		} else {
			err = io.ErrUnexpectedEOF
		}
	}
	for _, call := range c.pending {
		call.Error = err
		call.done()
	}
	c.mutex.Unlock()
}

// handleResponse handles a RPC call response message
func (c *Client) handleResponse(msg *message.Message) {

	//find corresponding call
	c.mutex.Lock()
	call := c.pending[msg.SeqID]
	delete(c.pending, msg.SeqID)
	c.mutex.Unlock()

	if msg.MessageType != message.MsgTypeResponse {
		//not response message
		log.Print("ferry.clientConn: Invalid Message Type: ", msg.MessageType)
		return
	}

	// get response body
	resp, err := msg.DecodeResponse()
	if err != nil {
		log.Print("ferry.clientConn: DecodeRequest: ", err.Error())
		return
	}

	switch {
	case call == nil:
		//TODO
	case resp.Error != "":
		//TODO
		call.done()
	default:
		call.Replys = resp.Result
		call.done()
	}
}

func (call *Call) done() {
	select {
	case call.Done <- call:
		//ok
	default:
		log.Print("ferry.Call: done: discarding Call reply due to insufficient Done chan capacity")
	}
}
