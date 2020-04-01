package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"
	"tracker_v1/lib"

	_ "github.com/go-sql-driver/mysql"
	"github.com/hashicorp/memberlist"
	"github.com/jmoiron/sqlx"
	"github.com/vmihailenco/msgpack"
)

// Work Directory Config
var (
	fullPathWorkDir string
	logFile         string
	logDir          string
	slaveConfDir    string
	sampleDir       string
)

var version int     // latest version number of ddg
var tDate time.Time // Current TimeDate
var tDateStr string // Current TimeDate string
var logFileHdl *os.File
var logHdl *log.Logger
var memberMap map[string][]string

var quitCh chan bool  // sync main goroutine quit
var urlCh chan string // process url
var ccCh chan int     // limit http request related goroutine numbers
var dbCh chan int     // limit number of db connections
var routineNum int    // count all goroutines
var finishCnt int     // count all finished goroutines

var nodelist string // nodelist file path for cmd  parameter
var UID string      // uid of memberlist node

const (
	ccServerListFile string = "cc_server.list"
	MAX_HTTPCONN     int    = 2048
	MAX_ROUTINE      int    = 20480
	DEFAULTVER       int    = 4007
	DEFAULTBINDPORT  int    = 7946
	WORKDIR          string = "ddg_tracker"
	ROUTERIP         string = "<Your_WAN_Router_IP>"
)

var SKEY []byte = []byte{
	0x48, 0x29, 0xF5, 0x5F,
	0x8C, 0x1F, 0x43, 0x4E,
	0xF9, 0xB7, 0x2D, 0x43,
	0x3A, 0x69, 0x99, 0x27,
	0x79, 0x79, 0xC2, 0x16,
	0x5A, 0x6E, 0x77, 0x64,
	0x3E, 0x52, 0xFF, 0x2B,
	0x3B, 0x7D, 0x3F, 0xD9,
}

var ccServerIPList = &ddg_tracker.CCHostList{}
var sampleInfoList = &ddg_tracker.SampleInfo{}

// DB Config
const (
	dbUName        string = "root"
	dbPWord        string = "<Your_MySQL_PASSWD>"
	dbHost         string = "127.0.0.1"
	dbPort         string = "<Your_MySQL_Srv_Port>"
	dbName         string = "ddg"
	dbMaxOpenConns int    = 128
	dbMaxIdleConns int    = 0
)

var DbConn *sqlx.DB

func epilogue() {
	<-ccCh
	quitCh <- true
}

func init() {
	cstSh, _ := time.LoadLocation("Asia/Shanghai")
	tDate = time.Now().In(cstSh)
	tDateStr = tDate.Format("20060102150405")

	memberMap = make(map[string][]string)

	// Set dirs path
	fullPathWorkDir = path.Join("/var/", WORKDIR)
	logDir = path.Join(fullPathWorkDir, "log")
	slaveConfDir = path.Join(fullPathWorkDir, "slave_conf")
	sampleDir = path.Join(fullPathWorkDir, "sample")

	_, dirErr := os.Stat(fullPathWorkDir)
	// if dir doesn't exist, Initialize all the related dirs
	if dirErr != nil && !os.IsExist(dirErr) {
		// Initialize Work directory
		err := initWorkDir()
		if err != nil {
			fmt.Printf("%v", err)
			os.Exit(1)
		}
	}

	// <logDir>/20060102150405.log
	logFile = path.Join(logDir, strings.Join([]string{tDateStr, "log"}, "."))

	// Create log handler
	logFileHdl, err := os.OpenFile(logFile, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		fmt.Printf("Open logFile failed: %s\n", err)
		os.Exit(1)
	}
	logHdl = log.New(logFileHdl, "", log.LstdFlags)

	// Connect to mysql db
	dbPath := strings.Join([]string{dbUName, ":", dbPWord, "@tcp(", dbHost, ":", dbPort, ")/", dbName, "?charset=utf8mb4&collation=utf8mb4_unicode_ci&parseTime=true"}, "")
	db, err := sqlx.Open("mysql", dbPath)
	if err != nil {
		logHdl.Printf("Open mysql failed: %v", err)
		os.Exit(1)
	}
	DbConn = db
	DbConn.SetMaxOpenConns(dbMaxOpenConns)
	DbConn.SetMaxIdleConns(dbMaxIdleConns)
	DbConn.SetConnMaxLifetime(10 * time.Second)

	// Initialize the version number
	getLatestVersion()

	// Get UID str
	UID = ddg_tracker.GetUID()

	// Prepare sampleInfoList
	extractSampleInfo()

	quitCh = make(chan bool, MAX_ROUTINE)
	urlCh = make(chan string, MAX_HTTPCONN)
	// set ccHost chan chan and db chan to limit there connections
	ccCh = make(chan int, MAX_HTTPCONN)
	dbCh = make(chan int, dbMaxOpenConns)
	routineNum = 0
	finishCnt = 0

	flag.StringVar(&nodelist, "nodelist", "", "file path to nodelist")
}

