package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type ReqData struct {
	Ip            string `json:"ip"`
	IsBad         string `json:"isBad"`
	Alias         string `json:"alias"`
	Txid          string `json:"txid"`
	WalletAddress string `json:"walletAddress"`
	URL_Path      string `json:"urlPath"`
	Tx            Transaction
}
type Status struct {
	Address     string `json:"address"`
	GroupName   string `json:"groupName"`
	MemoryUsage int    `json:"mem"`
	CpuUsage    int    `json:"cpuUsage"`
}
type Addr struct {
	NewNode  string `json:"node"`
	Type     string `json:"type"`
	Address  string `json:"address"`
	NodeName string `json:"nodeName"`
}
type Transaction struct {
	TxID        [32]byte `json:"TxID"`
	TimeStamp   []byte   `json:"Timestamp"`   // 블럭 생성 시간
	Applier     []byte   `json:"Applier"`     // 신청자
	Company     []byte   `json:"Company"`     // 경력회사
	CareerStart []byte   `json:"CareerStart"` // 경력기간
	CareerEnd   []byte   `json:"CareerEnd"`
	Payment     []byte   `json:"Payment"` // 결제수단
	Job         []byte   `json:"Job"`     // 직종, 업무
	Proof       []byte   `json:"Proof"`   // 경력증명서 pdf
	WAddr       []byte   `json:"Address"` // 지갑 주소
	Sign        []byte   `json:"Sign"`
}
type ConfigData struct {
	Port     string `json:"port"`
	DataBase string `json:"dataBase"`
	User     struct {
		UserName     string `json:"userName"`
		PassWord     string `json:"passWord"`
		DataBaseName string `json:"dataBaseName"`
	}
	Threshold string `json:"threshold"`
	Revive    string `json:"revive"`
	PBFT      string `json:"pBFT"`
	Gateway   string `json:"gateway"`
}

var CurrentAddressTable map[string]string
var ReadyAddressTable map[string]string
var ZombieAddressTable map[string]string

// var PbftHostAddressTable map[string]string
// var PbftReadyAddressTable map[string]string
var StatusOfAll []*Status
var MyPort string
var Threshold float64
var Revive float64
var ipBlackList []string
var config ConfigData
var Hash [32]byte
var result bool

// Key = address , value = NodeName
var NodeNameTable map[string]string
var Strategy string
var Boot time.Time
var SUT string
var OneTime int

