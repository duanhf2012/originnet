package rpc

import "fmt"

type RpcHandler struct {
	name string
	functions map[interface{}]interface{}
	callchannel chan *CallInfo
}

type IRpcHandler interface {
	GetName() string
	InitRpcHandler(rpcHandler IRpcHandler)
}

func (slf *RpcHandler) InitRpcHandler(rpcHandler IRpcHandler) {
	slf.name = fmt.Sprintf("%T",rpcHandler)
	slf.callchannel = make(chan *CallInfo,10000)
}

func (slf *RpcHandler) GetName() string {
	return slf.name
}

func  (slf *RpcHandler) RegisterRpc(id interface{}, f interface{}) error {
	switch f.(type) {
	case func(interface{},interface{}) error:
	default:
		//return fmt.Errorf("function id %v: already registered", id)
		panic(fmt.Sprintf("function id %v: already registered", id))
	}

	slf.functions[id] = f
	return nil
}

func (slf *RpcHandler) Send(callinfo *CallInfo) {
	if callinfo.connid == 0 {
		callinfo.done = make(chan *CallInfo,1)
	}

	slf.callchannel <- callinfo
}

func (slf *RpcHandler) Handler(callinfo *CallInfo) {
	v,ok := slf.functions[callinfo.serviceMethod]
	if ok == false{
		callinfo.reply = nil
		callinfo.err = fmt.Errorf("unregister rpc %d method.",callinfo.serviceMethod)
		return
	}

	pFun := v.(func(interface{},interface{}) error)
	callinfo.err = pFun(callinfo.arg,callinfo.reply)
}

func (slf *RpcHandler) GetRpcChannel() (chan *CallInfo) {
	return slf.callchannel
}

