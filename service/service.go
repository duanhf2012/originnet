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
	Init(iservice IService,closeSig chan bool)
	GetName() string

	OnInit() error
	OnRelease()
	Wait()
	Start()
}


type Service struct {
	//
	rpc.RpcHandler
	name string
	closeSig chan bool
	dispatcher         *timer.Dispatcher
	wg      sync.WaitGroup
	this    IService
}

func (slf *Service)GetRpcHandler() rpc.IRpcHandler{
	return slf.this.(rpc.IRpcHandler)
}

func (slf *Service) reflectRpcMethodInfo(iservice IService) {
	/*
	values := reflect.ValueOf(iservice)
	types := reflect.TypeOf(iservice)

	slf.name = reflect.Indirect(values).Type().Name()
	for m := 0; m < types.NumMethod(); m++ {
		method := types.Method(m)
		mtype := method.Type
		mname := method.Name
		if strings.Index(mname,"RPC_")!=0 {
			continue
		}
	}*/
}

func (slf *Service) Init(iservice IService,closeSig chan bool) {
	slf.name = reflect.Indirect(reflect.ValueOf(iservice)).Type().Name()
	slf.closeSig = closeSig
	slf.dispatcher =timer.NewDispatcher(timerDispatcherLen)
	slf.this = iservice
	slf.InitRpcHandler(iservice.(rpc.IRpcHandler))

	//
	//slf.reflectRpcMethodInfo(iservice)


	slf.OnInit()
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
		rpcChannel := slf.GetRpcChannel()
		select {
		case <- slf.closeSig:
			bStop = true
		case callinfo :=<- rpcChannel:
			slf.Handler(callinfo)
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

func Init(chanCloseSig chan bool) {
	closeSig=chanCloseSig
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
