package service

import (
	"fmt"
	"github.com/duanhf2012/originnet/util/timer"
	"time"
)

const InitModuleId = 1e18
type IModule interface {
	SetModuleId(moduleId int64) bool
	GetModuleId() int64
	AddModule(module IModule) (int64,error)
	GetModule(moduleId int64) IModule
	GetAncestor()IModule
	ReleaseModule(moduleId int64)
	NewModuleId() int64
	GetParent()IModule
	OnInit() error
	OnRelease()

	getBaseModule() IModule
}

//1.管理各模块树层关系
//2.提供定时器常用工具
type Module struct {
	moduleId int64
	parent IModule        //父亲
	self IModule        //父亲
	child map[int64]IModule //孩子们
	mapActiveTimer map[*timer.Timer]interface{}
	mapActiveCron map[*timer.Cron]interface{}

	dispatcher         *timer.Dispatcher //timer

	//根结点
	ancestor IModule      //始祖
	seedModuleId int64    //模块id种子
	descendants map[int64]IModule//始祖的后裔们
}

func (slf *Module) SetModuleId(moduleId int64) bool{
	if moduleId > 0 {
		return false
	}

	slf.moduleId = moduleId
	return true
}

func (slf *Module) GetModuleId() int64{
	return slf.moduleId
}

func (slf *Module) OnInit() error{
 	return nil
}

func (slf *Module) AddModule(module IModule) (int64,error){
	pAddModule := module.getBaseModule().(*Module)
	if pAddModule.GetModuleId()==0 {
		pAddModule.moduleId = slf.NewModuleId()
	}

	if slf.child == nil {
		slf.child = map[int64]IModule{}
	}
	_,ok := slf.child[module.GetModuleId()]
	if ok == true {
		return 0,fmt.Errorf("Exists module id %d",module.GetModuleId())
	}

	pAddModule.self = module
	pAddModule.parent = slf.self
	pAddModule.dispatcher = slf.GetAncestor().getBaseModule().(*Module).dispatcher
	pAddModule.ancestor = slf.ancestor

	err := module.OnInit()
	if err != nil {
		return 0,err
	}

	slf.child[module.GetModuleId()] = module
	slf.ancestor.getBaseModule().(*Module).descendants[module.GetModuleId()] = module

	return module.GetModuleId(),nil
}

func (slf *Module) ReleaseModule(moduleId int64){
	//pBaseModule :=  slf.GetModule(moduleId).getBaseModule().(*Module)
	pModule := slf.GetModule(moduleId).getBaseModule().(*Module)

	//释放子孙
	for id,_ := range pModule.child {
		slf.ReleaseModule(id)
	}
	pModule.self.OnRelease()
	for pTimer,_ := range pModule.mapActiveTimer {
		pTimer.Stop()
	}

	for pCron,_ := range pModule.mapActiveCron {
		pCron.Stop()
	}

	/*
	moduleId int64
		parent IModule        //父亲
		child map[int64]IModule //孩子们
		mapActiveTimer map[*timer.Timer]interface{}
		mapActiveCron map[*timer.Cron]interface{}

		dispatcher         *timer.Dispatcher //timer

		//根结点
		ancestor IModule      //始祖
		seedModuleId int64    //模块id种子
		descendants map[int64]IModule//始祖的后裔们
	*/

	delete(slf.child,moduleId)
	delete (slf.ancestor.getBaseModule().(*Module).descendants,moduleId)

	//清理被删除的Module
	pModule.self = nil
	pModule.parent = nil
	pModule.child = nil
	pModule.mapActiveTimer = nil
	pModule.mapActiveCron = nil
	pModule.dispatcher = nil
	pModule.ancestor = nil
	pModule.descendants = nil
}

func (slf *Module) NewModuleId() int64{
	slf.ancestor.getBaseModule().(*Module).seedModuleId+=1
	return slf.ancestor.getBaseModule().(*Module).seedModuleId
}

func (slf *Module) GetAncestor()IModule{
	return slf.ancestor
}

func (slf *Module) GetModule(moduleId int64) IModule{
	iModule,ok := slf.GetAncestor().getBaseModule().(*Module).descendants[moduleId]
	if ok == false{
		return nil
	}
	return iModule
}

func (slf *Module) getBaseModule() IModule{
	return slf
}


func (slf *Module) GetParent()IModule{
	return slf.parent
}

func (slf *Module) AfterFunc(d time.Duration, cb func()) *timer.Timer {
	if slf.mapActiveTimer == nil {
		slf.mapActiveTimer =map[*timer.Timer]interface{}{}
	}

	 tm := slf.dispatcher.AfterFuncEx(d,func(t *timer.Timer){
		cb()
		delete(slf.mapActiveTimer,t)
	 })

	 slf.mapActiveTimer[tm] = nil
	 return tm
}

func (slf *Module) CronFunc(cronExpr *timer.CronExpr, cb func()) *timer.Cron {
	if slf.mapActiveCron == nil {
		slf.mapActiveCron =map[*timer.Cron]interface{}{}
	}

	cron := slf.dispatcher.CronFuncEx(cronExpr, func(cron *timer.Cron) {
		cb()
		delete(slf.mapActiveCron,cron)
	})

	slf.mapActiveCron[cron] = nil
	return cron
}

func (slf *Module) OnRelease(){
}