package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net"
	"net/http"
	"os"
)

// structure for init value
type valueSet struct {
	Strategy  string
	NodeName  string
	MyPort    string
	Group     string
	Blacklist []string
}

var InitValue valueSet

// structure for config.json
type Config struct {
	URL      map[string]string `json:"url"`
	Public   string            `json:"public"`
	MspPort  string            `json:"mspPort"`
	GatePort string            `json:"gatePort"`
}

var ConfigData Config

// structure for Node info
type Addr struct {
	NewNode  string `json:"node"`
	Type     string `json:"type"`
	Address  string `json:"address"`
	NodeName string `json:"nodeName"`
}

// new node notify and try to connect to MSP
func NewNode(myPort string, group string, name string) {
	// Load config.json
	ConfigData = LoadConfig()
	log.Println(ConfigData)

	// Save initial hash
	Hash = MakeHashOfConfig(ConfigData)

	// Get my ip address as string type
	var myIP string

	host, _ := os.Hostname()
	addrs, _ := net.LookupIP(host)
	for _, addr := range addrs {
		if ipv4 := addr.To4(); ipv4 != nil {
			myIP = ipv4.String()
		}
	}

	var myIpStruct Addr
	myIpStruct.Address = myIP
	myIpStruct.NewNode = myPort
	myIpStruct.Type = group
	myIpStruct.NodeName = name

	ipMarshal, _ := json.Marshal(myIpStruct)

	// Notify that new node want to join to MSP
	res, err := http.Post("http://"+ConfigData.Public+":"+ConfigData.MspPort+"/RegNewNode", "application/json", bytes.NewBuffer(ipMarshal))
	log.Println("IP table from MSP:", res)

	// Get blacklist ip table from MSP
	UpdateBlacklist(res.Body)

	// Init status variable
	if group == "1" {
		InitValue.Strategy = "NORMAL"
	} else {
		InitValue.Strategy = "ABNORMAL"
	}
	InitValue.NodeName = name
	InitValue.MyPort = myPort
	closeResponse(res, err)
}

// Update blacklist ip table from MSP
func UpdateBlacklist(body io.Reader) {
	// Empty blacklist table
	InitValue.Blacklist = make([]string, 0)

	json.NewDecoder(body).Decode(&InitValue.Blacklist)
	for _, v := range InitValue.Blacklist {
		log.Println(v)
	}
}
