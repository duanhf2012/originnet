package main

import (
	"github.com/duanhf2012/originnet/node"
	"github.com/duanhf2012/originnet/service"
)

type TestService struct {
	service.Service
}

func init(){
	node.Setup(&TestService{})
}

func main(){
	node.Start()
}


