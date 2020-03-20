package rpc

import (

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

type Server struct {
	functions map[interface{}]interface{}
	listenAddr string //ip:port

	cmdchannel chan *CallInfo
	processor iprocessor
}

func (slf *Server) Init() {
	slf.cmdchannel = make(chan *CallInfo,10000)
}

func (slf *Server) Start(listenAddr string) {
	slf.listenAddr = listenAddr
}