func init() {
	temp, err := LoadConfig()
	if err != nil {
		fmt.Println(err)
	}
	config = temp
	num, err := strconv.ParseFloat(config.Threshold, 32)
	digit, err := strconv.ParseFloat(config.Revive, 32)
	Threshold = num
	Revive = digit
	Hash = MakeHashofConfig(config)
	Strategy = "NORMAL"
	OneTime = 0
}
func Server() {
	Boot = time.Now()
	fmt.Println(GetMyIP())
	CurrentAddressTable = make(map[string]string)
	ReadyAddressTable = make(map[string]string)
	ZombieAddressTable = make(map[string]string)
	ipBlackList = make([]string, 0)
	NodeNameTable = make(map[string]string)
	Client = make(map[string]net.Conn)
	// PbftHostAddressTable = make(map[string]string)
	// PbftReadyAddressTable = make(map[string]string)
	Handlers()
	// OpenLogFile()
	logFile := OpenLogFile("General")
	WriteLog(logFile, "logging,General log,systemname,MSP"+",starttime,"+Boot.Format("2006-01-02 15:04:05")+","+"name,Atena,"+"power,on,"+"strategy,"+Strategy+","+"enlapsedTime,"+time.Since(Boot).String())
	logFile.Close()
	// go func() {
	// 	for {
	// 		OpenConnection(config.DataBase, config.User.UserName, config.User.PassWord, config.User.DataBaseName)
	// 		time.Sleep(6 * time.Hour)
	// 	}
	// }()
	go func() {
		for {
			time.Sleep(10 * time.Second)
			// var Total float64
			Total := float64(len(CurrentAddressTable) + len(ReadyAddressTable) + len(ZombieAddressTable))
			CurrentLen := float64(len(CurrentAddressTable))
			ZombieLen := float64(len(ZombieAddressTable))
			if Total > 0 {
				temp := fmt.Sprint((1.0 / CurrentLen) * 100.0)
				PPS := strings.Split(temp, ".")
				Haza := fmt.Sprint((ZombieLen / Total) * 100.0)
				Hazardeous := strings.Split(Haza, ".")
				logFile := OpenLogFile("Performance")
				WriteLog(logFile, "ppn,"+PPS[0]+",sut,"+SUT+",hazardeous,"+Hazardeous[0]+",totalNode,"+fmt.Sprint(Total))
				defer logFile.Close()
			}
		}
	}()
	go func() {
		for {
			time.Sleep(10 * time.Second)
			logFile := OpenLogFile("General")
			WriteLog(logFile, "starttime,"+Boot.Format("2006-01-02 15:04:05")+","+"name,Atena,"+"power,on,"+"strategy,"+Strategy+","+"enlapsedTime,"+time.Since(Boot).String())
			defer logFile.Close()
		}
	}()
	//check config file if any changes are made
	go func() {
		for {
			CheckConfig()
			time.Sleep(3000 * time.Millisecond)
		}
	}()
	// go func() {
	// 	for {
	// 		// WriteLog("power:on" + "\n")
	// 		logFile := OpenLogFile("General")
	// 		WriteLog(logFile, "starttime,"+Boot.Format("2006-01-02 15:04:05")+","+"name,Atena,"+"power,on,"+"strategy,"+Strategy+","+"enlapsedTime,"+time.Since(Boot).String())
	// 		defer logFile.Close()
	// 		time.Sleep(10000 * time.Millisecond)
	// 	}
	// }()
	// Check who is working as a host in the GateWay
	go func() {
		for {
			if len(CurrentAddressTable) >= 4 {
				WriteHosts()
				time.Sleep(1 * time.Minute)
			}
		}
	}()
	go func() {
		for {
			//Check if the attack is over by checking the length of ZombieAddressTable for 5 min
			if Strategy != "NORMAL" {
				//CheckLazyBoyHost()
				CheckZombie()
				// WriteLog("strategy:" + Strategy + "\n")
				// logFile := OpenLogFile("General")
				// WriteLog(logFile, "starttime,"+Boot.Format("2006-01-02 15:04:05")+","+"name,Atena,"+"power,on,"+"strategy,"+Strategy+","+"enlapsedTime,"+time.Since(Boot).String())
			}
		}
	}()
	// put the Node in Ready to Current when strategy is not 'NORMAL'
	// go func() {
	// 	for {
	// 		if Strategy != "NORMAL" && len(ReadyAddressTable) >0 {
	// 			time.Sleep(1 * time.Minute)
	// 			ScaleUp()
	// 		}
	// 	}
	// }()
	go func() {
		for {
			Total := len(CurrentAddressTable) + len(ReadyAddressTable) + len(ZombieAddressTable)
			if Total > 0 {
				time.Sleep(500 * time.Millisecond)
				PingReq()
			}
		}
	}()

	log.Fatal(http.ListenAndServe(":"+config.Port, nil))
	defer CloseConnection(Client)

}
func LoadConfig() (ConfigData, error) {
	temp := new(ConfigData)
	file, err := os.Open("config.json")
	defer file.Close()
	if err != nil {
		log.Fatal(err)
	}
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&temp)
	if err != nil {
		log.Fatal(err)
	}
	return *temp, err
}
func CheckConfig() {
	temp, err := LoadConfig()
	if err != nil {
		fmt.Println(err)
	}
	tempHash := MakeHashofConfig(temp)
	if tempHash != Hash {
		config = temp
		Hash = tempHash
		num, err := strconv.ParseFloat(config.Threshold, 32)
		if err != nil {
			// WriteLog("error: " + err.Error() + "\n")
			logFile := OpenLogFile("Error")
			WriteLog(logFile, "logging,error log,system-name,MSP,errmsg,"+"ParseError")
			logFile.Close()
		}
		digit, err := strconv.ParseFloat(config.Revive, 32)
		if err != nil {
			logFile := OpenLogFile("Error")
			WriteLog(logFile, "logging,error log,system-name,MSP,errmsg,"+"ParseError")
			logFile.Close()
		}
		Threshold = num
		Revive = digit
		// WriteLog("Threshold:" + fmt.Sprint(Threshold) + "\n")
		logFile := OpenLogFile("Changes")
		WriteLog(logFile, "logging,changes log,system-name,MSP,"+"threshold,"+fmt.Sprint(Threshold)+",revive,"+fmt.Sprint(Revive))
		defer logFile.Close()
	}
}
func MakeHashofConfig(config ConfigData) [32]byte {
	data := PrepareData(config)
	Hash := sha256.Sum256(data)
	return Hash
}
func PrepareData(config ConfigData) []byte {
	data := bytes.Join([][]byte{
		[]byte(config.DataBase),
		[]byte(config.Port),
		[]byte(config.Threshold),
		[]byte(config.User.DataBaseName),
		[]byte(config.User.PassWord),
		[]byte(config.User.UserName),
		[]byte(config.Revive),
		[]byte(config.PBFT),
		[]byte(config.Gateway),
	}, []byte{})
	return data
}
func Handlers() {
	http.HandleFunc("/RegNewNode", RegNewNode)
	http.HandleFunc("/Switch", SwtichStrategy)
	// http.HandleFunc("/SendBlackIP", SendBlackIP)
	// http.HandleFunc("/GetLazyBoy", GetLazyBoy)
	// http.HandleFunc("/Rest", Rest)
}
func GetMyIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}

