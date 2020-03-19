package rpc

import "fmt"


type Client struct {
}

func (slf *Client) Call(NodeServiceMethod string, args interface{},replys interface{} ) error {
	return nil
}

func (slf *Client) Connect(addr string) error {
	return nil
}

func check(id interface{}, n int) (f interface{}, err error) {
	var ok bool
	switch n {
	case 0:
		_, ok = f.(func([]interface{}))
	case 1:
		_, ok = f.(func([]interface{}) interface{})
	case 2:
		_, ok = f.(func([]interface{}) []interface{})
	default:
		panic("bug")
	}

	if !ok {
		err = fmt.Errorf("function id %v: return type mismatch", id)
	}
	return
}

