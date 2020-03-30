package sysmodule

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/duanhf2012/originnet/log"
	"math/rand"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/duanhf2012/originnet/service"
	_ "github.com/go-sql-driver/mysql"
)

type SyncFun func()

const (
	MAX_EXECUTE_FUN = 10000
)

type DBExecute struct {
	syncExecuteFun   chan SyncFun
	syncExecuteExit  chan bool
}

type PingExecute struct {
	tickerPing *time.Ticker
	pintExit   chan bool
}

// DBModule ...
type DBModule struct {
	service.Module
	db               *sql.DB
	url              string
	username         string
	password         string
	dbname           string
	maxconn          int
	PrintTime        time.Duration

	pingCoroutine 	 PingExecute

	syncCoroutineNum int
	executeList  	 []DBExecute
	waitGroup    	 sync.WaitGroup
}

// Tx ...
type Tx struct {
	tx        *sql.Tx
	PrintTime time.Duration
}

// DBResult ...
type DBResult struct {
	Err          error
	LastInsertID int64
	RowsAffected int64
	res          *sql.Rows
	// 解码数据相关设置
	tag  string
	blur bool
}

// DBResult ...
type DBResultEx struct {
	LastInsertID int64
	RowsAffected int64

	rowNum  int
	RowInfo map[string][]interface{} //map[fieldname][row]sql.NullString
}

type DataSetList struct {
	dataSetList       []DBResultEx
	currentDataSetIdx int32
	tag               string
	blur              bool
}

// SyncDBResult ...
type SyncDBResult struct {
	sres chan DBResult
}

type SyncQueryDBResultEx struct {
	sres chan *DataSetList
	err  chan error
}

type SyncExecuteDBResult struct {
	sres chan *DBResultEx
	err  chan error
}

func (slf *DBModule) RunPing() {
	for {
		select {
		case <-slf.pingCoroutine.pintExit:
			log.Error("RunPing stopping %s...", fmt.Sprintf("%T", slf))
			return
		case <-slf.pingCoroutine.tickerPing.C:
			if slf.db != nil {
				slf.db.Ping()
			}
		}
	}
}

func (slf *DBModule) Init(maxConn, executeNum int, url string, userName string, password string, dbname string) error {
	slf.url = url
	slf.maxconn = maxConn
	slf.username = userName
	slf.password = password
	slf.dbname = dbname
	slf.syncCoroutineNum = executeNum

	if executeNum <= 0 {
		return fmt.Errorf("executeNum mast more than zero:%d", executeNum)
	}

	slf.executeList = []DBExecute{}
	for i := 0; i < executeNum; i++ {
		itemInfo := DBExecute{syncExecuteFun:make(chan SyncFun, MAX_EXECUTE_FUN), syncExecuteExit:make(chan bool, 1)}
		slf.executeList = append(slf.executeList, itemInfo)
	}
	slf.pingCoroutine = PingExecute{tickerPing : time.NewTicker(5*time.Second), pintExit : make(chan bool, 1)}

	rand.Seed(time.Now().Unix())

	return slf.Connect(slf.maxconn)
}

func (slf *DBModule) OnInit() error {
	for i := 0; i < slf.syncCoroutineNum; i++ {
		go slf.RunExecuteDBCoroutine(i)
	}
	go slf.RunPing()

	return nil
}

func (slf *DBModule) OnRelease() {
	for i := 0; i < slf.syncCoroutineNum; i++ {
		close(slf.executeList[i].syncExecuteExit)
	}
}

//Close ...
func (slf *DBResult) Close() {
	if slf.res != nil {
		slf.res.Close()
	}
}

//NextResult ...
func (slf *DBResult) NextResult() bool {
	if slf.Err != nil || slf.res == nil {
		return false
	}
	return slf.res.NextResultSet()
}

// SetSpecificTag ...
func (slf *DBResult) SetSpecificTag(tag string) *DBResult {
	slf.tag = tag
	return slf
}

// SetBlurMode ...
func (slf *DBResult) SetBlurMode(blur bool) *DBResult {
	slf.blur = blur
	return slf
}

