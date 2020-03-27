package rpc

import (
	"fmt"
	"github.com/duanhf2012/originnet/log"
	"reflect"
	"strings"
	"unicode"
	"unicode/utf8"
)

type FuncRpcClient func(serviceMethod string) ([]*Client,error)
type FuncRpcServer func() (*Server)
var NilError = reflect.Zero(reflect.TypeOf((*error)(nil)).Elem())

type RpcMethodInfo struct {
	method reflect.Method
	iparam interface{}
	oParam reflect.Value
}

type RpcHandler struct {
	callRequest chan *RpcRequest
	rpcHandler IRpcHandler
	mapfunctons map[string]RpcMethodInfo
	funcRpcClient FuncRpcClient
	funcRpcServer FuncRpcServer

	callResponeCallBack chan *Call //异步返回的回调
}

type IRpcHandler interface {
	GetName() string
	InitRpcHandler(rpcHandler IRpcHandler,getClientFun FuncRpcClient,getServerFun FuncRpcServer)
	GetRpcHandler() IRpcHandler
	PushRequest(callinfo *RpcRequest)
	HandlerRpcRequest(request *RpcRequest)
	HandlerRpcResponeCB(call *Call)

	GetRpcRequestChan() chan *RpcRequest
	GetRpcResponeChan() chan *Call
	CallMethod(ServiceMethod string,param interface{},reply interface{}) error
}

func (slf *RpcHandler) GetRpcHandler() IRpcHandler{
	return slf.rpcHandler
}

