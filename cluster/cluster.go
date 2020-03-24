package cluster

import (
	"fmt"
	"github.com/duanhf2012/originnet/log"
	"github.com/duanhf2012/originnet/rpc"
	"github.com/duanhf2012/originnet/service"
	"strings"
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

var cluster Cluster

type Cluster struct {
	localsubnet SubNet         //本子网
	mapSubNetInfo map[string] SubNet //子网名称，子网信息

	mapSubNetNodeInfo map[string]map[int]NodeInfo //map[子网名称]map[NodeId]NodeInfo
	localSubNetMapNode map[int]NodeInfo           //本子网内 map[NodeId]NodeInfo
	localSubNetMapService map[string][]NodeInfo   //本子网内所有ServiceName对应的结点列表
	localNodeMapService map[string]interface{}    //本Node支持的服务
	localNodeInfo NodeInfo

	mapRpc map[int] NodeRpcInfo//nodeid

	rpcServer rpc.Server
}


func SetConfigDir(cfgdir string){
	configdir = cfgdir
}

func (slf *Cluster) Init(currentNodeId int) error{
	//1.初始化配置
	err := slf.InitCfg(currentNodeId)
	if err != nil {
		return err
	}

	slf.rpcServer.Init(slf)

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

func (slf *Cluster) FindRpcHandler(servicename string) rpc.IRpcHandler {
	pService := service.GetService(servicename)
	if pService == nil {
		return nil
	}

	return pService.GetRpcHandler()
}

func (slf *Cluster) Start() {
	slf.rpcServer.Start(slf.localNodeInfo.ListenAddr)
}

func GetCluster() *Cluster{
	return &cluster
}

func (slf *Cluster) GetRpcClient(nodeid int) *rpc.Client {
	c,ok := slf.mapRpc[nodeid]
	if ok == false {
		return nil
	}

	return c.client
}


//servicename.method
func Call(irecv rpc.IRpcHandler,serviceMethod string,reply interface{},args ...interface{}) error {
	pClient,err := GetRpcClient(serviceMethod)
	if err != nil {
		log.Error("Call serviceMethod is error:%+v!",err)
		return err
	}

	//2.rpcclient调用
	pCall := pClient.Go(serviceMethod,reply,args)
	pResult := pCall.Done()
	return pResult.Err
}


func GetRpcClient(serviceMethod string) (*rpc.Client,error) {
	serviceAndMethod := strings.Split(serviceMethod,".")
	if len(serviceAndMethod)!=2 {
		return nil,fmt.Errorf("servicemethod param  %s is error!",serviceMethod)
	}

	//1.找到对应的rpcnodeid
	nodeidList := GetCluster().GetNodeIdByService(serviceAndMethod[0])
	if len(nodeidList) ==0 {
		return nil,fmt.Errorf("Cannot Find %s nodeid",serviceMethod)
	}else if len(nodeidList) >1 {
		return nil,fmt.Errorf("Find %s more than 1 nodeid",serviceMethod)
	}
	pClient := GetCluster().GetRpcClient(nodeidList[0])
	if pClient==nil {
		return nil,fmt.Errorf("Cannot connect node id %d",nodeidList[0])
	}

	return pClient,nil
}