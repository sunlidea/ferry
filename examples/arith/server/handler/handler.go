package handler

import (
	"errors"
	"github.com/sunlidea/ferry/examples/arith/server/api"
)

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
