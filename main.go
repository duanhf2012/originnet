package main

import (
	"fmt"
	"github.com/duanhf2012/originnet/node"
	"github.com/duanhf2012/originnet/service"
	"time"
)

type TestService struct {
	service.Service
}


func init(){
	node.Setup(&TestService{})
}

func (slf *TestService) Test(){
	fmt.Printf("xxxx\n")
	//slf.AfterFunc(time.Second,slf.Test)
}

func (slf *TestService) OnInit() error {
	slf.AfterFunc(time.Second,slf.Test)

	return nil
}


func main(){
	node.Start()
}


