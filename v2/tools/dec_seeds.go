package main

import (
	//"bufio"
	"bytes"
	"compress/gzip"
	"encoding/base64"
	//"encoding/hex"
	"fmt"
	//"github.com/davecgh/go-spew/spew"
	"github.com/vmihailenco/msgpack"
	"io/ioutil"
	"net"
	"os"
)

const (
	RAW_FILE     string = "../data/seeds.raw"
	B64_ALPHABET string = "eDy54SH1N2s-Y7g3qnurvaTW_0BlCMJhfb6wtdGUcROXPAV9KEzIpFoi8xLjkmZQ"
)

var b64EncObj *base64.Encoding

func init() {
	b64EncObj = base64.NewEncoding(B64_ALPHABET).WithPadding(base64.NoPadding)
}

func main() {
	seedsRaw, err := ioutil.ReadFile(RAW_FILE)
	if err != nil {
		fmt.Printf("Failed to read raw file content.\n%s\n", err.Error())
		os.Exit(1)
	}

	b64Reader := base64.NewDecoder(b64EncObj, bytes.NewReader(seedsRaw))
	//gzbr := bytes.NewReader(b64DecRes)
	gzr, err := gzip.NewReader(b64Reader)
	if err != nil {
		fmt.Printf("Failed to create GZ reader object.\n%s\n", err.Error())
		os.Exit(1)
	}
	defer gzr.Close()

	msgpackBts, err := ioutil.ReadAll(gzr)
	if err != nil {
		fmt.Printf("Failed to GZ Decompress data.\n%s\n", err.Error())
		os.Exit(1)
	}

	//err = ioutil.WriteFile("seeds.mp", msgpackBts, 0755)
	//if err != nil {
	//	fmt.Printf("Failed to write gz file.\n%s\n", err.Error())
	//	os.Exit(1)
	//}

	var hostList []net.TCPAddr
	err = msgpack.Unmarshal(msgpackBts, &hostList)
	if err != nil {
		fmt.Printf("Failed to msgpack.Unmarshal().\n%s\n", err.Error())
		os.Exit(1)
	}

	dstFileFD, err := os.Create("seeds.list")
	if err != nil {
		fmt.Printf("Failed to create dst file.")
		os.Exit(1)
	}
	defer dstFileFD.Close()

	for _, hostAddr := range hostList {
		//fmt.Println(hostAddr.String())
		dstFileFD.WriteString(hostAddr.String() + "\n")
	}
	fmt.Println("Done.")
}