// UserName , PassWord , DataBaseName
// Connection to the DataBase
func OpenConnection(DataBase string, UserName string, PassWord string, DataBaseName string) {
	DB, err := sql.Open(DataBase, UserName+":"+PassWord+"@tcp(127.0.0.1:3306)/"+DataBaseName)
	if err != nil {
		fmt.Println(err)
	}
	defer DB.Close()
	var K int
	var V string
	var temp []string
	rows, err := DB.Query("SELECT * FROM IPBlackList")
	if err != nil {
		logFile := OpenLogFile("Error")
		WriteLog(logFile, "error,"+err.Error())
		logFile.Close()
	}
	defer rows.Close()
	for rows.Next() {
		err := rows.Scan(K, V)
		if err != nil {
			logFile := OpenLogFile("Error")
			WriteLog(logFile, "error,"+err.Error())
			logFile.Close()
		}
		temp[K] = V
	}
	if ipBlackList == nil {
		ipBlackList = temp
	} else {
		result = CompareIpBlackList(temp)
	}
	TableUpdateAlarm()
}
func CompareIpBlackList(temp []string) bool {
	if len(ipBlackList) != len(temp) {
		// WriteLog("IP BlackList has been Updated" + "\n")
		ipBlackList = temp
	}
	return true
}
func TableUpdateAlarm() {
	//open the connection to the DB(Mysql) and Pull IP BlackList from the DB
	if result {
		Data, err := json.Marshal(ipBlackList)
		if err != nil {
			logFile := OpenLogFile("Error")
			WriteLog(logFile, "error,"+err.Error())
			logFile.Close()
		}
		for _, V := range CurrentAddressTable {
			_, err := http.Post("http://"+V+"/TableUpdateAlarm", "application/json", bytes.NewBuffer(Data))
			if err != nil {
				logFile := OpenLogFile("Error")
				WriteLog(logFile, "error,"+err.Error())
				logFile.Close()
			}
		}
		for _, V := range ReadyAddressTable {
			_, err := http.Post("http://"+V+"/TableUpdateAlarm", "application/json", bytes.NewBuffer(Data))
			if err != nil {
				logFile := OpenLogFile("Error")
				WriteLog(logFile, "error,"+err.Error())
				logFile.Close()
			}
		}
		for _, V := range ZombieAddressTable {
			_, err := http.Post("http://"+V+"/TableUpdateAlarm", "application/json", bytes.NewBuffer(Data))
			if err != nil {
				logFile := OpenLogFile("Error")
				WriteLog(logFile, "error,"+err.Error())
				logFile.Close()
			}
		}
	}
}
func PingReq() {
	for Node, NodeAddress := range CurrentAddressTable {
		// var Data []byte
		// Data = make([]byte, 0)
		temp := strings.Split(NodeAddress, ":")
		port, err := strconv.Atoi(temp[1])
		if Client[temp[0]+":"+fmt.Sprint(port+100)] != nil {
			err = json.NewEncoder(Client[temp[0]+":"+fmt.Sprint(port+100)]).Encode("Current")
			// if err exist
			if err != nil {
				logFile := OpenLogFile("Disconnection")
				WriteLog(logFile, "logging,disconnection log,system-name,MSP,node-name,"+NodeNameTable[NodeAddress]+",type,1")
				defer logFile.Close()
				Client[temp[0]+":"+fmt.Sprint(port+100)].Close()
				delete(CurrentAddressTable, Node)
				fmt.Println("delete CurrentAddressTable")
				delete(Client, temp[0]+":"+fmt.Sprint(port+100))
				fmt.Println("delete Client")
				if len(ReadyAddressTable) > 0 {
					for K, V := range ReadyAddressTable {
						NewNode := map[string]string{
							"newIp":    V,
							"zombieIp": NodeAddress,
						}
						address, err := json.Marshal(NewNode)
						if err != nil {
							logFile := OpenLogFile("Error")
							WriteLog(logFile, "logging,error log,system-name,MSP,errmsg,"+"ParseError")
							defer logFile.Close()
						}
						_, err = http.Post("http://"+GetMyIP()+":7000/modifyHost", "application/json", bytes.NewBuffer(address))
						fmt.Println("Delete Current and Send Ready")
						if err != nil {
							logFile := OpenLogFile("Error")
							WriteLog(logFile, "logging,error log,system-name,MSP,errmsg,"+"Connection error")
							defer logFile.Close()
						}
						CurrentAddressTable[K] = V
						delete(ReadyAddressTable, K)
						WriteHosts()
						break
					}
				} else {
					delete(CurrentAddressTable, Node)
				}
			} else {
				response, err := bufio.NewReader(Client[temp[0]+":"+fmt.Sprint(port+100)]).ReadString('\n')
				if response != "" {
					temp := response[1 : len(response)-2]
					if err != nil {
						logFile := OpenLogFile("Error")
						WriteLog(logFile, "logging,error log,system-name,MSP,errmsg,"+"Connection error")
						defer logFile.Close()
					}
					// Memory := string(Data[1 : n-2])
					fmt.Println("CPU:", temp)
					MemoryUsage, err := strconv.ParseFloat(temp, 32)
					if err != nil {
						logFile := OpenLogFile("Error")
						WriteLog(logFile, "logging,error log,system-name,MSP,errmsg,"+"ParseError")
						defer logFile.Close()
					}
					fmt.Printf("%s's Memory usage: %f\n", NodeNameTable[NodeAddress], MemoryUsage)
					if MemoryUsage >= Threshold {
						fmt.Println("Current to Zombie")
						if Strategy == "NORMAL" {
							ChangeStrategy()
						}
						var NewNode map[string]string
						NewNode = make(map[string]string)
						if len(ReadyAddressTable) == 0 {
							NewNode = map[string]string{
								"zombieIp": NodeAddress,
							}
							fmt.Println("Only Zombie no Ready", NodeAddress)
						}
						if len(ReadyAddressTable) > 0 {
							for _, V := range ReadyAddressTable {
								NewNode = map[string]string{
									"newIp":    V,
									"zombieIp": NodeAddress,
								}
								break
							}
							fmt.Println("Send Zombie And Ready", NodeAddress)
						}
						Data, err := json.Marshal(NewNode)
						res, err := http.Post("http://"+GetMyIP()+":7000/modifyHost", "application/json", bytes.NewBuffer(Data))
						if err != nil {
							logFile := OpenLogFile("Error")
							WriteLog(logFile, "logging,error log,system-name,MSP,errmsg,"+"Connection error")
							defer logFile.Close()
						}
						var ResFromGateway string
						json.NewDecoder(res.Body).Decode(&ResFromGateway)
						fmt.Println("ResFromGateway", ResFromGateway)
						delete(CurrentAddressTable, Node)
						ZombieAddressTable[Node] = NodeAddress
						WriteHosts()
					}
				}
			}
		}
	}
	for Node, NodeAddress := range ReadyAddressTable {
		temp := strings.Split(NodeAddress, ":")
		port, err := strconv.Atoi(temp[1])
		if Client[temp[0]+":"+fmt.Sprint(port+100)] != nil {
			err = json.NewEncoder(Client[temp[0]+":"+fmt.Sprint(port+100)]).Encode("Ready")
			var Data []byte
			if err == nil {
				_, err = Client[temp[0]+":"+fmt.Sprint(port+100)].Read(Data)
				if err != nil {
					logFile := OpenLogFile("Disconnection")
					WriteLog(logFile, "logging,disconnection log,system-name,MSP,node-name,"+NodeNameTable[NodeAddress]+",type,1")
					defer logFile.Close()
				}
			} else {
				logFile := OpenLogFile("Disconnection")
				WriteLog(logFile, "logging,disconnection log,system-name,MSP,node-name,"+NodeNameTable[NodeAddress]+",type,1")
				defer logFile.Close()
				Client[temp[0]+":"+fmt.Sprint(port+100)].Close()
				delete(ReadyAddressTable, Node)
				delete(Client, temp[0]+":"+fmt.Sprint(port+100))
			}
			Data = []byte{}
		}
	}
	for Node, NodeAddress := range ZombieAddressTable {
		temp := strings.Split(NodeAddress, ":")
		port, err := strconv.Atoi(temp[1])
		if Client[temp[0]+":"+fmt.Sprint(port+100)] != nil {
			err = json.NewEncoder(Client[temp[0]+":"+fmt.Sprint(port+100)]).Encode("Zombie")
			if err == nil {
				response, err := bufio.NewReader(Client[temp[0]+":"+fmt.Sprint(port+100)]).ReadString('\n')
				if response != "" {
					temp := response[1 : len(response)-2]
					if err != nil {
						logFile := OpenLogFile("Error")
						WriteLog(logFile, "logging,error log,system-name,MSP,errmsg,"+"Connection error")
						defer logFile.Close()
					}
					// Memory := string(Data[1 : n-2])
					fmt.Println("CPU:", temp)
					MemoryUsage, err := strconv.ParseFloat(temp, 32)
					if err != nil {
						logFile := OpenLogFile("Error")
						WriteLog(logFile, "logging,error log,system-name,MSP,errmsg,"+"Connection error")
						defer logFile.Close()
					}
					if MemoryUsage <= Revive {
						fmt.Println("Zombie to Ready")
						delete(ZombieAddressTable, Node)
						ReadyAddressTable[Node] = NodeAddress
						ScaleUp()
					}
				}
			}
		}
	}
}

