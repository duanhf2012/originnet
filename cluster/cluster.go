package cluster

import (
	"github.com/duanhf2012/originnet/rpc"
)

var mapRpcConn map[int]rpc.IRpcConn

func init(){
	localClient := &rpc.Client{}
	localClient.Connect("localhost:0")
	mapRpcConn[0] = localClient
}

func Call(NodeServiceMethod string, args interface{},reply interface{} ) error {
	return mapRpcConn[0].Call(NodeServiceMethod,args,reply)
}
