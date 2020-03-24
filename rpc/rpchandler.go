package rpc

import (
	"fmt"
	"reflect"
	"strings"
	"unicode"
	"unicode/utf8"
)

type RPCMethodType func(arg ...interface{}) (interface{},error)

type RpcMethodInfo struct {
	method reflect.Method
	param []reflect.Type
	returns reflect.Type
	iparam []interface{}
	ireturns interface{}
}

type RpcHandler struct {
	callchannel chan *CallInfo
	rpcHandler IRpcHandler
	mapfunctons map[string]RpcMethodInfo
}

type IRpcHandler interface {
	GetName() string
	InitRpcHandler(rpcHandler IRpcHandler)
	GetRpcHandler() IRpcHandler
}

func (slf *RpcHandler) GetRpcHandler() IRpcHandler{
	return slf.rpcHandler
}

func (slf *RpcHandler) InitRpcHandler(rpcHandler IRpcHandler) {
	slf.callchannel = make(chan *CallInfo,10000)
	slf.rpcHandler = rpcHandler
	slf.mapfunctons = map[string]RpcMethodInfo{}
	slf.RegisterRpc(rpcHandler)
}

// Is this an exported - upper case - name?
func isExported(name string) bool {
	rune, _ := utf8.DecodeRuneInString(name)
	return unicode.IsUpper(rune)
}

// Is this type exported or a builtin?
func (slf *RpcHandler) isExportedOrBuiltinType(t reflect.Type) bool {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	// PkgPath will be non-empty even for an exported type,
	// so we need to check the type name as well.
	return isExported(t.Name()) || t.PkgPath() == ""
}


func (slf *RpcHandler) suitableMethods(method reflect.Method) error {
	//只有RPC_开头的才能被调用
	if strings.Index(method.Name,"RPC_")!=0 {
		return nil
	}

	//取出输入参数类型
	var rpcMethodInfo RpcMethodInfo
	typ := method.Type
	if typ.NumOut() != 2 {
		return fmt.Errorf("%s The number of returned arguments must be 2!",method.Name)
	}

	if typ.Out(1).String() != "error" {
		return fmt.Errorf("%s The return parameter must be of type error!",method.Name)
	}

	for in := 1;in<typ.NumIn();in++{
		if slf.isExportedOrBuiltinType(typ.In(in)) == false {
			return fmt.Errorf("%s Unsupported parameter types!",method.Name)
		}
		rpcMethodInfo.param = append(rpcMethodInfo.param,typ.In(in))
		rpcMethodInfo.iparam = append(rpcMethodInfo.iparam,reflect.New(typ.In(in)))
	}

	rpcMethodInfo.returns = typ.Out(0)
	if slf.isExportedOrBuiltinType(typ.Out(0))== false{
		return fmt.Errorf( "rpc.Register: reply type of method %q is not exported\n", method.Name)
	}
	rpcMethodInfo.ireturns = reflect.New(typ.Out(0))

	slf.mapfunctons[method.Name] = rpcMethodInfo
	return nil
}

func  (slf *RpcHandler) RegisterRpc(rpcHandler IRpcHandler) error {
	typ := reflect.TypeOf(rpcHandler)
	for m:=0;m<typ.NumMethod();m++{
		method := typ.Method(m)
		err := slf.suitableMethods(method)
		if err != nil {
			panic(err)
		}
	}

	return nil
}

func (slf *RpcHandler) Send(callinfo *CallInfo) {
	if callinfo.connid == 0 {
		callinfo.done = make(chan *CallInfo,1)
	}

	slf.callchannel <- callinfo
}

func (slf *RpcHandler) Handler(callinfo *CallInfo) {

}

func (slf *RpcHandler) GetRpcChannel() (chan *CallInfo) {
	return slf.callchannel
}