//	if len(PbftHostAddressTable) > 0 {
//		for Node, NodeAddress := range PbftHostAddressTable {
//			_, err := http.Post("http://"+NodeAddress+"/PingReq", "text/plain", nil)
//			if err != nil {
//				logFile := OpenLogFile("Error")
//				WriteLog(logFile, "error,"+err.Error())
//				WriteLog(logFile, "pBFTDisconnected,"+NodeAddress+","+"type,"+"3")
//				defer logFile.Close()
//				delete(PbftHostAddressTable, Node)
//			}
//		}
//	}
//
//	for Node, NodeAddress := range PbftReadyAddressTable {
//		_, err := http.Post("http://"+NodeAddress+"/PingReq", "text/plain", nil)
//		if err != nil {
//			logFile := OpenLogFile("Error")
//			WriteLog(logFile, "error,"+err.Error())
//			WriteLog(logFile, "pBFTDisconnected,"+NodeAddress+","+"type,"+"3")
//			defer logFile.Close()
//			delete(PbftReadyAddressTable, Node)
//		}
//	}
//
// }
//
//	func SendPingRes(StatusOfAll []*Status) {
//		pid := os.Getpid()
//		p, err := procfs.NewProc(pid)
//		if err != nil {
//			logFile := OpenLogFile("Error")
//			WriteLog(logFile, "error,"+err.Error())
//			defer logFile.Close()
//		}
//		stat, err := p.Stat()
//		if err != nil {
//			logFile := OpenLogFile("Error")
//			WriteLog(logFile, "error,"+err.Error())
//			defer logFile.Close()
//		}
//		MyStatus := Status{GetMyIP() + MyPort, "MSP", stat.ResidentMemory(), stat.ResidentMemory()}
//		StatusOfAll = append(StatusOfAll, &MyStatus)
//	}
func RegNewNode(w http.ResponseWriter, req *http.Request) {
	addr := new(Addr)
	json.NewDecoder(req.Body).Decode(addr)
	logFile := OpenLogFile("NewNode")
	WriteLog(logFile, "logging,newnode log,system-name,MSP,"+"nodeName,"+addr.NodeName+",type,"+addr.Type+",address,"+addr.Address+":"+addr.NewNode)
	w.Header().Set("Content-Type", "application/json")
	ipBlackList = append(ipBlackList, "abcd")
	json.NewEncoder(w).Encode(ipBlackList)
	port, err := strconv.Atoi(addr.NewNode)
	if err != nil {
		logFile := OpenLogFile("Error")
		WriteLog(logFile, "logging,error log,system-name,MSP,errmsg,"+"Connection error")
		defer logFile.Close()
	}
	TCPConnection(addr.Address, fmt.Sprint(port+100))
	if addr.Type == "1" && Strategy == "NORMAL" {
		if len(CurrentAddressTable) <= 3 {
			fmt.Println("Current")
			CurrentAddressTable[addr.NewNode] = addr.Address + ":" + addr.NewNode
			fmt.Println(CurrentAddressTable)
			NodeNameTable[addr.Address+":"+addr.NewNode] = addr.NodeName
		} else {
			fmt.Println("Ready")
			ReadyAddressTable[addr.NewNode] = addr.Address + ":" + addr.NewNode
			NodeNameTable[addr.Address+":"+addr.NewNode] = addr.NodeName
		}
	} else if addr.Type == "2" {
		fmt.Println("Zombie")
		ZombieAddressTable[addr.NewNode] = addr.Address + ":" + addr.NewNode
		NodeNameTable[addr.Address+":"+addr.NewNode] = addr.NodeName
	} else if addr.Type == "1" && Strategy == "ABNORMAL" {
		fmt.Println("RegNewNode when Strategy is ABNORMAL")
		CurrentAddressTable[addr.NewNode] = addr.Address + ":" + addr.NewNode
		NodeNameTable[addr.Address+":"+addr.NewNode] = addr.NodeName
		Data, _ := json.Marshal(CurrentAddressTable)
		http.Post("http://"+GetMyIP()+":7000/UpdateHost", "application/json", bytes.NewBuffer(Data))
		WriteHosts()
	}
	if len(CurrentAddressTable) >= 4 && OneTime == 0 {
		Data, err := json.Marshal(CurrentAddressTable)
		if err != nil {
			logFile := OpenLogFile("Error")
			WriteLog(logFile, "logging,error log,system-name,MSP,errmsg,"+"Connection error")
			defer logFile.Close()
		}
		http.Post("http://"+GetMyIP()+":7000/UpdateHost", "application/json", bytes.NewBuffer(Data))
		WriteHosts()
		OneTime++
	}
}