// UnMarshal ...
func (slf *DBResult) UnMarshal(out interface{}) error {
	if slf.Err != nil {
		return slf.Err
	}
	tbm, err := dbResult2Map(slf.res)
	if err != nil {
		return err
	}
	//fmt.Println(tbm)
	v := reflect.ValueOf(out)
	if v.Kind() != reflect.Ptr {
		return errors.New("interface must be a pointer")
	}
	if v.Elem().Kind() == reflect.Struct {
		if len(tbm) != 1 {
			return fmt.Errorf("数据结果集的长度不匹配 len=%d", len(tbm))
		}
		return slf.mapSingle2interface(tbm[0], v)
	}
	if v.Elem().Kind() == reflect.Slice {
		return slf.mapSlice2interface(tbm, out)
	}
	return fmt.Errorf("错误的数据类型 %v", v.Elem().Kind())
}

func dbResult2Map(rows *sql.Rows) ([]map[string]string, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	count := len(columns)
	tableData := make([]map[string]string, 0)
	values := make([]string, count)
	valuePtrs := make([]interface{}, count)
	for rows.Next() {
		for i := 0; i < count; i++ {
			valuePtrs[i] = &values[i]
		}
		err := rows.Scan(valuePtrs...)
		if err != nil {
			fmt.Println(err)
		}
		entry := make(map[string]string)
		for i, col := range columns {
			entry[strings.ToLower(col)] = values[i]
		}
		tableData = append(tableData, entry)
	}
	return tableData, nil
}

func (slf *DBResult) mapSingle2interface(m map[string]string, v reflect.Value) error {
	t := v.Type()
	val := v.Elem()
	typ := t.Elem()

	if !val.IsValid() {
		return errors.New("数据类型不正确")
	}

	for i := 0; i < val.NumField(); i++ {
		value := val.Field(i)
		kind := value.Kind()
		tag := typ.Field(i).Tag.Get(slf.tag)
		if tag == "" {
			tag = typ.Field(i).Name
		}

		if tag != "" && tag != "-" {
			vtag := strings.Split(strings.ToLower(tag), ",")
			meta, ok := m[vtag[0]]
			if !ok {
				if !slf.blur {
					return fmt.Errorf("没有在结果集中找到对应的字段 %s", tag)
				}
				continue
			}
			if !value.CanSet() {
				return errors.New("结构体字段没有读写权限")
			}
			if len(meta) == 0 {
				continue
			}
			switch kind {
			case reflect.String:
				value.SetString(meta)
			case reflect.Float32, reflect.Float64:
				f, err := strconv.ParseFloat(meta, 64)
				if err != nil {
					return err
				}
				value.SetFloat(f)
			case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int:
				integer64, err := strconv.ParseInt(meta, 10, 64)
				if err != nil {
					return err
				}
				value.SetInt(integer64)
			case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint:
				integer64, err := strconv.ParseUint(meta, 10, 64)
				if err != nil {
					return err
				}
				value.SetUint(integer64)
			case reflect.Bool:
				b, err := strconv.ParseBool(meta)
				if err != nil {
					return err
				}
				value.SetBool(b)
			default:
				return errors.New("数据库映射存在不识别的数据类型")
			}
		}
	}
	return nil
}

func (slf *DBModule) SetQuerySlowTime(Time time.Duration) {
	slf.PrintTime = Time
}

func (slf *DBModule) IsPrintTimeLog(Time time.Duration) bool {
	if slf.PrintTime != 0 && Time >= slf.PrintTime {
		return true
	}
	return false
}

