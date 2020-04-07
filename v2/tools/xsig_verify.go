package main

import (
	"crypto/ed25519"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	//"github.com/davecgh/go-spew/spew"
	"github.com/vmihailenco/msgpack"
	//"io/ioutil"
	"os"
)

var PUB_KEY = []byte{
	0x20, 0x0A, 0x51, 0x81, 0x91, 0xE9, 0xF2, 0x54,
	0x78, 0xFC, 0x1E, 0x66, 0x7B, 0x8F, 0x8D, 0xAC,
	0xCF, 0x62, 0x28, 0x18, 0x46, 0xEC, 0x45, 0x7C,
	0xF5, 0xC3, 0xBA, 0x4C, 0x86, 0xB0, 0xB5, 0x41,
}

const (
	XSIG_RAW     string = "fOSIE4y3ZPcTuT7weiMSSr7-0-Vem5IfTxEbUirWGS9j5NsDJh2k54RsnK08lG-ECaHQ4ARiWy3mJs0O9HzBpP6iANY7cTHnPw_i-wNK7u8E7wfVYweLg5eKYe"
	B64_ALPHABET string = "eDy54SH1N2s-Y7g3qnurvaTW_0BlCMJhfb6wtdGUcROXPAV9KEzIpFoi8xLjkmZQ"
)

func main() {
	b64EncObj := base64.NewEncoding(B64_ALPHABET).WithPadding(base64.NoPadding)
	//spew.Dump(b64EncObj)
	b64DecRes, err := b64EncObj.DecodeString(XSIG_RAW)
	if err != nil {
		fmt.Printf("Failed to Base64 DecodeString().\n%s\n", err.Error())
		os.Exit(1)
	}

	fmt.Printf("Base64 decoding result:\n%s\n", hex.Dump(b64DecRes))

	var sigInfo map[string]interface{}
	err = msgpack.Unmarshal(b64DecRes, &sigInfo)
	if err != nil {
		fmt.Printf("Failed to msgpack.Unmarshal():\n%s\n", err.Error())
		os.Exit(1)
	}
	//fmt.Printf("%+v", sigInfo)
	//fmt.Printf("C2 List: %+v\n\n", sigInfo["a"])
	ccList, ok := sigInfo["a"].([]interface{})
	if !ok {
		fmt.Println("Failed to conver cc List")
		os.Exit(1)
	}
	cc, ok := ccList[0].(string)
	if !ok {
		fmt.Printf("Failt to convert cc")
		os.Exit(1)
	}
	fmt.Printf("CC:%s\n\n", cc)

	sig, ok := sigInfo["s"].([]byte)
	if !ok {
		fmt.Printf("Failed to convert sig data.\n")
		os.Exit(1)
	}
	fmt.Printf("Signature:\n%s\n", hex.Dump(sig))

	sha512Hdl := sha512.New()
	sha512Hdl.Write([]byte(cc))
	sha512hashVal := sha512Hdl.Sum(nil)
	fmt.Printf("CC SHA512 Hash value:\n%s\n\n", hex.EncodeToString(sha512hashVal))

	ok = ed25519.Verify(PUB_KEY, sha512hashVal, sig)
	if !ok {
		fmt.Println("Failed to ED25519 Verify.")
		os.Exit(1)
	}

	fmt.Printf("ED25519 Verify SUCEESS.\n")

	fmt.Println("Done.")
}