//	func InitialHosts() {
//		var Hosts []string
//		for K, V := range CurrentAddressTable {
//			HostAddressTable[K] = V
//			Hosts = append(Hosts, V)
//			if len(HostAddressTable) == 3 {
//				// Data, _ := json.Marshal(Hosts)
//				// http.Post("http://localhost:7000/UpdateHost", "application/json", bytes.NewBuffer(Data))
//				break
//			}
//		}
//		WriteLog("Hosts:" + Hosts[0] + Hosts[1] + Hosts[2])
//	}
func CheckHost() {
	if len(CurrentAddressTable) >= 4 {
		var temp []string
		for _, V := range CurrentAddressTable {
			temp = append(temp, V)
		}
		var AllHostAddress string
		// logFile := OpenLogFile("Hosts")
		for i := 0; i < len(temp); i++ {
			// WriteLog(logFile, NodeNameTable[temp[i]]+","+temp[i])
			// log.Println("hosts:")
			// log.Println(temp[i] + ",")
			// defer logFile.Close()
			if i <= len(temp)-2 {
				AllHostAddress += temp[i] + "-"
			} else {
				AllHostAddress += temp[i]
			}
		}
		var AllHostName string
		for i := 0; i < len(temp); i++ {
			if i <= len(temp)-2 {
				AllHostName += NodeNameTable[temp[i]] + "-"
			} else {
				AllHostName += NodeNameTable[temp[i]]
			}
		}
		logFile := OpenLogFile("Hosts")
		WriteLog(logFile, "HostNames,"+AllHostName+","+"HostAddressTable,"+AllHostAddress)
		defer logFile.Close()
	}
}
func ChangeToNormal() {
	for K, V := range CurrentAddressTable {
		if len(ReadyAddressTable) <= 7 {
			ReadyAddressTable[K] = V
			delete(CurrentAddressTable, K)
		} else {
			Data, err := json.Marshal(CurrentAddressTable)
			if err != nil {
				logFile := OpenLogFile("Error")
				WriteLog(logFile, "logging,error log,system-name,MSP,errmsg,"+"ParseError")
				defer logFile.Close()
			}
			_, err = http.Post("http://"+GetMyIP()+":7000/UpdateHost", "application/json", bytes.NewBuffer(Data))
			if err != nil {
				logFile := OpenLogFile("Error")
				WriteLog(logFile, "logging,error log,system-name,MSP,errmsg,"+"ParseError")
				defer logFile.Close()
			}
			WriteHosts()
		}
	}

}
func ChangeStrategy() {
	if Strategy == "NORMAL" {
		Strategy = "ABNORMAL"
		ScaleUp()
	} else {
		Strategy = "NORMAL"
		ChangeToNormal()
	}
	Data, err := json.Marshal(Strategy)
	if err != nil {
		logFile := OpenLogFile("Error")
		WriteLog(logFile, "logging,error log,system-name,MSP,errmsg,"+"ParseError")
		defer logFile.Close()
	}
	for _, V := range CurrentAddressTable {
		_, err = http.Post("http://"+V+"/ChangeStrategy", "application/json", bytes.NewBuffer(Data))
		if err != nil {
			logFile := OpenLogFile("Error")
			WriteLog(logFile, "logging,error log,system-name,MSP,errmsg,"+"ConnectionError")
			defer logFile.Close()
		}
	}
	for _, V := range ReadyAddressTable {
		_, err = http.Post("http://"+V+"/ChangeStrategy", "application/json", bytes.NewBuffer(Data))
		if err != nil {
			logFile := OpenLogFile("Error")
			WriteLog(logFile, "logging,error log,system-name,MSP,errmsg,"+"ConnectionError")
			defer logFile.Close()
		}
	}
	for _, V := range ZombieAddressTable {
		_, err = http.Post("http://"+V+"/ChangeStrategy", "application/json", bytes.NewBuffer(Data))
		if err != nil {
			logFile := OpenLogFile("Error")
			WriteLog(logFile, "logging,error log,system-name,MSP,errmsg,"+"ConnectionError")
			defer logFile.Close()
		}
	}
}
func CheckZombie() {
	result := 0
	Now := time.Now()
	for time.Since(Now).Minutes() <= 5 {
		if len(ZombieAddressTable) > 0 {
			result++
		}
	}
	if result == 0 {
		fmt.Println("Change the Strategy Since there was no Zombie node for 5 mins")
		Strategy = "NORMAL"
		ChangeToNormal()
		Data, _ := json.Marshal(Strategy)
		for _, V := range CurrentAddressTable {
			_, err := http.Post("http://"+V+"/ChangeStrategy", "application/json", bytes.NewBuffer(Data))
			if err != nil {
				logFile := OpenLogFile("Error")
				WriteLog(logFile, "logging,error log,system-name,MSP,errmsg,"+"ConnectionError")
				defer logFile.Close()
			}
		}
		for _, V := range ReadyAddressTable {
			_, err := http.Post("http://"+V+"/ChangeStrategy", "application/json", bytes.NewBuffer(Data))
			if err != nil {
				logFile := OpenLogFile("Error")
				WriteLog(logFile, "logging,error log,system-name,MSP,errmsg,"+"ConnectionError")
				defer logFile.Close()
			}
		}
	}
}
func ScaleUp() {
	if len(ReadyAddressTable) > 0 {
		start := time.Now()
		for K, V := range ReadyAddressTable {
			CurrentAddressTable[K] = V
		}
		data, err := json.Marshal(CurrentAddressTable)
		if err != nil {
			logFile := OpenLogFile("Error")
			WriteLog(logFile, "logging,error log,system-name,MSP,errmsg,"+"ConnectionError")
			defer logFile.Close()
		}
		http.Post("http://"+GetMyIP()+":7000/UpdateHost", "application/json", bytes.NewBuffer(data))
		SUT = time.Since(start).String()
		for k := range ReadyAddressTable {
			delete(ReadyAddressTable, k)
		}
		WriteHosts()
	}
}

