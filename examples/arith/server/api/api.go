package api

type ArithProxy struct {
	Multiply func(args *Args) (int, error)
	Divide   func(args *Args) (*Quotient, error)
}

type Args struct {
	A, B int
}

type Quotient struct {
	Quo, Rem int
}
