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

type Module1 struct{
	service.Module
}

type Module2 struct{
	service.Module
}

type Module3 struct{
	service.Module
}

type Module4 struct{
	service.Module
}
var moduleid1 int64
var moduleid2 int64
var moduleid3 int64
var moduleid4 int64

func (slf *Module1) OnInit() error {
	fmt.Printf("I'm Module1:%d\n",slf.GetModuleId())
	return nil
}

func (slf *Module2) OnInit() error {
	fmt.Printf("I'm Module2:%d\n",slf.GetModuleId())
	moduleid3,_ = slf.AddModule(&Module3{})
	return nil
}
func (slf *Module3) OnInit() error {
	fmt.Printf("I'm Module3:%d\n",slf.GetModuleId())
	moduleid4,_ = slf.AddModule(&Module4{})

	return nil
}

func (slf *Module4) OnInit() error {
	fmt.Printf("I'm Module4:%d\n",slf.GetModuleId())

	return nil
}

func (slf *Module1) OnRelease() {
	fmt.Printf("Release Module1:%d\n",slf.GetModuleId())
}
func (slf *Module2) OnRelease() {
	fmt.Printf("Release Module2:%d\n",slf.GetModuleId())
}
func (slf *Module3) OnRelease() {
	fmt.Printf("Release Module3:%d\n",slf.GetModuleId())
}
func (slf *Module4) OnRelease() {
	fmt.Printf("Release Module4:%d\n",slf.GetModuleId())
}

func (slf *TestServiceCall) OnInit() error {
	//slf.AfterFunc(time.Second*1,slf.Run)
	moduleid1,_ = slf.AddModule(&Module1{})
	moduleid2,_ = slf.AddModule(&Module2{})
	fmt.Print(moduleid1,moduleid2)


	slf.AfterFunc(time.Second*5,slf.Release)
	return nil
}

func  (slf *TestServiceCall) Release(){
	slf.ReleaseModule(moduleid1)
	slf.ReleaseModule(moduleid2)
}

func  (slf *TestServiceCall) Run(){
	var ret int
	var input int = 100000
	bT := time.Now()            // 开始时间

	err := slf.Call("TestServiceCall.RPC_Test",&ret,&input)
	eT := time.Since(bT)      // 从开始到当前所消耗的时间
	fmt.Print(err,eT.Nanoseconds())
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

func (slf *TestServiceCall) RPC_Test(a *int,b *int) error {
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