func (slf *RpcHandler) InitRpcHandler(rpcHandler IRpcHandler,getClientFun FuncRpcClient,getServerFun FuncRpcServer) {
	slf.callRequest = make(chan *RpcRequest,10000)
	slf.callResponeCallBack = make(chan *Call,10000)

	slf.rpcHandler = rpcHandler
	slf.mapfunctons = map[string]RpcMethodInfo{}
	slf.funcRpcClient = getClientFun
	slf.funcRpcServer = getServerFun

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

	if typ.NumIn() != 3 {
		return fmt.Errorf("%s The number of input arguments must be 1!",method.Name)
	}

	if slf.isExportedOrBuiltinType(typ.In(1)) == false ||   slf.isExportedOrBuiltinType(typ.In(2)) == false {
		return fmt.Errorf("%s Unsupported parameter types!",method.Name)
	}

	rpcMethodInfo.iparam = reflect.New(typ.In(1).Elem()).Interface() //append(rpcMethodInfo.iparam,)
	rpcMethodInfo.oParam = reflect.New(typ.In(2).Elem())

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

func (slf *RpcHandler) GetRpcResponeChan() chan *Call{
	return slf.callResponeCallBack
}

func (slf *RpcHandler) HandlerRpcResponeCB(call *Call){
	if call.Err == nil {
		call.callback.Call([]reflect.Value{reflect.ValueOf(call.Reply),NilError})
	}else{
		call.callback.Call([]reflect.Value{reflect.ValueOf(call.Reply),reflect.ValueOf(call.Err)})
	}

}

func (slf *RpcHandler) HandlerRpcRequest(request *RpcRequest) {
	v,ok := slf.mapfunctons[request.ServiceMethod]
	if ok == false {
		err := fmt.Errorf("RpcHandler %s cannot find %s",slf.rpcHandler.GetName(),request.ServiceMethod)
		log.Error("%s",err.Error())
		if request.requestHandle!=nil {
			request.requestHandle(nil,err)
		}

		return
	}

	var paramList []reflect.Value
	var err error
	if request.localParam==nil{
		err = processor.Unmarshal(request.InParam,&v.iparam)
		if err!=nil {
			rerr := fmt.Errorf("Call Rpc %s Param error %+v",request.ServiceMethod,err)
			log.Error("%s",rerr.Error())
			if request.requestHandle!=nil {
				request.requestHandle(nil, rerr)
			}
		}
	}else {
		v.iparam = request.localParam
	}


	paramList = append(paramList,reflect.ValueOf(slf.GetRpcHandler())) //接受者
	if request.localReply!=nil {
		v.oParam = reflect.ValueOf(request.localReply)
	}
	paramList = append(paramList,v.oParam) //输出参数
	//其他输入参数
	paramList = append(paramList,reflect.ValueOf(v.iparam))


	returnValues := v.method.Func.Call(paramList)
	errInter := returnValues[0].Interface()
	if errInter != nil {
		err = errInter.(error)
	}

	if request.requestHandle!=nil {
		request.requestHandle(v.oParam.Interface(), err)
	}
}

func (slf *RpcHandler) CallMethod(ServiceMethod string,param interface{},reply interface{}) error{
	var err error
	v,ok := slf.mapfunctons[ServiceMethod]
	if ok == false {
		err = fmt.Errorf("RpcHandler %s cannot find %s",slf.rpcHandler.GetName(),ServiceMethod)
		log.Error("%s",err.Error())

		return err
	}

	var paramList []reflect.Value
	paramList = append(paramList,reflect.ValueOf(slf.GetRpcHandler())) //接受者
	paramList = append(paramList,reflect.ValueOf(param))
	paramList = append(paramList,reflect.ValueOf(reply)) //输出参数

	returnValues := v.method.Func.Call(paramList)
	errInter := returnValues[0].Interface()
	if errInter != nil {
		err = errInter.(error)
	}

	return err
}

func (slf *RpcHandler) goRpc(serviceMethod string,mutiCoroutine bool,args interface{}) error {
	pClientList,err := slf.funcRpcClient(serviceMethod)
	if err != nil {
		log.Error("Call serviceMethod is error:%+v!",err)
		return err
	}
	if len(pClientList) > 1 {
		log.Error("Cannot call more then 1 node!")
		return fmt.Errorf("Cannot call more then 1 node!")
	}

	//2.rpcclient调用
	//如果调用本结点服务
	pClient := pClientList[0]
	if pClient.blocalhost == true {
		pLocalRpcServer:=slf.funcRpcServer()
		//判断是否是同一服务
		sMethod := strings.Split(serviceMethod,".")
		if len(sMethod)!=2 {
			err := fmt.Errorf("Call serviceMethod %s is error!",serviceMethod)
			log.Error("%+v",err)
			return err
		}
		//调用自己rpcHandler处理器
		if sMethod[0] == slf.rpcHandler.GetName() { //自己服务调用
			//
			return pLocalRpcServer.myselfRpcHandlerGo(sMethod[0],sMethod[1],args,nil)
		}
		//其他的rpcHandler的处理器
		pCall := pLocalRpcServer.rpcHandlerGo(true,mutiCoroutine,sMethod[0],sMethod[1],args,nil)
		return pCall.Err
	}

	//跨node调用
	pCall := pClient.Go(true,mutiCoroutine,serviceMethod,args,nil)
	return pCall.Err
}

func (slf *RpcHandler) callRpc(serviceMethod string,mutiCoroutine bool,args interface{},reply interface{}) error {
	pClientList,err := slf.funcRpcClient(serviceMethod)
	if err != nil {
		log.Error("Call serviceMethod is error:%+v!",err)
		return err
	}
	if len(pClientList) > 1 {
		log.Error("Cannot call more then 1 node!")
		return fmt.Errorf("Cannot call more then 1 node!")
	}

	//2.rpcclient调用
	//如果调用本结点服务
	pClient := pClientList[0]
	if pClient.blocalhost == true {
		pLocalRpcServer:=slf.funcRpcServer()
		//判断是否是同一服务
		sMethod := strings.Split(serviceMethod,".")
		if len(sMethod)!=2 {
			err := fmt.Errorf("Call serviceMethod %s is error!",serviceMethod)
			log.Error("%+v",err)
			return err
		}
		//调用自己rpcHandler处理器
		if sMethod[0] == slf.rpcHandler.GetName() { //自己服务调用
			//
			return pLocalRpcServer.myselfRpcHandlerGo(sMethod[0],sMethod[1],args,reply)
		}
		//其他的rpcHandler的处理器
		pCall := pLocalRpcServer.rpcHandlerGo(false,mutiCoroutine,sMethod[0],sMethod[1],args,reply)
		pResult := pCall.Done()
		return pResult.Err
	}

	//跨node调用
	pCall := pClient.Go(false,mutiCoroutine,serviceMethod,args,reply)
	pResult := pCall.Done()
	return pResult.Err
}

func (slf *RpcHandler) asyncCallRpc(serviceMethod string,mutiCoroutine bool,args interface{},callback interface{}) error {
	fVal := reflect.ValueOf(callback)
	if fVal.Kind()!=reflect.Func{
		return fmt.Errorf("input function is error!")
	}

	reply := reflect.New(fVal.Type().In(0).Elem()).Interface()
	pClientList,err := slf.funcRpcClient(serviceMethod)
	if err != nil {
		log.Error("Call serviceMethod is error:%+v!",err)
		return err
	}
	if len(pClientList) > 1 {
		log.Error("Cannot call more then 1 node!")
		return fmt.Errorf("Cannot call more then 1 node!")
	}

	//2.rpcclient调用
	//如果调用本结点服务
	pClient := pClientList[0]
	if pClient.blocalhost == true {
		pLocalRpcServer:=slf.funcRpcServer()
		//判断是否是同一服务
		sMethod := strings.Split(serviceMethod,".")
		if len(sMethod)!=2 {
			err := fmt.Errorf("Call serviceMethod %s is error!",serviceMethod)
			log.Error("%+v",err)
			return err
		}
		//调用自己rpcHandler处理器
		if sMethod[0] == slf.rpcHandler.GetName() { //自己服务调用
			err := pLocalRpcServer.myselfRpcHandlerGo(sMethod[0],sMethod[1],args,reply)
			if err == nil {
				fVal.Call([]reflect.Value{reflect.ValueOf(reply),NilError})
			}else{
				fVal.Call([]reflect.Value{reflect.ValueOf(reply),reflect.ValueOf(err)})
			}

		}

		//其他的rpcHandler的处理器
		if callback!=nil {
			return  pLocalRpcServer.rpcHandlerAsyncGo(slf,false,mutiCoroutine,sMethod[0],sMethod[1],args,reply,fVal)
		}
		pCall := pLocalRpcServer.rpcHandlerGo(false,mutiCoroutine,sMethod[0],sMethod[1],args,reply)
		pResult := pCall.Done()
		return pResult.Err
	}

	//跨node调用
	return pClient.AsycGo(slf,mutiCoroutine,serviceMethod,fVal,args,reply)
}

func (slf *RpcHandler) GetName() string{
	return slf.rpcHandler.GetName()
}


//func (slf *RpcHandler) asyncCallRpc(serviceMethod string,mutiCoroutine bool,callback interface{},args ...interface{}) error {
//func (slf *RpcHandler) callRpc(serviceMethod string,reply interface{},mutiCoroutine bool,args ...interface{}) error {
//func (slf *RpcHandler) goRpc(serviceMethod string,mutiCoroutine bool,args ...interface{}) error {

func (slf *RpcHandler) AsyncCall(serviceMethod string,args interface{},callback interface{}) error {
	return slf.asyncCallRpc(serviceMethod,false,args,callback)
}

func (slf *RpcHandler) GRAsyncCall(serviceMethod string,args interface{},callback interface{}) error {
	return slf.asyncCallRpc(serviceMethod,true,args,callback)
}

func (slf *RpcHandler) Call(serviceMethod string,args interface{},reply interface{}) error {
	return slf.callRpc(serviceMethod,false,args,reply)
}

func (slf *RpcHandler) GRCall(serviceMethod string,args interface{},reply interface{}) error {
	return slf.callRpc(serviceMethod,true,args,reply)
}

func (slf *RpcHandler) Go(serviceMethod string,args interface{}) error {
	return slf.goRpc(serviceMethod,false,args)
}

func (slf *RpcHandler) GRGo(serviceMethod string,args interface{}) error {
	return slf.goRpc(serviceMethod,true,args)
}