func (slf *DBResult) mapSlice2interface(data []map[string]string, in interface{}) error {
	length := len(data)

	if length > 0 {
		v := reflect.ValueOf(in).Elem()
		newv := reflect.MakeSlice(v.Type(), 0, length)
		v.Set(newv)
		v.SetLen(length)

		for i := 0; i < length; i++ {
			idxv := v.Index(i)
			if idxv.Kind() == reflect.Ptr {
				newObj := reflect.New(idxv.Type().Elem())
				v.Index(i).Set(newObj)
				idxv = newObj
			} else {
				idxv = idxv.Addr()
			}
			err := slf.mapSingle2interface(data[i], idxv)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// Connect ...
func (slf *DBModule) Connect(maxConn int) error {
	cmd := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8&parseTime=true&loc=%s&readTimeout=30s&timeout=15s&writeTimeout=30s",
		slf.username,
		slf.password,
		slf.url,
		slf.dbname,
		url.QueryEscape(time.Local.String()))

	db, err := sql.Open("mysql", cmd)
	if err != nil {
		return err
	}
	err = db.Ping()
	if err != nil {
		db.Close()
		return err
	}
	slf.db = db
	db.SetMaxOpenConns(maxConn)
	db.SetMaxIdleConns(maxConn)
	db.SetConnMaxLifetime(time.Second * 90)

	return nil
}

// Get ...
func (slf *SyncDBResult) Get(timeoutMs int) DBResult {
	timerC := time.NewTicker(time.Millisecond * time.Duration(timeoutMs)).C
	select {
	case <-timerC:
		break
	case rsp := <-slf.sres:
		return rsp
	}
	return DBResult{
		Err: fmt.Errorf("Getting the return result timeout [%d]ms", timeoutMs),
	}
}

func (slf *SyncQueryDBResultEx) Get(timeoutMs int) (*DataSetList, error) {
	timerC := time.NewTicker(time.Millisecond * time.Duration(timeoutMs)).C
	select {
	case <-timerC:
		break
	case err := <-slf.err:
		dataset := <-slf.sres
		return dataset, err
	}

	return nil, fmt.Errorf("Getting the return result timeout [%d]ms", timeoutMs)
}

func (slf *SyncExecuteDBResult) Get(timeoutMs int) (*DBResultEx, error) {
	timerC := time.NewTicker(time.Millisecond * time.Duration(timeoutMs)).C
	select {
	case <-timerC:
		break
	case err := <-slf.err:
		dataset := <-slf.sres
		return dataset, err
	}

	return nil, fmt.Errorf("Getting the return result timeout [%d]ms", timeoutMs)
}

func (slf *DBModule) CheckArgs(args ...interface{}) error {
	for _, val := range args {
		if reflect.TypeOf(val).Kind() == reflect.String {
			retVal := val.(string)
			if strings.Contains(retVal, "-") == true {
				return fmt.Errorf("CheckArgs is error arg is %+v", retVal)
			}
			if strings.Contains(retVal, "#") == true {
				return fmt.Errorf("CheckArgs is error arg is %+v", retVal)
			}
			if strings.Contains(retVal, "&") == true {
				return fmt.Errorf("CheckArgs is error arg is %+v", retVal)
			}
			if strings.Contains(retVal, "=") == true {
				return fmt.Errorf("CheckArgs is error arg is %+v", retVal)
			}
			if strings.Contains(retVal, "%") == true {
				return fmt.Errorf("CheckArgs is error arg is %+v", retVal)
			}
			if strings.Contains(retVal, "'") == true {
				return fmt.Errorf("CheckArgs is error arg is %+v", retVal)
			}
			if strings.Contains(strings.ToLower(retVal), "delete ") == true {
				return fmt.Errorf("CheckArgs is error arg is %+v", retVal)
			}
			if strings.Contains(strings.ToLower(retVal), "truncate ") == true {
				return fmt.Errorf("CheckArgs is error arg is %+v", retVal)
			}
			if strings.Contains(strings.ToLower(retVal), " or ") == true {
				return fmt.Errorf("CheckArgs is error arg is %+v", retVal)
			}
			if strings.Contains(strings.ToLower(retVal), "from ") == true {
				return fmt.Errorf("CheckArgs is error arg is %+v", retVal)
			}
			if strings.Contains(strings.ToLower(retVal), "set ") == true {
				return fmt.Errorf("CheckArgs is error arg is %+v", retVal)
			}
		}
	}

	return nil
}

// Query ...
func (slf *DBModule) Query(query string, args ...interface{}) DBResult {
	if slf.CheckArgs(args) != nil {
		ret := DBResult{}
		log.Error("CheckArgs is error :%s", query)
		ret.Err = fmt.Errorf("CheckArgs is error!")
		return ret
	}

	if slf.db == nil {
		ret := DBResult{}
		log.Error("cannot connect database:%s", query)
		ret.Err = fmt.Errorf("cannot connect database!")
		return ret
	}
	rows, err := slf.db.Query(query, args...)
	if err != nil {
		log.Error("Query:%s(%v)", query, err)
	}

	return DBResult{
		Err:  err,
		res:  rows,
		tag:  "json",
		blur: true,
	}
}

func (slf *DBModule) QueryEx(query string, args ...interface{}) (*DataSetList, error) {
	datasetList := DataSetList{}
	datasetList.tag = "json"
	datasetList.blur = true

	if slf.CheckArgs(args) != nil {
		log.Error("CheckArgs is error :%s", query)
		return &datasetList, fmt.Errorf("CheckArgs is error!")
	}

	if slf.db == nil {
		log.Error("cannot connect database:%s", query)
		return &datasetList, fmt.Errorf("cannot connect database!")
	}

	TimeFuncStart := time.Now()
	rows, err := slf.db.Query(query, args...)
	TimeFuncPass := time.Since(TimeFuncStart)

	if slf.IsPrintTimeLog(TimeFuncPass) {
		log.Error("DBModule QueryEx Time %s , Query :%s , args :%+v", TimeFuncPass, query, args)
	}
	if err != nil {
		log.Error("Query:%s(%v)", query, err)
		if rows != nil {
			rows.Close()
		}
		return &datasetList, err
	}
	defer rows.Close()

	for {
		dbResult := DBResultEx{}
		//取出当前结果集所有行
		for rows.Next() {
			if dbResult.RowInfo == nil {
				dbResult.RowInfo = make(map[string][]interface{})
			}
			//RowInfo map[string][][]sql.NullString //map[fieldname][row][column]sql.NullString
			colField, err := rows.Columns()
			if err != nil {
				return &datasetList, err
			}
			count := len(colField)
			valuePtrs := make([]interface{}, count)
			for i := 0; i < count; i++ {
				valuePtrs[i] = &sql.NullString{}
			}
			rows.Scan(valuePtrs...)

			for idx, fieldname := range colField {
				fieldRowData := dbResult.RowInfo[strings.ToLower(fieldname)]
				fieldRowData = append(fieldRowData, valuePtrs[idx])
				dbResult.RowInfo[strings.ToLower(fieldname)] = fieldRowData
			}
			dbResult.rowNum += 1
		}

		datasetList.dataSetList = append(datasetList.dataSetList, dbResult)
		//取下一个结果集
		hasRet := rows.NextResultSet()

		if hasRet == false {
			if rows.Err() != nil {
				log.Error( "Query:%s(%+v)", query, rows)
			}
			break
		}
	}

	return &datasetList, nil
}

// SyncQuery ...
func (slf *DBModule) SyncQuery(queryHas int, query string, args ...interface{}) SyncQueryDBResultEx {
	ret := SyncQueryDBResultEx{
		sres: make(chan *DataSetList, 1),
		err:  make(chan error, 1),
	}

	chanIndex := queryHas % len(slf.executeList)
	if chanIndex < 0 {
		chanIndex = rand.Intn(len(slf.executeList))
	}

	if len(slf.executeList[chanIndex].syncExecuteFun) >= MAX_EXECUTE_FUN {
		dbret := DataSetList{}
		ret.err <- fmt.Errorf("chan is full,sql:%s", query)
		ret.sres <- &dbret

		return ret
	}

	slf.executeList[chanIndex].syncExecuteFun <- func() {
		rsp, err := slf.QueryEx(query, args...)
		ret.err <- err
		ret.sres <- rsp
	}

	return ret
}

// Exec ...
func (slf *DBModule) Exec(query string, args ...interface{}) (*DBResultEx, error) {
	ret := &DBResultEx{}
	if slf.db == nil {
		log.Error("cannot connect database:%s", query)
		return ret, fmt.Errorf("cannot connect database!")
	}

	if slf.CheckArgs(args) != nil {
		log.Error("CheckArgs is error :%s", query)
		//return ret, fmt.Errorf("cannot connect database!")
		return ret, fmt.Errorf("CheckArgs is error!")
	}

	TimeFuncStart := time.Now()
	res, err := slf.db.Exec(query, args...)
	TimeFuncPass := time.Since(TimeFuncStart)
	if slf.IsPrintTimeLog(TimeFuncPass) {
		log.Error("DBModule QueryEx Time %s , Query :%s , args :%+v", TimeFuncPass, query, args)
	}
	if err != nil {
		log.Error("Exec:%s(%v)", query, err)
		return nil, err
	}

	ret.LastInsertID, _ = res.LastInsertId()
	ret.RowsAffected, _ = res.RowsAffected()

	return ret, nil
}

// SyncExec ...
func (slf *DBModule) SyncExec(queryHas int, query string, args ...interface{}) *SyncExecuteDBResult {
	ret := &SyncExecuteDBResult{
		sres: make(chan *DBResultEx, 1),
		err:  make(chan error, 1),
	}

	chanIndex := queryHas % len(slf.executeList)
	if chanIndex < 0 {
		chanIndex = rand.Intn(len(slf.executeList))
	}

	if len(slf.executeList[chanIndex].syncExecuteFun) >= MAX_EXECUTE_FUN {
		ret.err <- fmt.Errorf("chan is full,sql:%s", query)
		return ret
	}

	slf.executeList[chanIndex].syncExecuteFun <- func() {
		rsp, err := slf.Exec(query, args...)
		if err != nil {
			ret.err <- err
			return
		}

		ret.sres <- rsp
		return
	}

	return ret
}

func (slf *DBModule) RunExecuteDBCoroutine(has int) {
	slf.waitGroup.Add(1)
	defer slf.waitGroup.Done()
	for {
		select {
		case <-slf.executeList[has].syncExecuteExit:
			log.Error("stopping module %s...", fmt.Sprintf("%T", slf))
			return
		case fun := <-slf.executeList[has].syncExecuteFun:
			fun()
		}
	}
}

func (slf *DataSetList) UnMarshal(args ...interface{}) error {
	if len(slf.dataSetList) != len(args) {
		return errors.New(fmt.Sprintf("Data set len(%d,%d) is not equal to args!", len(slf.dataSetList), len(args)))
	}

	for _, out := range args {
		v := reflect.ValueOf(out)
		if v.Kind() != reflect.Ptr {
			return errors.New("interface must be a pointer")
		}

		if v.Kind() != reflect.Ptr {
			return errors.New("interface must be a pointer")
		}

		if v.Elem().Kind() == reflect.Struct {
			err := slf.rowData2interface(0, slf.dataSetList[slf.currentDataSetIdx].RowInfo, v)
			if err != nil {
				return err
			}
		}
		if v.Elem().Kind() == reflect.Slice {
			err := slf.slice2interface(out)
			if err != nil {
				return err
			}
		}

		slf.currentDataSetIdx = slf.currentDataSetIdx + 1
	}

	return nil
}

func (slf *DataSetList) slice2interface(in interface{}) error {
	length := slf.dataSetList[slf.currentDataSetIdx].rowNum
	if length == 0 {
		return nil
	}

	v := reflect.ValueOf(in).Elem()
	newv := reflect.MakeSlice(v.Type(), 0, length)
	v.Set(newv)
	v.SetLen(length)

	for i := 0; i < length; i++ {
		idxv := v.Index(i)
		if idxv.Kind() == reflect.Ptr {
			newObj := reflect.New(idxv.Type().Elem())
			v.Index(i).Set(newObj)
			idxv = newObj
		} else {
			idxv = idxv.Addr()
		}

		err := slf.rowData2interface(i, slf.dataSetList[slf.currentDataSetIdx].RowInfo, idxv)
		if err != nil {
			return err
		}
	}

	return nil
}

func (slf *DataSetList) rowData2interface(rowIdx int, m map[string][]interface{}, v reflect.Value) error {
	t := v.Type()
	val := v.Elem()
	typ := t.Elem()

	if !val.IsValid() {
		return errors.New("数据类型不正确")
	}

	for i := 0; i < val.NumField(); i++ {
		value := val.Field(i)
		kind := value.Kind()
		tag := typ.Field(i).Tag.Get(slf.tag)
		if tag == "" {
			tag = typ.Field(i).Name
		}

		if tag != "" && tag != "-" {
			vtag := strings.ToLower(tag)
			columnData, ok := m[vtag]
			if ok == false {
				if !slf.blur {
					return fmt.Errorf("Cannot find filed name %s", vtag)
				}
				continue
			}
			if len(columnData) <= rowIdx {
				return fmt.Errorf("datasource column is error %s", tag)
			}
			meta := columnData[rowIdx].(*sql.NullString)
			if !ok {
				if !slf.blur {
					return fmt.Errorf("没有在结果集中找到对应的字段 %s", tag)
				}
				continue
			}
			if !value.CanSet() {
				return errors.New("结构体字段没有读写权限")
			}

			if meta.Valid == false {
				continue
			}

			if len(meta.String) == 0 {
				continue
			}

			switch kind {
			case reflect.String:
				value.SetString(meta.String)
			case reflect.Float32, reflect.Float64:
				f, err := strconv.ParseFloat(meta.String, 64)
				if err != nil {
					return err
				}
				value.SetFloat(f)
			case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int:
				integer64, err := strconv.ParseInt(meta.String, 10, 64)
				if err != nil {
					return err
				}
				value.SetInt(integer64)
			case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint:
				integer64, err := strconv.ParseUint(meta.String, 10, 64)
				if err != nil {
					return err
				}
				value.SetUint(integer64)
			case reflect.Bool:
				b, err := strconv.ParseBool(meta.String)
				if err != nil {
					return err
				}
				value.SetBool(b)
			default:
				return errors.New("数据库映射存在不识别的数据类型")
			}
		}
	}
	return nil
}

// Begin starts a transaction.
func (slf *DBModule) Begin() (*Tx, error) {
	var txDBMoudule Tx
	txdb, err := slf.db.Begin()
	if err != nil {
		log.Error("Begin error:%s", err.Error())
		return &txDBMoudule, err
	}
	txDBMoudule.tx = txdb
	return &txDBMoudule, nil
}

// Rollback aborts the transaction.
func (slf *Tx) Rollback() error {
	return slf.tx.Rollback()
}

// Commit commits the transaction.
func (slf *Tx) Commit() error {
	return slf.tx.Commit()
}

// CheckArgs...
func (slf *Tx) CheckArgs(args ...interface{}) error {
	for _, val := range args {
		if reflect.TypeOf(val).Kind() == reflect.String {
			retVal := val.(string)
			if strings.Contains(retVal, "-") == true {
				return fmt.Errorf("CheckArgs is error arg is %+v", retVal)
			}
			if strings.Contains(retVal, "#") == true {
				return fmt.Errorf("CheckArgs is error arg is %+v", retVal)
			}
			if strings.Contains(retVal, "&") == true {
				return fmt.Errorf("CheckArgs is error arg is %+v", retVal)
			}
			if strings.Contains(retVal, "=") == true {
				return fmt.Errorf("CheckArgs is error arg is %+v", retVal)
			}
			if strings.Contains(retVal, "%") == true {
				return fmt.Errorf("CheckArgs is error arg is %+v", retVal)
			}
			if strings.Contains(retVal, "'") == true {
				return fmt.Errorf("CheckArgs is error arg is %+v", retVal)
			}
			if strings.Contains(strings.ToLower(retVal), "delete ") == true {
				return fmt.Errorf("CheckArgs is error arg is %+v", retVal)
			}
			if strings.Contains(strings.ToLower(retVal), "truncate ") == true {
				return fmt.Errorf("CheckArgs is error arg is %+v", retVal)
			}
			if strings.Contains(strings.ToLower(retVal), " or ") == true {
				return fmt.Errorf("CheckArgs is error arg is %+v", retVal)
			}
			if strings.Contains(strings.ToLower(retVal), "from ") == true {
				return fmt.Errorf("CheckArgs is error arg is %+v", retVal)
			}
			if strings.Contains(strings.ToLower(retVal), "set ") == true {
				return fmt.Errorf("CheckArgs is error arg is %+v", retVal)
			}
		}
	}

	return nil
}

// Query executes a query that returns rows, typically a SELECT.
func (slf *Tx) Query(query string, args ...interface{}) DBResult {
	if slf.CheckArgs(args) != nil {
		ret := DBResult{}
		log.Error("CheckArgs is error :%s", query)
		ret.Err = fmt.Errorf("CheckArgs is error!")
		return ret
	}

	if slf.tx == nil {
		ret := DBResult{}
		log.Error("cannot connect database:%s", query)
		ret.Err = fmt.Errorf("cannot connect database!")
		return ret
	}

	rows, err := slf.tx.Query(query, args...)
	if err != nil {
		log.Error("Tx Query:%s(%v)", query, err)
	}

	return DBResult{
		Err:  err,
		res:  rows,
		tag:  "json",
		blur: true,
	}
}

// IsPrintTimeLog...
func (slf *Tx) IsPrintTimeLog(Time time.Duration) bool {
	if slf.PrintTime != 0 && Time >= slf.PrintTime {
		return true
	}
	return false
}

// QueryEx executes a query that return rows.
func (slf *Tx) QueryEx(query string, args ...interface{}) (*DataSetList, error) {
	datasetList := DataSetList{}
	datasetList.tag = "json"
	datasetList.blur = true

	if slf.CheckArgs(args) != nil {
		log.Error("CheckArgs is error :%s", query)
		return &datasetList, fmt.Errorf("CheckArgs is error!")
	}

	if slf.tx == nil {
		log.Error("cannot connect database:%s", query)
		return &datasetList, fmt.Errorf("cannot connect database!")
	}

	TimeFuncStart := time.Now()
	rows, err := slf.tx.Query(query, args...)
	TimeFuncPass := time.Since(TimeFuncStart)

	if slf.IsPrintTimeLog(TimeFuncPass) {
		log.Error("Tx QueryEx Time %s , Query :%s , args :%+v", TimeFuncPass, query, args)
	}
	if err != nil {
		log.Error("Tx Query:%s(%v)", query, err)
		if rows != nil {
			rows.Close()
		}
		return &datasetList, err
	}
	defer rows.Close()

	for {
		dbResult := DBResultEx{}
		//取出当前结果集所有行
		for rows.Next() {
			if dbResult.RowInfo == nil {
				dbResult.RowInfo = make(map[string][]interface{})
			}
			//RowInfo map[string][][]sql.NullString //map[fieldname][row][column]sql.NullString
			colField, err := rows.Columns()
			if err != nil {
				return &datasetList, err
			}
			count := len(colField)
			valuePtrs := make([]interface{}, count)
			for i := 0; i < count; i++ {
				valuePtrs[i] = &sql.NullString{}
			}
			rows.Scan(valuePtrs...)

			for idx, fieldname := range colField {
				fieldRowData := dbResult.RowInfo[strings.ToLower(fieldname)]
				fieldRowData = append(fieldRowData, valuePtrs[idx])
				dbResult.RowInfo[strings.ToLower(fieldname)] = fieldRowData
			}
			dbResult.rowNum += 1
		}

		datasetList.dataSetList = append(datasetList.dataSetList, dbResult)
		//取下一个结果集
		hasRet := rows.NextResultSet()

		if hasRet == false {
			if rows.Err() != nil {
				log.Error("Tx Query:%s(%+v)", query, rows)
			}
			break
		}
	}

	return &datasetList, nil
}

// Exec executes a query that doesn't return rows.
func (slf *Tx) Exec(query string, args ...interface{}) (*DBResultEx, error) {
	ret := &DBResultEx{}
	if slf.tx == nil {
		log.Error("cannot connect database:%s", query)
		return ret, fmt.Errorf("cannot connect database!")
	}

	if slf.CheckArgs(args) != nil {
		log.Error("CheckArgs is error :%s", query)
		//return ret, fmt.Errorf("cannot connect database!")
		return ret, fmt.Errorf("CheckArgs is error!")
	}

	TimeFuncStart := time.Now()
	res, err := slf.tx.Exec(query, args...)
	TimeFuncPass := time.Since(TimeFuncStart)
	if slf.IsPrintTimeLog(TimeFuncPass) {
		log.Error("Tx QueryEx Time %s , Query :%s , args :%+v", TimeFuncPass, query, args)
	}
	if err != nil {
		log.Error("Tx Exec:%s(%v)", query, err)
		return nil, err
	}

	ret.LastInsertID, _ = res.LastInsertId()
	ret.RowsAffected, _ = res.RowsAffected()

	return ret, nil
}
