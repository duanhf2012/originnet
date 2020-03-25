package main

import (
	"fmt"
	"github.com/duanhf2012/originnet/node"
	"github.com/duanhf2012/originnet/service"
	"time"
)

type TestService1 struct {
	service.Service
}

type TestService2 struct {
	service.Service
}

type TestServiceCall struct {
	service.Service
}

func init(){
	node.Setup(&TestService1{},&TestService2{},&TestServiceCall{})
}

func (slf *TestServiceCall) OnInit() error {
	slf.AfterFunc(time.Second*1,slf.Run)
	return nil
}


func  (slf *TestServiceCall) Run(){
	var ret int
	var input int = 100000

	slf.Call("TestService2.RPC_Test",&ret,&input)
}

func (slf *TestService1) RPC_Test(a *int,b *int) error {
	fmt.Printf("TestService1\n")
	*a = *b*1
	//slf.AfterFunc(time.Second,slf.Test)
	return nil
}

func (slf *TestService1) OnInit() error {
	return nil
}

func (slf *TestService2) RPC_Test(a *int,b *int) error {
	fmt.Printf("TestService2\n")
	*a = *b
	return nil
}

func (slf *TestService2) OnInit() error {
	return nil
}





func main(){
	node.Init()
	node.Start()
}


