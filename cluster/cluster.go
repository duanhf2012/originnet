package cluster

import (
	"github.com/duanhf2012/originnet/rpc"
)

var configdir = "./config/"

type SubNet struct {
	NodeList []NodeInfo
}

type NodeInfo struct {
	NodeId int
	ListenAddr string
	NodeName string
	ServiceList []string
}

type NodeRpcInfo struct {
	nodeinfo NodeInfo
	client *rpc.Client
}

type Cluster struct {
	localsubnet SubNet         //本子网
	mapSubNetInfo map[string] SubNet //子网名称，子网信息

	mapSubNetNodeInfo map[string]map[int]NodeInfo //map[子网名称]map[NodeId]NodeInfo
	localSubNetMapNode map[int]NodeInfo           //本子网内 map[NodeId]NodeInfo
	localSubNetMapService map[string][]NodeInfo   //本子网内所有ServiceName对应的结点列表
	localNodeMapService map[string]interface{}    //本Node支持的服务

	mapRpc map[int] NodeRpcInfo//nodeid
	rpcServer rpc.Server
}


type RemoteNode struct {
	rpcconn rpc.IRpcConn
}




func SetConfigDir(cfgdir string){
	configdir = cfgdir
}


func CallNode(NodeId int,NodeServiceMethod string, args interface{},reply interface{} ) error {
	/*v,ok := mapRpcConn[NodeId]
	if ok  == false {
		return fmt.Errorf("cannot find nodeid:%d",NodeId)
	}

	return v.rpcconn.Call(NodeServiceMethod,args,reply)

	 */

	return nil
}

func (slf *Cluster) Init(currentNodeId int) error{
	//1.初始化配置
	err := slf.InitCfg(currentNodeId)
	if err != nil {
		return err
	}

	//2.建议rpc连接
	slf.mapRpc = map[int] NodeRpcInfo{}
	for _,nodeinfo := range slf.localSubNetMapNode {
		rpcinfo := NodeRpcInfo{}
		rpcinfo.nodeinfo = nodeinfo
		rpcinfo.client = &rpc.Client{}
		if nodeinfo.NodeId == currentNodeId {
			rpcinfo.client.Connect("localhost")
		}else{
			rpcinfo.client.Connect(nodeinfo.ListenAddr)
		}
		slf.mapRpc[nodeinfo.NodeId] = rpcinfo
	}

	return nil
}

