package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/process"
)

func NewServer(myPort string, group string, name string) {
	NewNode(myPort, group, name)
	setRoute()
}

func ServerStart(myPort string) {
	log.Println("노드 실행 : ", ConfigData.Public+":"+myPort)
	if err := http.ListenAndServe(":"+myPort, nil); err != nil {
		log.Println(err)
		return
	}
}

func TcpStart(myPort string) {
	port, _ := strconv.Atoi(myPort)
	port += 100
	log.Println(port)
	ln, err := net.Listen("tcp", ":"+fmt.Sprint(port))
	if err != nil {
		log.Println(err)
	}
	defer ln.Close()

	for {
		conn, err := ln.Accept() // 클라이언트가 연결되면 TCP 연결을 리턴
		log.Println("conn", conn)
		if err != nil {
			fmt.Println(err)
			continue
		}
		defer conn.Close() // main 함수가 끝나기 직전에 TCP 연결을 닫음
		go GetStatus(conn) // 패킷을 처리할 함수를 고루틴으로 실행
	}
}

func setRoute() {
	// http.HandleFunc("/PingReq", GetStatus)
	http.HandleFunc("/ChangeStrategy", GetStrategy)
	http.HandleFunc("/TableUpdateAlarm", TableUpdate)
	//http.HandleFunc("/DelayResult", delayResult)

	for k := range ConfigData.URL {
		http.HandleFunc(k, ServiceReq)
	}
}

// Check client IP if it is blacklist
func CheckBlacklist(clientIP string) bool {
	for _, v := range InitValue.Blacklist {
		if clientIP == v {
			return true
		}
	}
	return false
}

// Select Back-end URL
func SelectURL(reqURL string) string {
	var targetURL string

	for k, v := range ConfigData.URL {
		if reqURL == k {
			targetURL = v
			break
		}
	}
	return targetURL
}

// Get ping and check my memory status and then send response with memory status to MSP
func GetStatus(conn net.Conn) {
	for {
		var groupName string
		json.NewDecoder(conn).Decode(&groupName)
		if len(groupName) > 0 {
			cpu, mem, usage := GetMemoryUsage()
			log.Println("=====================================================")
			log.Println("cpu percent:", cpu)
			log.Println("memory percent:", mem)
			log.Println("memory usage:", usage)

			InitValue.Group = groupName

			logData := "Nodename," + InitValue.NodeName + ",clientIP,null,url,null,address," + ConfigData.Public + ":" + InitValue.MyPort + ",cpuUsed," + fmt.Sprint(cpu) + ",group," + InitValue.Group
			go func() {
				statusQue <- logData
			}()
			json.NewEncoder(conn).Encode(cpu)
		}
	}
}

// Get and calculate cpu and memory usage
func GetMemoryUsage() (string, string, string) {
	vm, _ := mem.VirtualMemory()
	total := vm.Total

	pid := os.Getpid()
	ps, _ := process.NewProcess(int32(pid))
	Mem, _ := ps.MemoryInfo()
	percent, _ := ps.MemoryPercent()
	vms := Mem.VMS
	usage := fmt.Sprint(((float32(total) * (percent / 100.0)) / float32(vms)) * 100.0)
	cpuPercent, _ := ps.CPUPercent()
	return fmt.Sprint(cpuPercent), fmt.Sprint(percent), usage
}

// Get and change strategy
func GetStrategy(w http.ResponseWriter, req *http.Request) {
	log.Println("Get request for changing strategy")

	var stgy string
	json.NewDecoder(req.Body).Decode(&stgy)
	log.Println("Change to", stgy)
	InitValue.Strategy = stgy
}

// Manage blacklist ip table from MSP
func TableUpdate(w http.ResponseWriter, req *http.Request) {
	log.Println("Get request(/TableUpdateAlarm) from MSP")
	UpdateBlacklist(req.Body)
}

