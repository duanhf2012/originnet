package service

import (
	"github.com/duanhf2012/originnet/rpc"
	"github.com/duanhf2012/originnet/util/timer"
	"reflect"
	"sync"
)


var closeSig chan bool
var timerDispatcherLen = 10

type IService interface {
	Init(iservice IService,getClientFun rpc.FuncRpcClient,getServerFun rpc.FuncRpcServer)
	GetName() string

	OnInit() error
	OnRelease()
	Wait()
	Start()
	GetRpcHandler() rpc.IRpcHandler
}


type Service struct {
	Module

	rpc.RpcHandler   //rpc
	name string    //service name
	closeSig chan bool
	wg      sync.WaitGroup
	this    IService
}

func (slf *Service) Init(iservice IService,getClientFun rpc.FuncRpcClient,getServerFun rpc.FuncRpcServer) {
	slf.name = reflect.Indirect(reflect.ValueOf(iservice)).Type().Name()
	slf.dispatcher =timer.NewDispatcher(timerDispatcherLen)
	slf.this = iservice
	slf.InitRpcHandler(iservice.(rpc.IRpcHandler),getClientFun,getServerFun)

	//初始化祖先
	slf.ancestor = iservice.(IModule)
	slf.seedModuleId =InitModuleId
	slf.descendants = map[int64]IModule{}

	slf.this.OnInit()
}


func (slf *Service) Start() {
	slf.wg.Add(1)
	go func(){
		slf.Run()
	}()
}

func (slf *Service) Run() {
	defer slf.wg.Done()
	var bStop = false
	for{
		rpcRequestChan := slf.GetRpcRequestChan()
		select {
		case <- closeSig:
			bStop = true
		case rpcRequest :=<- rpcRequestChan:
			slf.GetRpcHandler().HandlerRpcRequest(rpcRequest)
		case t := <- slf.dispatcher.ChanTimer:
			t.Cb()
		}

		if bStop == true {
			slf.OnRelease()
			break
		}
	}

}

func (slf *Service) GetName() string{
	return slf.name
}


func (slf *Service) OnRelease(){
}

func (slf *Service) OnInit() error {
	return nil
}


func (slf *Service) Wait(){
	slf.wg.Wait()
}
