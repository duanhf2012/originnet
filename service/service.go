package service

import (
	"github.com/duanhf2012/originnet/rpc"
	"fmt"
)


var closeSig chan bool

type IService interface {
	Init(iservice IService,closeSig chan bool)
	GetName() string
}

type Service struct {
	//
	rpc.RpcHandler
	name string
	closeSig chan bool
}

func (slf *Service) Init(iservice IService,closeSig chan bool) {
	slf.name = fmt.Sprintf("%T",iservice)
	slf.closeSig = closeSig
}


func (slf *Service) Start() {
	go func(){
		slf.Run()
	}()
}

func (slf *Service) Run() {
	rpcChannel := slf.GetRpcChannel()
	select {
		case <- slf.closeSig:
			return
		case callinfo :=<- rpcChannel:
			slf.Handler(callinfo)
	}
}

func (slf *Service) GetName() string{
	return slf.name
}


func Init(chanCloseSig chan bool) {
	closeSig=chanCloseSig
}


