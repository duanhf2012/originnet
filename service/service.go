package service

import (
	"github.com/duanhf2012/originnet/rpc"
	"github.com/duanhf2012/originnet/util/timer"
	"reflect"
	"sync"
	"sync/atomic"
)


var closeSig chan bool
var timerDispatcherLen = 10

type IService interface {
	Init(iservice IService,getClientFun rpc.FuncRpcClient,getServerFun rpc.FuncRpcServer,serviceCfg interface{})
	GetName() string

	OnInit() error
	OnRelease()
	Wait()
	Start()
	GetRpcHandler() rpc.IRpcHandler
	GetServiceCfg()interface{}
}



type Service struct {
	Module
	rpc.RpcHandler   //rpc
	name string    //service name
	closeSig chan bool
	wg      sync.WaitGroup
	this    IService
	serviceCfg interface{}
	gorouterNum int32
	startStatus bool
}

func (slf *Service) Init(iservice IService,getClientFun rpc.FuncRpcClient,getServerFun rpc.FuncRpcServer,serviceCfg interface{}) {
	slf.name = reflect.Indirect(reflect.ValueOf(iservice)).Type().Name()
	slf.dispatcher =timer.NewDispatcher(timerDispatcherLen)
	slf.this = iservice
	slf.InitRpcHandler(iservice.(rpc.IRpcHandler),getClientFun,getServerFun)

	//初始化祖先
	slf.ancestor = iservice.(IModule)
	slf.seedModuleId =InitModuleId
	slf.descendants = map[int64]IModule{}
	slf.serviceCfg = serviceCfg
	slf.gorouterNum = 1
	slf.this.OnInit()
}

func (slf *Service) SetGoRouterNum(gorouterNum int32) bool {
	//已经开始状态不允许修改协程数量
	if slf.startStatus == true {
		return false
	}

	slf.gorouterNum = gorouterNum
	return true
}

func (slf *Service) Start() {
	slf.startStatus = true
	for i:=int32(0);i<slf.gorouterNum;i++{
		slf.wg.Add(1)
		go func(){
			slf.Run()
		}()
	}
}

func (slf *Service) Run() {
	defer slf.wg.Done()
	var bStop = false
	for{
		rpcRequestChan := slf.GetRpcRequestChan()
		rpcResponeCallBack := slf.GetRpcResponeChan()
		eventChan := slf.GetEventChan()
		select {
		case <- closeSig:
			bStop = true
		case rpcRequest :=<- rpcRequestChan:
			slf.GetRpcHandler().HandlerRpcRequest(rpcRequest)
		case rpcResponeCB := <- rpcResponeCallBack:
				slf.GetRpcHandler().HandlerRpcResponeCB(rpcResponeCB)
		case event := <- eventChan:
				slf.OnEventHandler(event)
		case t := <- slf.dispatcher.ChanTimer:
			t.Cb()
		}

		if bStop == true {
			if atomic.AddInt32(&slf.gorouterNum,-1)<=0 {
				slf.startStatus = false
				slf.OnRelease()
			}
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

func (slf *Service) GetServiceCfg()interface{}{
	return slf.serviceCfg
}
