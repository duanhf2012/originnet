package node

import (
	"fmt"
	"github.com/duanhf2012/originnet/service"
	"github.com/duanhf2012/originnet/cluster"
	"io/ioutil"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

var mapServiceName map[string]service.IService
var closeSig chan bool
var sigs chan os.Signal

func init(){
	mapServiceName = make(map[string]service.IService,5)
	closeSig = make(chan bool,1)
}

func init() {
	sigs = make(chan os.Signal, 3)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM,syscall.Signal(10))
}


func  getRunProcessPid() (int,error) {
	f, err := os.OpenFile(os.Args[0]+".pid", os.O_RDONLY, 0600)
	defer f.Close()
	if err!= nil {
		return 0,err
	}

	pidbyte,errs := ioutil.ReadAll(f)
	if errs!=nil {
		return 0,errs
	}

	return strconv.Atoi(string(pidbyte))
}

func writeProcessPid() {
	//pid
	f, err := os.OpenFile(os.Args[0]+".pid", os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0600)
	defer f.Close()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(-1)
	} else {
		_,err=f.Write([]byte(fmt.Sprintf("%d",os.Getpid())))
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(-1)
		}
	}
}

var cls cluster.Cluster

func GetNodeId() int {
	return 1
}

func Start() {
	cls.Init(GetNodeId())
	for _,s := range mapServiceName {
		s.OnInit()
	}
	for _,s := range mapServiceName {
		s.Start()
	}
	writeProcessPid()
	for {
		select {
		case <-sigs:
			fmt.Printf("Recv stop sig")
		default:
			time.Sleep(time.Second)
		}
	}


	close(closeSig)
	for _,s := range mapServiceName {
		s.Wait()
	}
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

func SetConfigDir(configdir string){
	cluster.SetConfigDir(configdir)
}