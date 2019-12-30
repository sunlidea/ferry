package server

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
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	// read buffer size
	ReadSize = 1024
)

var typeOfError = reflect.TypeOf((*error)(nil)).Elem()

// service consists of the methods providing by the service
type service struct {
	name   string                 // name of service
	rcvr   reflect.Value          // receiver of methods for the service
	typ    reflect.Type           // type of the receiver
	method map[string]*methodType // registered methods
}

// methodType represents the information of a method of a service.
type methodType struct {
	sync.Mutex // protects counters
	method     reflect.Method
	ArgTypes   []reflect.Type
	ReplyTypes []reflect.Type
	numCalls   uint
}

// Server is a RPC server to serve RPC requests
type Server struct {
	serviceMapMu sync.RWMutex
	serviceMap   map[string]*service // map[string]*service
}

// NewServer creates the RPC server instance
func NewServer() *Server {
	return &Server{
		serviceMap: make(map[string]*service),
	}
}

// Register publishes the receiver's methods in the Server
func (s *Server) Register(rcvr interface{}) error {
	return s.register(rcvr, "", false)
}

// Register publishes in the server the set of methods of the
// receiver value
func (s *Server) register(rcvr interface{}, name string, useName bool) error {
	s.serviceMapMu.Lock()
	defer s.serviceMapMu.Unlock()

	service := new(service)
	service.typ = reflect.TypeOf(rcvr)
	service.rcvr = reflect.ValueOf(rcvr)

	//check and set service.name
	sname := reflect.Indirect(service.rcvr).Type().Name()
	if useName {
		sname = name
	}
	if sname == "" {
		s := "rpc.Register: no service name for type " + service.typ.String()
		log.Print(s)
		return errors.New(s)
	}
	if !isExported(sname) && !useName {
		s := "rpc.Register: type " + sname + " is not exported"
		log.Print(s)
		return errors.New(s)
	}
	service.name = sname

	// install the methods
	service.method = s.suitableMethods(service.typ)

	if len(service.method) == 0 {
		str := ""
		// To help the user, see if a pointer receiver would work.
		method := s.suitableMethods(reflect.PtrTo(service.typ))
		if len(method) != 0 {
			str = "rpc.Register: type " + sname + " has no exported methods of suitable type (hint: pass a pointer to value of that type)"
		} else {
			str = "rpc.Register: type " + sname + " has no exported methods of suitable type"
		}
		log.Print(str)
		return errors.New(str)
	}

	s.serviceMap[service.name] = service
	return nil
}

// suitableMethods returns suitable Rpc methods of typ
func (s *Server) suitableMethods(typ reflect.Type) map[string]*methodType {
	methods := make(map[string]*methodType)
	for m := 0; m < typ.NumMethod(); m++ {
		method := typ.Method(m)
		if method.PkgPath != "" {
			// skip unexported method
			continue
		}
		mtype := method.Type

		if !mtype.Out(mtype.NumOut() - 1).Implements(typeOfError) {
			// last return type must be error
			log.Printf("rpc.Register: last return type of method %q is %q, must be error\n",
				method.Name, mtype.Out(m))
			continue
		}

		//TODO
		//need check Argtypes and ReplyTypes

		// input arguments of the method
		argTypes := make([]reflect.Type, 0, mtype.NumIn())
		// first param is method receiver
		for i := 1; i < mtype.NumIn(); i++ {
			argTypes = append(argTypes, mtype.In(i))
		}

		// return arguments of the method
		replyTypes := make([]reflect.Type, 0, mtype.NumOut())
		for i := 0; i < mtype.NumOut(); i++ {
			replyTypes = append(replyTypes, mtype.Out(i))
		}

		// add the method to the set
		methods[method.Name] = &methodType{
			method:     method,
			ArgTypes:   argTypes,
			ReplyTypes: replyTypes,
		}
	}
	return methods
}

func isExported(name string) bool {
	r, _ := utf8.DecodeRuneInString(name)
	return unicode.IsUpper(r)
}

// Serve accepts connections on the listener and serves requests
// for each incoming connection.
func (s *Server) Serve(lis net.Listener) {
	for {
		conn, err := lis.Accept()
		if err != nil {
			log.Print("ferry.Serve: accpet:", err.Error())
			return
		}
		go s.ServeConn(conn)
	}
}

// ServeConn reads message from conn then handle the message.
func (s *Server) ServeConn(conn net.Conn) {

	r := bufio.NewReaderSize(conn, ReadSize)

	for {

		// read the message
		msg, err := message.RecvMessage(r)
		if err != nil {
			if err == io.EOF {
				log.Print("ferry.ServeConn: RecvMessage EOF: ", time.Now())
				return
			} else {
				log.Print("ferry.ServeConn: RecvMessage Fail: ", err.Error())
				return
			}
		}
		if msg.MessageType != message.MsgTypeRequest {
			// not request message
			log.Print("ferry.ServeConn: Invalid Message: ", err.Error())
			return
		}

		// get request body
		req, err := msg.DecodeRequest()
		if err != nil {
			log.Print("ferry.ServeConn: DecodeRequest: ", err.Error())
			return
		}

		// handle the request
		go func() {
			replys, err := s.handleRequest(req)
			if err != nil {
				log.Print("ferry.ServeConn: handleRequest: ", err.Error())
				return
			}
			resp := message.Response{
				Result: replys,
			}
			respData, err := json.Marshal(resp)
			if err != nil {
				log.Print("ferry.ServeConn: Marshal: ", err.Error())
				return
			}

			// wrap the response message
			respMsg := message.Message{
				Header: &message.Header{
					Version:      msg.Version,
					MessageType:  message.MsgTypeResponse,
					CompressType: msg.CompressType,
					SeqID:        msg.SeqID,
					BodyLength:   uint32(len(respData)),
				},
				Data: respData,
			}
			// send the message to client
			_, err = conn.Write(respMsg.Encode())
			if err != nil {
				log.Print("ferry.ServeConn: Write: ", err.Error())
				return
			}
		}()
	}
}

// handleRequest finds the corresponding method,
// then executes the method with arguments in the message.
func (s *Server) handleRequest(req *message.RawRequest) ([]interface{}, error) {
	serviceName := req.Path
	serviceMethod := req.Method

	// find the service
	s.serviceMapMu.RLock()
	service := s.serviceMap[serviceName]
	s.serviceMapMu.RUnlock()
	if service == nil {
		return nil, fmt.Errorf("ferry.handleRequest can't find service: %s", serviceName)
	}

	// find the method
	m := service.method[serviceMethod]
	if m == nil {
		return nil, fmt.Errorf("ferry.handleRequest can't find method: %s", serviceMethod)
	}

	//args count must equal
	if len(req.Args) != len(m.ArgTypes) {
		return nil, fmt.Errorf("ferry.handleRequest method args count unequal, demand %d have %d",
			len(m.ArgTypes), len(req.Args))
	}

	function := m.method.Func
	// wrap the input arguments
	in := make([]reflect.Value, 0, len(req.Args)+1)
	in = append(in, service.rcvr)
	for i, arg := range req.Args {
		inst := reflect.New(m.ArgTypes[i])
		err := json.Unmarshal(arg, inst.Interface())
		if err != nil {
			return nil, fmt.Errorf("ferry.handleRequest args %d Unmarshal Fail:%v", i, err)
		}
		in = append(in, inst.Elem())
	}

	// call the method
	replys := function.Call(in)
	result := make([]interface{}, 0, len(replys))
	for _, r := range replys {
		result = append(result, r.Interface())
	}

	return result, nil
}
