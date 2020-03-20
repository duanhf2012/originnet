package rpc

import (
	"fmt"
	"net"
	"strings"
)

type IRpcConn interface {
	ReConnect() error
	Connect(addr string) error
	Call(NodeServiceMethod string, args interface{},replys interface{} ) error
}

type Client struct {
	blocalhost bool
	conn net.Conn
	saddr string
}

func (slf *Client) Call(NodeServiceMethod string, args interface{},replys interface{} ) error {
	if slf.blocalhost == true {
		//node.GetService("")
	}
	return nil
}

func (slf *Client) ReConnect() error {
	if slf.blocalhost== true || slf.conn!=nil {
		return nil
	}

	return slf.Connect(slf.saddr)
}

func (slf *Client) Connect(addr string) error {
	slf.saddr = addr
	if addr == "" || strings.Index(addr,"localhost")!=-1 {
		slf.blocalhost = true
		return nil
	}

	tcpAddr,err := net.ResolveTCPAddr("tcp",addr)
	if err != nil {
		return err
	}

	slf.conn,err = net.DialTCP("tcp",nil,tcpAddr)
	if err!=nil {
		fmt.Println("Client connect error ! " + err.Error())
		return err
	}

	return nil
}

