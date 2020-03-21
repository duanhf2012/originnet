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

var closeSig chan bool
var sigs chan os.Signal
var cls cluster.Cluster

func init() {
	closeSig = make(chan bool,1)
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

func GetNodeId() int {
	return 1
}

func Init(){
	cls.Init(GetNodeId())
	service.Init(closeSig)
}

func Start() {
	cls.Start()
	service.Start()
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
	service.WaitStop()
}


func Setup(s service.IService) bool {
	return service.Setup(s)
}

func GetService(servicename string) service.IService {
	return service.GetService(servicename)
}

func SetConfigDir(configdir string){
	cluster.SetConfigDir(configdir)
}