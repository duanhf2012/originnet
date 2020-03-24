package rpc

import (
	"github.com/duanhf2012/originnet/network"
	"math"
	"strings"
	"sync"
	"time"
)

type Client struct {
	blocalhost bool
	network.TCPClient
	conn *network.TCPConn

	//
	pendingLock sync.RWMutex
	startSeq uint64
	pending map[uint64]*Call
}

func (slf *Client) NewClientAgent(conn *network.TCPConn) network.Agent {
	slf.conn = conn
	return slf
}

func (slf *Client) Connect(addr string) error {
	slf.Addr = addr

	if addr == "" || strings.Index(addr,"localhost")!=-1 {
		slf.blocalhost = true
		return nil
	}

	slf.ConnNum = 1
	slf.ConnectInterval = time.Second*2
	slf.PendingWriteNum = 10000
	slf.AutoReconnect = true
	slf.LenMsgLen =2
	slf.MinMsgLen = 2
	slf.MaxMsgLen = math.MaxUint16
	slf.NewAgent =slf.NewClientAgent
	slf.LittleEndian = LittleEndian
	slf.Start()
	return nil
}

func (slf *Client) Go(serviceMethod string,reply interface{}, args ...interface{}) *Call {
	call := new(Call)
	call.done = make(chan *Call,1)
	call.Reply = reply

	request := &RpcRequest{}
	//call.ServiceMethod = serviceMethod
	call.Arg = args
	slf.pendingLock.Lock()
	slf.startSeq+=1
	call.Seq = slf.startSeq
	//request.Seq = slf.startSeq
	slf.pending[call.Seq] = call
	slf.pendingLock.Unlock()

	request.ServiceMethod = serviceMethod
	request.InputParam = args

	bytes,err := processor.Marshal(call.Seq,request)
	if err != nil {
		call.Err = err
		return call
	}
	err = slf.conn.WriteMsg(bytes)
	if err != nil {
		call.Err = err
	}

	return call
}

type RequestHandler func(Returns interface{},Err error)
type RpcRequest struct {
	//
	ServiceMethod string   // format: "Service.Method"
	//Seq           uint64   // sequence number chosen by client
	InputParam []interface{}

	RequestHandle RequestHandler
}

type RpcResponse struct {
	ServiceMethod string   // format: "Service.Method"
	Seq           uint64   // sequence number chosen by client
	Returns interface{}
	Err error
}

func (slf *Client) Run(){
	for {
		bytes,err := slf.conn.ReadMsg()
		if err != nil {
			slf.Close()
			slf.Start()
		}

		seq := processor.GetSeq(bytes)
		slf.pendingLock.Lock()
		defer  slf.pendingLock.Unlock()

		v,ok := slf.pending[seq]
		if ok == true {
			err = processor.Unmarshal(bytes,v.Respone)
			if err != nil {
				//error
				break
			}

			//发送至接受者
			delete(slf.pending,seq)
			v.done <- v
		}


	}

}

func (slf *Client) OnClose(){
	//关闭时，重新连接
	slf.Start()
}