func main() {
	defer logFileHdl.Close()

	flag.Parse()

	wanIP := ddg_tracker.GetWanIP()

	var nodeIPs []string
	if len(nodelist) == 0 {
		nodeLst, err := getHostList()
		if err != nil {
			logHdl.Printf("%+v", err)
			return
		}
		nodeIPs = nodeLst
	} else {
		// Read host:port from memberlist file by line
		fd, err := os.Open(nodelist)
		if err != nil {
			logHdl.Printf("File open failed: %+v\n", err)
			return
		}
		br := bufio.NewReader(fd)
		for {
			line, err := br.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					break
				}
			}
			nodeIPs = append(nodeIPs, strings.TrimSpace(line))
		}
		fmt.Println("Success to read hub list from file.")
		fd.Close()
	}

	ntConf := memberlist.NetTransportConfig{
		BindAddrs: []string{"127.0.0.1"},
		BindPort:  DEFAULTBINDPORT,
	}
	nt, err := memberlist.NewNetTransport(&ntConf)
	if err != nil {
		logHdl.Printf("Error ocurr while memberlist.NewNetTransport(): %+v", err)
		return
	}

	nodeUid := fmt.Sprintf("%d.%s", version, UID)

	mListConf := memberlist.DefaultWANConfig()
	mListConf.BindAddr = wanIP
	mListConf.BindPort = DEFAULTBINDPORT
	mListConf.Transport = nt
	mListConf.Name = nodeUid
	mListConf.SecretKey = SKEY
	mListConf.AdvertiseAddr = ROUTERIP
	mListConf.AdvertisePort = DEFAULTBINDPORT
	mListConf.Logger = logHdl

	logHdl.Printf("memconf:\n%v\n\n", mListConf)

	memList, err := memberlist.Create(mListConf)
	if err != nil {
		logHdl.Printf("Failed to create memberlist: %v", err)
		return
	}

	// Join an existing cluster by specifying at least one known member.
	n, err := memList.Join(nodeIPs)
	if err != nil {
		logHdl.Printf("Failed to join cluster: %v", err)
	} else {
		logHdl.Printf("Member Num: %d\n", n)
	}

	// Ask for members of the cluster
	for _, member := range memList.Members() {
		//fmt.Printf("Member: %s:%d --> %s\n", member.Addr, member.Port, member.Name)
		memberIP := fmt.Sprintf("%s", member.Addr)
		memberPort := fmt.Sprintf("%d", member.Port)
		nameInfo := strings.Split(member.Name, ".")
		_, ok := memberMap[memberIP]
		if !ok {
			memberMap[memberIP] = []string{memberIP, memberPort, nameInfo[0], nameInfo[1]}
		}
	}

	// shutdown memberlist before processing members' info
	memList.Shutdown()

	// Extract members info from log file, and insert them to DB
	extractMembersFromLog()

	// Insert member info to DB
	for _, memberInfo := range memberMap {
		memberIP := memberInfo[0]
		memberPort, _ := strconv.Atoi(memberInfo[1])
		memberVer, _ := strconv.Atoi(memberInfo[2])
		memberHash := memberInfo[3]

		// Insert new member info to DB
		routineNum++
		dbCh <- 1
		go func(ip string, port int, ver int, hash string) {
			_, err = DbConn.Exec("INSERT INTO tracker (ip, port, version, hash, tdate) VALUES (?, ?, ?, ?, ?)", ip, port, ver, hash, tDate)
			if err != nil {
				logHdl.Printf("Error ocurr while inserting (%s:%d -> %d.%s)\n%+v", ip, port, ver, hash, err)
			}
			<-dbCh
			quitCh <- true
		}(memberIP, memberPort, memberVer, memberHash)

		// Process an memberIP to test if a CC Server
		routineNum++
		ccCh <- 1
		go processCCHost(memberIP)
	}

	fmt.Println(routineNum)

	// process shell urls from urlCh
	for {
		time.Sleep(time.Duration(1) * time.Millisecond)
		select {
		case shellURL := <-urlCh:
			routineNum++
			ccCh <- 1
			go processShellUrl(shellURL)
		case quitFlag := <-quitCh:
			if quitFlag == true {
				finishCnt++
			}
			if finishCnt == routineNum {
				goto EPILOG
			}
		}
	}

