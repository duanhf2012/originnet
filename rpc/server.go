package rpc

import (
	"github.com/duanhf2012/originnet/log"
	"github.com/duanhf2012/originnet/network"
	"math"
	"net"
	"strings"
)

var processor iprocessor = &JsonProcessor{}
var LittleEndian bool

type Call struct {
	Seq uint64
	//ServiceMethod string
	Arg []interface{}
	Reply interface{}
	Respone *RpcResponse
	Err error
	done          chan *Call  // Strobes when call is complete.
	connid int
}

func (slf *Call) Done() *Call{
	return <-slf.done
}

type iprocessor interface {
	Marshal(v interface{}) ([]byte, error)
	Unmarshal(data []byte, v interface{}) error
}

type Server struct {
	functions map[interface{}]interface{}
	listenAddr string //ip:port

	cmdchannel chan *Call

	rpcHandleFinder RpcHandleFinder
	rpcserver *network.TCPServer
}

type RpcHandleFinder interface {
	FindRpcHandler(serviceMethod string) IRpcHandler
}

func (slf *Server) Init(rpcHandleFinder RpcHandleFinder) {
	slf.cmdchannel = make(chan *Call,10000)
	slf.rpcHandleFinder = rpcHandleFinder
	slf.rpcserver = &network.TCPServer{}
}

func (slf *Server) Start(listenAddr string) {
	slf.listenAddr = listenAddr
	slf.rpcserver.Addr = listenAddr
	slf.rpcserver.LenMsgLen = 2 //uint16
	slf.rpcserver.MinMsgLen = 2
	slf.rpcserver.MaxMsgLen = math.MaxUint16
	slf.rpcserver.MaxConnNum = 10000
	slf.rpcserver.PendingWriteNum = 10000
	slf.rpcserver.NewAgent =slf.NewAgent
	slf.rpcserver.LittleEndian = LittleEndian
	slf.rpcserver.Start()
}


func (gate *RpcAgent) OnDestroy() {}

type RpcAgent struct {
	conn     network.Conn
	rpcserver     *Server
	userData interface{}
}

type RpcRequestRw struct {
	//
	ServiceMethod string   // format: "Service.Method"
	//Seq           uint64   // sequence number chosen by client
	InputParam []byte

	requestHandle RequestHandler
}

func (agent *RpcAgent) Run() {
	for {
		data,err := agent.conn.ReadMsg()
		if err != nil {
			log.Debug("read message: %v", err)
			break
		}

		if processor==nil{
			log.Error("Rpc Processor not set!")
			continue
		}

		var req RpcRequest
		//解析head
		err = processor.Unmarshal(data,&req)
		if err != nil {
			log.Debug("processor message: %v", err)
			agent.Close()
			break
		}

		//交给程序处理
		serviceMethod := strings.Split(req.ServiceMethod,".")
		if len(serviceMethod)!=2 {
			log.Debug("rpc request req.ServiceMethod is error")
			continue
		}
		rpcHandler := agent.rpcserver.rpcHandleFinder.FindRpcHandler(serviceMethod[0])
		if rpcHandler== nil {
			log.Error("service method %s not config!", req.ServiceMethod)
			continue
		}
		req.requestHandle = func(Returns interface{},Err error){
		var rpcRespone RpcResponse
			rpcRespone.Seq = req.Seq
			rpcRespone.Err = Err
			if Err==nil {
				rpcRespone.Returns,rpcRespone.Err = processor.Marshal(Returns)
			}

			bytes,err :=  processor.Marshal(rpcRespone)
			if err != nil {
				log.Error("service method %s Marshal error:%+v!", req.ServiceMethod,err)
				return
			}

			agent.conn.WriteMsg(bytes)
		}

	rpcHandler.PushRequest(&req)

	}
}

func (agent *RpcAgent) OnClose() {
}

func (agent *RpcAgent) WriteMsg(msg interface{}) {
}

func (agent *RpcAgent) LocalAddr() net.Addr {
	return agent.conn.LocalAddr()
}

func (agent *RpcAgent) RemoteAddr() net.Addr {
	return agent.conn.RemoteAddr()
}

func (agent *RpcAgent)  Close() {
	agent.conn.Close()
}

func (agent *RpcAgent) Destroy() {
	agent.conn.Destroy()
}


func (slf *Server) NewAgent(conn *network.TCPConn) network.Agent {
	agent := &RpcAgent{conn: conn, rpcserver: slf}

	return agent
}