package rpc

import (
	"github.com/duanhf2012/originnet/log"
	"github.com/duanhf2012/originnet/network"
	"math"
	"net"
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
	GetSeq(data []byte) uint64
	Marshal(seq uint64,v interface{}) ([]byte, error)
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
}


func (gate *RpcAgent) OnDestroy() {}

type RpcAgent struct {
	conn     network.Conn
	rpcserver     *Server
	userData interface{}
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
		seq := processor.GetSeq(data)
		err = processor.Unmarshal(data,&req)
		if err != nil {
			log.Debug("processor message: %v", err)
			agent.Close()
			break
		}

		//交给程序处理
		rpcHandler := agent.rpcserver.rpcHandleFinder.FindRpcHandler(req.ServiceMethod)
		if rpcHandler== nil {
			log.Error("service method %s not config!", req.ServiceMethod)
			continue
		}
		req.RequestHandle = func(Returns interface{},Err error){
		var rpcRespone RpcResponse
			rpcRespone.ServiceMethod = req.ServiceMethod
			rpcRespone.Returns = Returns
			rpcRespone.Err = Err
			bytes,err :=  processor.Marshal(seq,Returns)
			if err != nil {
				log.Error("service method %s Marshal error:%+v!", req.ServiceMethod,err)
				return
			}

			agent.conn.WriteMsg(bytes)
		}

	rpcHandler.PushRequest(&req)

	}
	/*
	for {
		data, err := agent.conn.ReadMsg()
		if err != nil {
			log.Debug("read message: %v", err)
			break
		}

		if agent.gate.Processor != nil {
			msg, err := agent.gate.Processor.Unmarshal(data)
			if err != nil {
				log.Debug("unmarshal message error: %v", err)
				break
			}
			err = agent.gate.Processor.Route(msg, a)
			if err != nil {
				log.Debug("route message error: %v", err)
				break
			}
		}
	}*/

}

func (agent *RpcAgent) OnClose() {
	/*
	if agent.gate.AgentChanRPC != nil {
		err := agent.gate.AgentChanRPC.Call0("CloseAgent", agent)
		if err != nil {
			log.Error("chanrpc error: %v", err)
		}
	}

	 */
}

func (agent *RpcAgent) WriteMsg(msg interface{}) {
	/*
	if agent.gate.Processor != nil {
		data, err := agent.gate.Processor.Marshal(msg)
		if err != nil {
			log.Error("marshal message %v error: %v", reflect.TypeOf(msg), err)
			return
		}
		err = agent.conn.WriteMsg(datagent...)
		if err != nil {
			log.Error("write message %v error: %v", reflect.TypeOf(msg), err)
		}
	}*/
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


//Run()
//OnClose()
func (slf *Server) NewAgent(conn *network.TCPConn) network.Agent {
	agent := &RpcAgent{conn: conn, rpcserver: slf}

	return agent
}