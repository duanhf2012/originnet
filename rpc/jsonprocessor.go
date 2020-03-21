package rpc

type RpcCallData struct {
	ServiceMethod string
	InParam interface{}
	OutParam interface{}
	Err error
}

type JsonProcessor struct {
	mapRpcCallData map[string]RpcCallData
}


// must goroutine safe
func (slf *JsonProcessor) Unmarshal(data []byte) (interface{}, error){
	return nil,nil
}

// must goroutine safe
func (slf *JsonProcessor) Marshal(msg interface{}) ([][]byte, error) {
	return nil,nil
}

func (slf *JsonProcessor) Register(ServiceMethod string,method RPCMethodType) {

}
