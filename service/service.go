package service

import (
	"github.com/duanhf2012/originnet/rpc"
	"github.com/duanhf2012/originnet/util/timer"
	"reflect"
	"sync"
	"time"
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
	rpc.RpcHandler   //rpc
	dispatcher         *timer.Dispatcher //timer
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


func (slf *Service) AfterFunc(d time.Duration, cb func()) *timer.Timer {
	return slf.dispatcher.AfterFunc(d, cb)
}

func (slf *Service) CronFunc(cronExpr *timer.CronExpr, cb func()) *timer.Cron {
	return slf.dispatcher.CronFunc(cronExpr, cb)
}

func (slf *Service) Wait(){
	slf.wg.Wait()
}
