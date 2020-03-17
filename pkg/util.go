package ddg_tracker

/*
	Some func utils
*/

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"unsafe"

	"github.com/shirou/gopsutil/host"
)

const (
	DEFAULT_UID_HASH string = "d41d8cd98f00b204e9800998ecf8427e"
	DEFAULT_WANIP    string = "Your_DEFAULT_WAN_IP"
	SLACK_MSG_API    string = "https://hooks.slack.com/services/<Your_Slack_App_Token>"
)

var WANIP_APIS = []string{
	"http://v4.ident.me/",
	"http://bot.whatismyipaddress.com/",
	"https://www.fbisb.com/ip.php",
	"https://api.ipify.org",
	"https://ipecho.net/plain",
	"http://icanhazip.com",
	"http://checkip.amazonaws.com",
}

// Get uniq Uid
// Uid is generated from Host info and Net.Interfaces info
func GetUID() string {
	md5Val := DEFAULT_UID_HASH
	md5Hash := md5.New()

	// Get Host info
	resultHostInfo, err := host.Info()
	if err != nil {
		return md5Val
	} else {
		hostInfo, err := json.Marshal(resultHostInfo)
		if err != nil {
			return md5Val
		} else {
			md5Hash.Write(hostInfo)
		}
	}

	// Get Interfaces info
	resultInterfaces, err := net.Interfaces()
	if err != nil {
		return md5Val
	} else {
		interfaceInfo, err := json.Marshal(resultInterfaces)
		if err != nil {
			return md5Val
		} else {
			md5Hash.Write(interfaceInfo)
		}
	}

	md5Val = fmt.Sprintf("%x", md5Hash.Sum(nil))
	return md5Val
}

// Get WANIP
// Get WANIP from HTTP Request to public service
func GetWanIP() string {
	httpClient := &http.Client{}
	for _, wanIPUrl := range WANIP_APIS {
		req, err := http.NewRequest("GET", wanIPUrl, nil)
		if err != nil {
			continue
		}
		req.Header.Add("User-Agent", "curl/7.58.0")
		resp, err := httpClient.Do(req)
		if err == nil && resp.StatusCode == 200 {
			defer resp.Body.Close()
			bodyBytes, err := ioutil.ReadAll(resp.Body)
			if err == nil {
				return *(*string)(unsafe.Pointer(&bodyBytes))
			}
		}
	}
	return DEFAULT_WANIP
}

// Calculate file's MD5 Value
func MD5Calc(filePath string) (error, string) {
	file, err := os.Open(filePath)
	if err != nil {
		return err, ""
	}
	defer file.Close()

	md5hash := md5.New()
	io.Copy(md5hash, file)
	md5Str := fmt.Sprintf("%x", md5hash.Sum(nil))
	return nil, md5Str
}

// Send message to slack channel
func SendMsg2Slack(botName, msgTitle, msgTxt string) {
	msgContent := fmt.Sprintf("*[%s] %s*:\n```%s```", botName, msgTitle, msgTxt)

	msgStru := make(map[string]string)
	msgStru["text"] = msgContent

	bytesData, err := json.Marshal(msgStru)
	if err != nil {
		return
	}

	postData := bytes.NewReader(bytesData)
	req, err := http.NewRequest("POST", SLACK_MSG_API, postData)
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	client := http.Client{}
	client.Do(req)
}