EPILOG:
	// Write ccServerHostList to file
	ccServerListPath := path.Join(fullPathWorkDir, ccServerListFile)
	ccServerListFileHdl, err := os.OpenFile(ccServerListPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		logHdl.Printf("Failed to open ccServerListFile: %v", err)
	} else {
		bw := bufio.NewWriter(ccServerListFileHdl)

		ccServerIPList.L.Lock()
		for _, ccServer := range ccServerIPList.Hosts {
			ccServerInfo := fmt.Sprintf("%s\t%s\n", tDateStr, ccServer)
			_, err = bw.WriteString(ccServerInfo)
			if err != nil {
				logHdl.Printf("Failed to write ccServerInfo: %s\n", ccServerInfo)
			}
			logHdl.Printf("Success to write ccServerInfo: %s", ccServerInfo)
		}
		ccServerIPList.L.Unlock()

		bw.Flush()

		ccServerListFileHdl.Close()
	}
	logHdl.Printf("Done\n")
}

// Initialize work directories
func initWorkDir() error {
	err := os.MkdirAll(logDir, os.ModePerm)
	if err != nil {
		return fmt.Errorf("Error ocurr while MkdirAll(\"%s\"): %v\n", logDir, err)
	}

	err = os.MkdirAll(slaveConfDir, os.ModePerm)
	if err != nil {
		return fmt.Errorf("Error ocurr while MkdirAll(\"%s\"): %v\n", slaveConfDir, err)
	}

	err = os.MkdirAll(sampleDir, os.ModePerm)
	if err != nil {
		return fmt.Errorf("Error ocurr while MkdirAll(\"%s\"): %v\n", sampleDir, err)
	}

	return nil
}

// Get latest ddg version
// Get the latest version number from DB, OR return the const DEFAULTVER
func getLatestVersion() {
	row := DbConn.QueryRow("select MAX(version) from tracker")
	err := row.Scan(&version)
	if err != nil {
		version = DEFAULTVER
	}
}

// Get member host list from DB
// latest 1000 ip:port
func getHostList() ([]string, error) {
	var hostList []string
	rows, err := DbConn.Query("SELECT ip,port,version,hash FROM tracker ORDER BY id DESC LIMIT 1024")
	if err != nil {
		errInfo := fmt.Errorf("DB Error ocurr while getting host list from DB: \n%+v\n", err)
		return hostList, errInfo
	}
	defer rows.Close()

	for rows.Next() {
		var ip string
		var port int
		var ver int
		var hash string
		err = rows.Scan(&ip, &port, &ver, &hash)
		if err != nil {
			errInfo := fmt.Errorf("DB Error ocurr while fetching host info rows: \n%+v\n", err)
			return hostList, errInfo
		}
		//fmt.Printf("%s:%d:%d:%s\n", ip, port, ver, hash)
		hostInfo := strings.Join([]string{ip, strconv.Itoa(port)}, ":")
		hostList = append(hostList, hostInfo)
	}
	return hostList, nil
}

