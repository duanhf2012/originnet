package rpc

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
)

const SeqSize = 8

type JsonProcessor struct {
}




func (slf *JsonProcessor) GetSeq(data []byte) uint64{
	if len(data)<8 {
		return 0
	}

	if LittleEndian {
		return binary.LittleEndian.Uint64(data[:SeqSize])
	}

	return binary.BigEndian.Uint64(data[:SeqSize])
}

func (slf *JsonProcessor) Marshal(seq uint64,v interface{}) ([]byte, error){
	bytes,err := json.Marshal(v)
	if err != nil {
		return nil,err
	}

	ret := make([]byte,SeqSize+len(bytes))
	if LittleEndian {
		binary.LittleEndian.PutUint64(ret,SeqSize)
	}else{
		binary.BigEndian.PutUint64(ret,SeqSize)
	}
	ret = append(ret,bytes...)
	return ret,nil
}

func (slf *JsonProcessor) Unmarshal(data []byte, v interface{}) error{
	if len(data) < SeqSize {
		return fmt.Errorf("data size < %d",SeqSize)
	}

	return json.Unmarshal(data[8:],v)
}

