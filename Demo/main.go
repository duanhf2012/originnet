package main

import (
	"fmt"
	"github.com/duanhf2012/originnet/node"
	"github.com/duanhf2012/originnet/service"
)

type TestService struct {
	service.Service
}


func init(){
	node.Setup(&TestService{})
}

func (slf *TestService) RPC_Test(a interface{},b interface{}) error{
	fmt.Printf("xxxx\n")
	//slf.AfterFunc(time.Second,slf.Test)
	return nil
}

func (slf *TestService) OnInit() error {
	//slf.AfterFunc(time.Second,slf.Test)
	slf.RegisterRpc(slf.RPC_Test)
	return nil
}



func main(){
	node.Start()
}