//----------------------- LazyBoy-----------------------------------
// Node -> MSP GetLazyBoy
// pbft -> MSP FinishDelay
// MSP -> Node DelayResult
// var ADT int64
// var Start time.Time

//	func GetLazyBoy(w http.ResponseWriter, r *http.Request) {
//		logFile := OpenLogFile("Performance")
//		WriteLog(logFile, "startDelay,"+time.Now().Format("2006-01-02 15:04:05"))
//		defer logFile.Close()
//		Start = time.Now()
//		if len(PbftHostAddressTable) == 0 {
//			for K, V := range PbftReadyAddressTable {
//				PbftHostAddressTable[K] = V
//				err := Awake(V)
//				logFile := OpenLogFile("Error")
//				WriteLog(logFile, "error,"+err.Error())
//				defer logFile.Close()
//				json.NewEncoder(w).Encode(V)
//				break
//			}
//		} else {
//			for _, V := range PbftHostAddressTable {
//				json.NewEncoder(w).Encode(V)
//			}
//		}
//	}
// func Awake(address string) error {
// 	_, err := http.Post("http://"+address+"/awake", "text/plain", nil)
// 	return err
// }
// func FinishDelay(w http.ResponseWriter, r *http.Request) {
// 	reqData := new(ReqData)
// 	json.NewDecoder(r.Body).Decode(&reqData)
// 	DelayResult(*reqData)
// 	ADT = time.Since(Start).Milliseconds()
// 	logFile := OpenLogFile("Performance")
// 	WriteLog(logFile, "ADT,"+fmt.Sprint(ADT))

// }
// func DelayResult(reqData ReqData) {
// 	Data, err := json.Marshal(reqData)
// 	if err != nil {
// 		logFile := OpenLogFile("Error")
// 		WriteLog(logFile, "error,"+err.Error())
// 		defer logFile.Close()
// 	}
// 	for _, V := range CurrentAddressTable {
// 		_, err = http.Post("http://"+V+"/DelayResult", "application/json", bytes.NewBuffer(Data))
// 		logFile := OpenLogFile("Error")
// 		WriteLog(logFile, "error,"+err.Error())
// 		defer logFile.Close()
// 	}
// }
//	func CheckLazyBoyHost() {
//		if len(PbftHostAddressTable) > 0 {
//			for _, V := range PbftHostAddressTable {
//				logFile := OpenLogFile("Hosts")
//				WriteLog(logFile, "lazyBoyHost,"+V)
//				defer logFile.Close()
//			}
//		}
//	}
//
//	func Rest(w http.ResponseWriter, r *http.Request) {
//		for Node := range PbftHostAddressTable {
//			for K, V := range PbftReadyAddressTable {
//				PbftHostAddressTable[K] = V
//			}
//			delete(PbftHostAddressTable, Node)
//		}
//	}