// Sending semi-blackIP to MSP
func SendIP(ip string, code string) {
	if code == "warning" {
		data := "Nodename," + InitValue.NodeName + ",warning," + ip + ",danger,null"
		go func() {
			warningQue <- data
		}()
	} else if code == "danger" {
		data := "Nodename," + InitValue.NodeName + ",warning,null,danger," + ip
		go func() {
			warningQue <- data
		}()
	}
}

// Return client ip from http request
func getIP(r *http.Request) (string, error) {
	//Get IP from the X-REAL-IP header
	ip := r.Header.Get("X-REAL-IP")
	netIP := net.ParseIP(ip)
	if netIP != nil {
		return ip, nil
	}

	//Get IP from X-FORWARDED-FOR header
	ips := r.Header.Get("X-FORWARDED-FOR")
	splitIps := strings.Split(ips, ",")
	for _, ip := range splitIps {
		netIP := net.ParseIP(ip)
		if netIP != nil {
			return ip, nil
		}
	}

	//Get IP from RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return "", err
	}
	netIP = net.ParseIP(ip)
	if netIP != nil {
		return ip, nil
	}
	return "", fmt.Errorf("No valid ip found")
}

// HandleFunc about all of service requests
func ServiceReq(w http.ResponseWriter, req *http.Request) {
	startTime := time.Now()

	ip, err := getIP(req)
	if err != nil {
		log.Println(err)
	}

	// Check blacklist
	IsBlack := CheckBlacklist(ip)
	if IsBlack {
		log.Println("Access of blacklist is found. Block the request!")
		return
	}
	url_path := req.URL.Path

	log.Println(url_path, "접속, ClientIP:", ip)

	// Write log
	cpu, mem, usage := GetMemoryUsage()
	log.Println("=====================================================")
	log.Println("cpu percent:", cpu)
	log.Println("memory percent:", mem)
	log.Println("memory usage:", usage)

	logData := "Nodename," + InitValue.NodeName + ",clientIP," + ip + ",url," + url_path + ",address," + ConfigData.Public + ":" + InitValue.MyPort + ",cpuUsed," + cpu + ",group," + InitValue.Group
	go func() {
		statusQue <- logData
	}()

	if InitValue.Strategy == "ABNORMAL" {
		SendIP(ip, "danger")
	} else {
		cpuPer, _ := strconv.ParseFloat(cpu, 32)
		if cpuPer >= 5 && cpuPer < 8 {
			SendIP(ip, "warning")
		}
	}
	// targetURL := SelectURL(url_path)
	// res, err := http.Post("http://"+ConfigData.Public+":"+ConfigData.GatePort+targetURL, "application/json", req.Body)
	// closeResponse(res, err)
	totalTime := time.Since(startTime)
	vps := float64(totalTime) / float64(time.Millisecond)

	go func() {
		performanceQue <- "Nodename," + InitValue.NodeName + ",vps," + fmt.Sprint(vps)
	}()
	response := make(map[string]string)
	w.Header().Set("Content-Type", "application/json")
	response["key"] = "response"
	json.NewEncoder(w).Encode(response)
}

// start pBFT for delay
func SendReqPBFT(body io.Reader, ip string) {
	res, err := http.Post("http://"+ip+"/StartDelay", "application/json", body)
	closeResponse(res, err)
}

// Close response for preventing memory leak
func closeResponse(res *http.Response, err error) {
	if res != nil {
		defer res.Body.Close()
	}

	if err != nil {
		log.Println("log close error:", err)
	}

	_, errs := io.Copy(ioutil.Discard, res.Body)
	if errs != nil {
		log.Println("io discard error: ", errs)
	}
}

// Get result of delay and forwarding request to BackEnd
func delayResult(w http.ResponseWriter, req *http.Request) {
	targetURL := SelectURL(req.URL.Path)
	res, err := http.Post("http://"+ConfigData.Public+":"+ConfigData.MspPort+targetURL, "application/json", req.Body)
	closeResponse(res, err)
}
