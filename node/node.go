package node

import (
	"github.com/duanhf2012/originnet/service"
)

var mapServiceName map[string]service.IService
var closeSig chan bool
type node struct {

}


func Start() {
	closeSig = make(chan bool,1)
}


func Setup(s service.IService) bool {
	_,ok := mapServiceName[s.GetName()]
	if ok == true {
		return false
	}

	s.Init(s,closeSig)
	mapServiceName[s.GetName()] = s

	return true
}

func GetService(servicename string) service.IService {
	s,ok := mapServiceName[servicename]
	if ok == false {
		return nil
	}

	return s
}