// Insert latest member host info to DB from log file
func extractMembersFromLog() {
	reErrLine := regexp.MustCompile("Failed to send gossip to")
	reMemberInfo := regexp.MustCompile(`(?m)\(([\d\.]+):(\d{4}):(\d{4}):(\w{32})\)`)

	logFileReadHdl, err := os.OpenFile(logFile, os.O_RDONLY, 0644)
	if err != nil {
		logHdl.Printf("Failed to Open Logfile again to Read: %+v", err)
		return
	}
	br := bufio.NewReader(logFileReadHdl)
	logContentBytes, err := ioutil.ReadAll(br)
	if err != nil {
		logHdl.Printf("Logfile: failed to ioutil.ReadAll() logfile content: %+v", err)
	}
	logFileReadHdl.Close()

	logContent := string(logContentBytes)
	if reErrLine.MatchString(logContent) {
		// err line example:
		// 2019/01/21 04:11:32 [ERR] memberlist: Failed to send gossip to (104.248.243.235:7946:3019:97b1d8834d5085db18d28fc2478c432e): write udp 127.0.0.1:7946->104.248.243.235:7946: sendto: invalid argument

		hostInfoList := reMemberInfo.FindAllStringSubmatch(logContent, -1)

		for _, hostInfo := range hostInfoList {
			_, ok := memberMap[hostInfo[1]]
			if !ok {
				memberMap[hostInfo[1]] = []string{hostInfo[1], hostInfo[2], hostInfo[3], hostInfo[4]}
			}
		}
	}
}

// Process a host ip to test if it's a CC Server
// If true, then download latest config data and latest malwr samples
// A CC Server serves as a config and sample downloading server
func processCCHost(hostIP string) {
	defer epilogue()

	var ddgConf ddg_tracker.Conf
	var ddgConfData ddg_tracker.ConfData

	httpHost := fmt.Sprintf("http://%s:8000/slave", hostIP)
	httpClient := &http.Client{}
	req, err := http.NewRequest("POST", httpHost, nil)
	if err != nil {
		logHdl.Printf("Gen Req Obj failed: %s", hostIP)
		return
	}
	req.Header.Add("Host", fmt.Sprintf("%s:8000", hostIP))
	req.Header.Add("Uid", UID)
	resp, err := httpClient.Do(req)
	if err != nil || resp.StatusCode != 200 {
		logHdl.Printf("Client.Do failed: %v", err)
		return
	}

	defer resp.Body.Close()
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil || len(bodyBytes) == 0 || (bodyBytes[0] != 0x82 && bodyBytes[1] != 0xa4) { // sig of msgPack
		logHdl.Printf("Read %s resp.body failed: %v", hostIP, err)
		return
	}

	err = msgpack.Unmarshal(bodyBytes, &ddgConf)
	if err != nil {
		logHdl.Printf("IP: %s msgpack.Unmarshal.l1 failed: %v\n", hostIP, err)
	}
	// fmt.Printf("Signature:\n%+v\n\n", hex.Dump(conf.Signature))
	err = msgpack.Unmarshal(ddgConf.Data, &ddgConfData)
	if err != nil {
		logHdl.Printf("msgpack.Unmarshal.l2 failed: %v\n", err)
	}

	logHdl.Printf("IP: %s decoded resp conf:\n%+v\n", hostIP, ddgConfData)

	// Write the raw conf data to file
	cstSh, _ := time.LoadLocation("Asia/Shanghai")
	timeStr := time.Now().In(cstSh).Format("20060102150405")
	confFileName := fmt.Sprintf("%s__%s.raw", strings.Replace(hostIP, ".", "_", -1), timeStr)
	confFilePath := path.Join(slaveConfDir, confFileName)
	confFileHdl, err := os.Create(confFilePath)
	if err != nil {
		logHdl.Printf("Failed to Create conf data file: %s", confFilePath)
	} else {
		if _, err = confFileHdl.Write(bodyBytes); err != nil {
			logHdl.Printf("Failed to Write conf data file: %s", confFilePath)
		}
		confFileHdl.Close()
	}

	// Sucees to parse config data, process i.sh on this server
	oldShellURL := fmt.Sprintf("http://%s:8000/i.sh", hostIP)
	urlCh <- oldShellURL

	newShellURL := ddgConfData.Cmd.AAredis.ShellUrl
	if strings.Compare(oldShellURL, newShellURL) != 0 {
		// Get Latest config data and malwr samples
		urlCh <- newShellURL
	}
}

