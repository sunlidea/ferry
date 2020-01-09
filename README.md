# ferry

Ferry is a RPC framework implemented by Go.

Using reflex, ferry makes remote calls as convenient as local calls.

## Installation

To install this package, you need to install Go and setup your Go workspace on your computer. 
The simplest way to install the library is to run:

```shell

go get -u github.com/sunlidea/ferry

```

## Examples

There is an example in ferry/examples/arith

### server

handler.go: define the handler of server

```go

type Arith int

func (t *Arith) Multiply(args *api.Args) (int, error) {
	return args.A * args.B, nil
}

func (t *Arith) Divide(args *api.Args) (*api.Quotient, error) {

	if args.B == 0 {
		return nil, errors.New("divide by zero")
	}

	quo := &api.Quotient{}
	quo.Quo = args.A / args.B
	quo.Rem = args.A % args.B
	return quo, nil
}

```

server.go: register service and start listening.

```go

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

```

api.go: define the struct that contains all methods in the handler

```go

type ArithProxy struct {
	Multiply func(args *Args) (int, error)
	Divide   func(args *Args) (*Quotient, error)
}

```

### client 

register the arith service

```go

	c, err := client.Dail("tcp", "127.0.0.1"+":1234",
		"Arith", new(api.ArithProxy))
	if err != nil {
		panic(err)
	}

	arithService := c.GetService().(*api.ArithProxy)
	args := &api.Args{A: 10, B:5}
	mul, err := arithService.Multiply(args)
	if err != nil {
		panic(fmt.Errorf("arithService.Multiply Fail|%v", err))
	}
	log.Printf("%d * %d = %d\n", args.A, args.B, mul)

```