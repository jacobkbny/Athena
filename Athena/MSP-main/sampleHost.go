package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"
)

func WriteHosts() {
	jsonTest()
}

type HostInfo struct {
	Hostname string
	Hostip   string
}
type HostInfoJSON struct {
	Systemname      string
	Hostinformation []HostInfo
	Timestamp       string
	Type            string
}

func jsonTest() {
	//hostinfos := []HostInfo{}
	hostinfos := make([]HostInfo, len(CurrentAddressTable))
	// hostinfos[0].Hostname = "N1"
	// hostinfos[0].Hostip = "111.111.111.111"
	// hostinfos[1].Hostname = "N2"
	// hostinfos[1].Hostip = "111.111.111.112"
	var AllHostName []string
	var AllHostAddress []string
	AllHostName = make([]string, 0)
	AllHostAddress = make([]string, 0)
	for _, V := range CurrentAddressTable {
		AllHostName = append(AllHostName, NodeNameTable[V])
		AllHostAddress = append(AllHostAddress, V)
	}
	for i := 0; i < len(CurrentAddressTable); i++ {
		hostinfos[i].Hostname = AllHostName[i]
		hostinfos[i].Hostip = AllHostAddress[i]
	}
	hostinfo := &HostInfoJSON{
		Systemname:      "system",
		Hostinformation: hostinfos,
		Timestamp:       time.Now().Format("2006-01-02 15:04:05"),
		Type:            "Current",
	}
	byteData, _ := json.Marshal(hostinfo)
	// fmt.Println(string(byteData))
	date := time.Now().Format("2006-01-02")
	logFolderPath := "./log"
	logFilePath := fmt.Sprintf("%s/%s-%s.txt", logFolderPath, date, "Hosts")
	if _, err := os.Stat(logFolderPath); os.IsNotExist(err) {
		os.MkdirAll(logFolderPath, 0666)
	}
	var isExistFile bool = true
	if _, err := os.Stat(logFilePath); err != nil {
		os.Create(logFilePath)
		isExistFile = false
	}
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Println("open error")
		panic(err)
	}
	if isExistFile {
		deleteLine(logFilePath, "]")
		WriteLog(logFile, ",")
	} else {
		WriteLog(logFile, "[")
	}
	logFile.Write(byteData)
	WriteLog(logFile, "\n]")
	logFile.Close()
}
