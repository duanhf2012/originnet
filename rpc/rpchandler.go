package rpc

import (
	"fmt"
	"github.com/duanhf2012/originnet/log"
	"reflect"
	"strings"
	"unicode"
	"unicode/utf8"
)

type RPCMethodType func(arg ...interface{}) (interface{},error)

type RpcMethodInfo struct {
	method reflect.Method
	tparam []reflect.Type
	//returns reflect.Type

	oParam reflect.Value
	iparam [] interface{}
	ireturns interface{}
}

type RpcHandler struct {
	callRequest chan *RpcRequest
	//callchannel chan *Call //待处理队列
	rpcHandler IRpcHandler
	mapfunctons map[string]RpcMethodInfo
	funcRpcClient FuncRpcClient
}

type IRpcHandler interface {
	GetName() string
	InitRpcHandler(rpcHandler IRpcHandler,fun FuncRpcClient)
	GetRpcHandler() IRpcHandler
	PushRequest(callinfo *RpcRequest)
	HandlerRpcRequest(request *RpcRequest)
}

func (slf *RpcHandler) GetRpcHandler() IRpcHandler{
	return slf.rpcHandler
}

type FuncRpcClient func(serviceMethod string) (*Client,error)

func (slf *RpcHandler) InitRpcHandler(rpcHandler IRpcHandler,fun FuncRpcClient) {
	slf.callRequest = make(chan *RpcRequest,10000)

	slf.rpcHandler = rpcHandler
	slf.mapfunctons = map[string]RpcMethodInfo{}
	slf.funcRpcClient = fun

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
	if typ.NumOut() != 1 {
		return fmt.Errorf("%s The number of returned arguments must be 1!",method.Name)
	}

	if typ.Out(0).String() != "error" {
		return fmt.Errorf("%s The return parameter must be of type error!",method.Name)
	}

	for i := 1;i<typ.NumIn();i++{
		if slf.isExportedOrBuiltinType(typ.In(i)) == false {
			return fmt.Errorf("%s Unsupported parameter types!",method.Name)
		}

		//第一个参数为返回参数
		if i == 1 {
			rpcMethodInfo.oParam = reflect.New(typ.In(i).Elem())
		}else{
			rpcMethodInfo.tparam = append(rpcMethodInfo.tparam,typ.In(i))
			rpcMethodInfo.iparam = append(rpcMethodInfo.iparam,reflect.New(typ.In(i).Elem()).Interface())
		}
	}
/*
	//rpcMethodInfo.returns = typ.Out(0)
	if slf.isExportedOrBuiltinType(typ.Out(0))== false{
		return fmt.Errorf( "rpc.Register: reply type of method %q is not exported\n", method.Name)
	}*/
	//rpcMethodInfo.ireturns = reflect.New(typ.Out(0))
	rpcMethodInfo.method = method
	slf.mapfunctons[slf.rpcHandler.GetName()+"."+method.Name] = rpcMethodInfo
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

func (slf *RpcHandler) PushRequest(req *RpcRequest) {
	slf.callRequest <- req
}



func (slf *RpcHandler) GetRpcRequestChan() (chan *RpcRequest) {
	return slf.callRequest
}

func (slf *RpcHandler) Call(serviceMethod string,reply interface{},args ...interface{}) error {
	pClient,err := slf.funcRpcClient(serviceMethod)
	if err != nil {
		log.Error("Call serviceMethod is error:%+v!",err)
		return err
	}

	//2.rpcclient调用
	pCall := pClient.Go(serviceMethod,reply,args...)
	pResult := pCall.Done()
	return pResult.Err
}


func (slf *RpcHandler) HandlerRpcRequest(request *RpcRequest) {
	v,ok := slf.mapfunctons[request.ServiceMethod]
	if ok == false {
		err := fmt.Errorf("RpcHandler %s cannot find %s",slf.rpcHandler.GetName(),request.ServiceMethod)
		log.Error("%s",err.Error())
		request.requestHandle(nil,err)
		return
	}

	var paramList []reflect.Value
	//var test []*string
	err := processor.Unmarshal(request.InParam,&v.iparam)
	if err!=nil {
		err := fmt.Errorf("Call Rpc %s Param error %+v",request.ServiceMethod,err)
		log.Error("%s",err.Error())
		request.requestHandle(nil,err)
	}

	paramList = append(paramList,reflect.ValueOf(slf.GetRpcHandler())) //接受者
	paramList = append(paramList,v.oParam) //输出参数

	//其他输入参数
	for _,iv := range v.iparam {
		paramList = append(paramList,reflect.ValueOf(iv))
	}

	//paramList = append(paramList,reflect.ValueOf(in))
	returnValues := v.method.Func.Call(paramList)
	errInter := returnValues[0].Interface()
	//var err error
	if errInter != nil {
		err = errInter.(error)
	}

	request.requestHandle(v.oParam.Interface(),err)
}
