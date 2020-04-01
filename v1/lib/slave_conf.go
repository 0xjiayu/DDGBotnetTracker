package ddg_tracker

/*
	Salve conf struct
*/

type Conf struct {
	Data      []byte
	Signature []byte
}

type ConfData struct {
	CfgVer int
	Config MainConf
	Miner  []MinerConf
	Cmd    CmdConf
}

type MainConf struct {
	Interval string
}

type MinerConf struct {
	Exe string
	Md5 string
	Url string
}

type CmdConf struct {
	_msgpack   struct{} `msgpack:",omitempty"`
	AALocalSSH CmdConfDetail
	AAredis    CmdConfDetail
	AAssh      CmdConfDetail
	Sh         []ShConf
	Killer     []ProcConf
	LKProc     []ProcConf
}

type CmdConfDetail struct {
	_msgpack   struct{} `msgpack:",omitempty"`
	Id         int
	Version    int
	ShellUrl   string
	Duration   string
	NThreads   int
	IPDuration string
	GenLan     bool
	GenAAA     bool
	Timeout    string
	Ports      []int
}

type ShConf struct {
	Id      int
	Version int
	Line    string
	Timeout string
}

type ProcConf struct {
	_msgpack struct{} `msgpack:",omitempty"`
	Id       int
	Version  int
	Expr     string
	Timeout  string
}