// Process the url of i.sh
func processShellUrl(ishURL string) {
	defer epilogue()

	logHdl.Printf("Processing ShellURL: %s", ishURL)
	ishUri := ishURL[7:]

	// Get ishHost and store it to ccServerIPList
	ishHost := ishUri[:strings.Index(ishUri, "/")]
	ccServerIPList.Append(ishHost)

	err, newMD5, ishFilePath := DownLoadSample(ishURL)
	if err != nil {
		logHdl.Printf("%v", err)
		return
	}
	if (newMD5 == false) || (ishFilePath == "") {
		return
	}

	ishFileHdl, err := os.OpenFile(ishFilePath, os.O_CREATE|os.O_RDONLY, 0644)
	if err != nil {
		logHdl.Printf("Error ocurr when open ish file [%s]: %v", ishFilePath, err)
		return
	}
	defer ishFileHdl.Close()

	br := bufio.NewReader(ishFileHdl)
	ishContent, err := ioutil.ReadAll(br)
	if err != nil {
		logHdl.Printf("Error when ioutil.Readll([%s]): %v", ishFilePath, err)
		return
	}

	var reSampleUrl = regexp.MustCompile(`(?m)(http\:\/\/\d+\.\d+\.\d+\.\d+:\d+\/\w+\/\d+\/\w+)\.\$\(`)
	var reShUrl = regexp.MustCompile(`(?m)(http\:\/\/.+?\/i.sh)`)

	// Get url of i.sh and download i.sh
	latestShUrl := reShUrl.FindString(string(ishContent))
	if len(latestShUrl) > 0 {
		// Get the latest host of ish and store it to ccServerIPList
		latestShUri := latestShUrl[7:]
		latestShHost := latestShUri[:strings.Index(latestShUri, "/")]
		ccServerIPList.Append(latestShHost)

		// Download latest i.sh and process it
		err, _, _ = DownLoadSample(latestShUrl)
		if err != nil {
			logHdl.Printf("%+v", err)
		}
	}

	// Get url of binary samples and download them
	reSamRes := reSampleUrl.FindStringSubmatch(string(ishContent))
	if len(reSamRes) > 1 {
		samURL := reSamRes[1]
		i686SamUrl := samURL + ".i686"
		x86_64SamUrl := samURL + ".x86_64"

		err, _, _ = DownLoadSample(i686SamUrl)
		if err != nil {
			logHdl.Printf("%v", err)
		}

		err, _, _ = DownLoadSample(x86_64SamUrl)
		if err != nil {
			logHdl.Printf("%v", err)
		}
	}
}

