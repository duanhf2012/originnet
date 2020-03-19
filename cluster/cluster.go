package cluster

import (
	"github.com/duanhf2012/originnet/rpc"
)

var mapRpcConn map[int]rpc.IRpcConn


func Call(NodeServiceMethod string, args interface{},replys interface{} ) error {



	return ri.err
}
