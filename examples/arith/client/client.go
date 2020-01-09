package main

import (
	"fmt"
	"github.com/sunlidea/ferry/client"
	"github.com/sunlidea/ferry/examples/arith/server/api"
	"log"
)

func main() {

	c, err := client.Dail("tcp", "127.0.0.1"+":1234",
		"Arith", new(api.ArithProxy))
	if err != nil {
		panic(err)
	}

	arithService := c.GetService().(*api.ArithProxy)
	args := &api.Args{A: 10, B: 5}
	mul, err := arithService.Multiply(args)
	if err != nil {
		panic(fmt.Errorf("arithService.Multiply Fail|%v", err))
	}
	log.Printf("%d * %d = %d\n", args.A, args.B, mul)
}
