package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/vmihailenco/msgpack"
	"io/ioutil"
	"log"
	"sync"
	"time"
)

const CONFFILE string = "../data/slave_latest.raw"

type SignData struct {
	Signature []byte
	Data      []byte
}

type Paylaod struct {
	CfgVer int
	Miners []miner  `msgpack:"Miner"`
	Cmds   CmdTable `msgpack:"Cmd"`
}

type IntervalConf struct {
	Interval string
}

type CmdTable struct {
	_msgpack      struct{} `msgpack:",omitempty"`
	AALocalSSH    AAsshSt
	AAredis       AAredisSt
	AAssh         AAsshSt
	AAnexus       AAnexusSt
	AAsupervisord AAsupervisordSt
}

type miner struct {
	XRun
}

type XRun struct {
	Exe string
	Url string
	Md5 string
}

type AAsshSt struct {
	_msgpack struct{} `msgpack:",omitempty"`
	BaseOptions
	AAOptions
	UrlPath string
}

type AAredisSt struct {
	_msgpack struct{} `msgpack:",omitempty"`
	BaseOptions
	AAOptions
	ShellUrl string
}

type AAnexusSt struct {
	_msgpack struct{} `msgpack:",omitempty"`
	BaseOptions
	AAOptions
	ShellUrl string
}

type AAsupervisordSt struct {
	_msgpack struct{} `msgpack:",omitempty"`
	BaseOptions
	AAOptions
	ShellUrl string
}

type AALocalSSHSt struct {
	_msgpack struct{} `msgpack:",omitempty"`
	BaseOptions
	AAOptions
	UrlPath  string
	ShellUrl string
}

type BaseOptions struct {
	_msgpack struct{} `msgpack:",omitempty"`
	Id       int
	Version  int
	Timeout  string
	Result   result
	StartAt  int64
	handler  *func()
}

type AAOptions struct {
	_msgpack   struct{} `msgpack:",omitempty"`
	NThreads   int
	Duration   string
	IPDuration string
	GenLan     bool
	GenAAA     bool
	Ports      []int
}

type timeout struct {
	Duation time.Duration
}

type result struct {
	Map sync.Map
}

func main() {
	var slaveConf SignData
	var payloadConf Paylaod

	confData, err := ioutil.ReadFile(CONFFILE)
	if err != nil {
		log.Fatalf("Failed to Open and Read CONFFILE:\n%s", err.Error())
	}

	err = msgpack.Unmarshal(confData, &slaveConf)
	if err != nil {
		log.Fatalf("Failed to Unmashal Slave Config data:\n%s", err.Error())
	}

	fmt.Println("Slave Config Signature:\n------------------------------------------------------------------------------")
	fmt.Println(hex.Dump(slaveConf.Signature))

	fmt.Println("Slave Config Raw Payload Data:\n------------------------------------------------------------------------------")
	fmt.Println(hex.Dump(slaveConf.Data))

	err = msgpack.Unmarshal(slaveConf.Data, &payloadConf)
	if err != nil {
		log.Fatalf("Failed to Unmarshal Paylaod Data:\n%s", err.Error())
	}
	fmt.Println("Payload Config Plain Text:\n------------------------------------------------------------------------------")
	jsBytes, err := json.MarshalIndent(payloadConf, "", "    ")
	if err != nil {
		log.Fatalf("Failed to convert config struct data to JSON:\n%s", err.Error())
	}
	fmt.Println(string(jsBytes))
}
