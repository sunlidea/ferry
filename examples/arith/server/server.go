package main

import (
	"github.com/sunlidea/ferry/examples/arith/server/handler"
	"github.com/sunlidea/ferry/server"
	"log"
	"net"
)

func main() {
	s := server.NewServer()
	err := s.Register(new(handler.Arith))
	if err != nil {
		panic(err)
	}

	l, err := net.Listen("tcp", ":1234")
	if err != nil {
		panic(err)
	}
	log.Printf("start the ferry server\n")
	s.Serve(l)
}
