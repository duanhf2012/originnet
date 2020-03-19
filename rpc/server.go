package rpc

import (
	"fmt"
)

type CallInfo struct {
	serviceMethod string
	arg interface{}
	reply interface{}
	err error
	done          chan *CallInfo  // Strobes when call is complete.
	connid int
}

type iprocessor interface {
	Marshal(v interface{}) ([]byte, error)
	Unmarshal(data []byte, v interface{}) error
}

type server struct {
	functions map[interface{}]interface{}
	listenAddr string //ip:port

	cmdchannel chan *CallInfo
	processor iprocessor
}

func (slf *server) Init() {
	slf.cmdchannel = make(chan *CallInfo,10000)
}

func  (slf *server) Register(id interface{}, f interface{}) error {
	switch f.(type) {
	case func(interface{},interface{}) error:
	default:
		//return fmt.Errorf("function id %v: already registered", id)
		panic(fmt.Sprintf("function id %v: already registered", id))
	}

	slf.functions[id] = f

	return nil
}

func (slf *server) Start(listenAddr string) {
	slf.listenAddr = listenAddr
}

