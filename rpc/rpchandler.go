package rpc

import (
	"fmt"

	"reflect"
	"runtime"
	"strings"
)

type RPCMethodType func(arg ...interface{}) (interface{},error)

type RpcHandler struct {
	functions map[interface{}]RPCMethodType
	callchannel chan *CallInfo
	rpcHandler IRpcHandler
}

type IRpcHandler interface {
	GetName() string
	InitRpcHandler(rpcHandler IRpcHandler)
}

func (slf *RpcHandler) InitRpcHandler(rpcHandler IRpcHandler) {
	slf.callchannel = make(chan *CallInfo,10000)
	slf.rpcHandler = rpcHandler
	slf.functions = make(map[interface{}]RPCMethodType,1)
}



func getFunctionName(i interface{}) string {
	name := runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
	startx := strings.LastIndex(name,".")
	if startx == -1{
		return name
	}
	endx := strings.LastIndex(name,"-")
	if endx == -1 {
		endx = len(name)
	}
	name = string([]byte(name)[startx+1:endx])

	return name
}


func  (slf *RpcHandler) RegisterRpc(f RPCMethodType,inParam interface{},outParam interface{}) error {
	name := getFunctionName(f)

	if strings.Index(name,"RPC_")!=0 {
		panic("register error rpc method")
	}
	id := slf.rpcHandler.GetName()+"."+name

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

	callinfo.reply,callinfo.err = v(callinfo.arg)
}

func (slf *RpcHandler) GetRpcChannel() (chan *CallInfo) {
	return slf.callchannel
}