// Download file to workdir
func DownLoadSample(fileURL string) (error, bool, string) {
	logHdl.Printf("Downloading Sample: %s", fileURL)
	var newMd5Flag bool = false // if got new file md5, set newMd5Flag=true

	currTS := fmt.Sprintf("%d", time.Now().UnixNano())

	uriStr := fileURL[7:]
	fileName := strings.Replace(strings.Replace(strings.Replace(uriStr, ".", "_", -1), ":", "__", -1), "/", "__", -1)
	filePath := path.Join(sampleDir, fileName)
	tmpFilePath := path.Join(sampleDir, fmt.Sprintf("%s%s", ".", currTS))
	finalFilePath := ""

	dlCmd := exec.Command("wget", fileURL, "-O", tmpFilePath, "-t 5", "--no-check-certificate")
	// Time limitation of downloading was set to 2.5h
	//dlCmd := exec.Command("curl", "-sSL", "--retry", "8", "-m", "9000", fileURL, "-o", tmpFilePath)
	//logHdl.Printf("%q\n", dlCmd.Args)

	var outErr bytes.Buffer
	dlCmd.Stderr = &outErr

	err := dlCmd.Run()
	if err != nil {
		errMsg := fmt.Errorf("%v:\n%v", err, outErr.String())
		// If crashed tmpfile exists,then delete it
		_, fileErr := os.Stat(tmpFilePath)
		if fileErr == nil || os.IsExist(fileErr) {
			os.Remove(tmpFilePath)
		}
		return errMsg, newMd5Flag, finalFilePath
	}

	md5CalcErr, fileMD5 := ddg_tracker.MD5Calc(tmpFilePath)
	if err != nil {
		// If crashed tmpfile exists,then delete it
		_, fileErr := os.Stat(tmpFilePath)
		if fileErr == nil || os.IsExist(fileErr) {
			os.Remove(tmpFilePath)
		}
		return md5CalcErr, newMd5Flag, finalFilePath
	}

	newSampleFlag := sampleInfoList.IsNew(fileMD5, fileName)
	if newSampleFlag < 3 { // New md5 or new file name(download URL) or both new
		// Reserves the sample file
		finalFilePath = path.Join(sampleDir, fmt.Sprintf("%s+%s", fileName, fileMD5)) // finaleFilePath: [downloadURL+md5]
		renameFileErr := os.Rename(tmpFilePath, finalFilePath)
		if renameFileErr != nil {
			// If crashed tmpfile exists,then delete it
			_, fileErr := os.Stat(tmpFilePath)
			if fileErr == nil || os.IsExist(fileErr) {
				os.Remove(tmpFilePath)
			}
			return renameFileErr, newMd5Flag, finalFilePath
		}

		// Store file info to sampleInfoList
		if newSampleFlag == 2 { // Only new md5
			newMd5Flag = true
			sampleInfoList.AddMD5(fileMD5)
			ddg_tracker.SendMsg2Slack("DDG", "New Sample MD5", fmt.Sprintf("MD5: %s\nURL: %s", fileMD5, fileURL))
		} else if newSampleFlag == 1 { // Only new File name(download URL)
			sampleInfoList.AddFile(filePath)
			ddg_tracker.SendMsg2Slack("DDG", "New Sample Download URL", fmt.Sprintf("MD5: %s\nURL: %s", fileMD5, fileURL))
		} else if newSampleFlag == 0 { // Both new
			newMd5Flag = true
			sampleInfoList.AddMD5(fileMD5)
			sampleInfoList.AddFile(filePath)
			ddg_tracker.SendMsg2Slack("DDG", "New Sample", fmt.Sprintf("MD5: %s\nURL: %s", fileMD5, fileURL))
		}
	}

	return nil, newMd5Flag, finalFilePath
}

// Read curr sample filenames and md5s to Fullfil sampleInfoList
func extractSampleInfo() {
	if sampleDir == "" {
		return
	}

	fiList, err := ioutil.ReadDir(sampleDir)
	if err != nil {
		logHdl.Println(err)
		os.Exit(1)
	}

	for _, fileInfo := range fiList {
		baseFileName := fileInfo.Name()

		// Ignore hidden file(tmp download file)
		if strings.HasPrefix(baseFileName, ".") {
			continue
		}

		nameAndMd5 := strings.Split(baseFileName, "+")
		if len(nameAndMd5) == 2 {
			sampleInfoList.AddFile(nameAndMd5[0])
			sampleInfoList.AddMD5(nameAndMd5[1])
		}
	}
}
