package rpc

import "fmt"

var functions map[interface{}]interface{}
type IRpcConn interface {
	Call(NodeServiceMethod string, args interface{},replys interface{} ) error
}


type CallInfo struct {
	serviceMethod string
	arg interface{}
	reply interface{}
	err error
	done          chan *CallInfo  // Strobes when call is complete.
}

func  Register(id interface{}, f interface{}) error {
	switch f.(type) {
	case func(interface{},interface{}) error:
	default:
		//return fmt.Errorf("function id %v: already registered", id)
		panic(fmt.Sprintf("function id %v: already registered", id))
	}

	functions[id] = f

	return nil
}