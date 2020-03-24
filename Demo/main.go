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

func (slf *TestService) RPC_Test(a *int,b *int) (*string,error) {
	fmt.Printf("xxxx\n")
	//slf.AfterFunc(time.Second,slf.Test)
	return nil,nil
}

func (slf *TestService) OnInit() error {

	return nil
}



func main(){
	node.Init()
	node.Start()
}


