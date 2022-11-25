package main

import (
	"net"
)

var Client map[string]net.Conn

//	func TCP() {
//		Client = make(map[string]net.Conn)
//		for K, V := range CurrentAddressTable {
//			Port, err := strconv.Atoi(K)
//			if err != nil {
//				logFile := OpenLogFile("Error")
//				WriteLog(logFile, "error,"+err.Error())
//			}
//			address := strings.Split(V, ":")
//			client, err := net.Dial("tcp", address[0]+fmt.Sprint(Port+100))
//			Client = append(Client, client)
//			data := make([]byte, 4096)
//			Group := []byte("Current")
//			client.Write(Group)
//			n, err := client.Read(data)
//			if err != nil {
//				if len(ReadyAddressTable) > 0 {
//					for K, V := range ReadyAddressTable {
//						client, err := net.Dial("tcp", GetMyIP()+":7000")
//						defer client.Close()
//						if err != nil {
//							logFile := OpenLogFile("Error")
//							WriteLog(logFile, "error,"+err.Error())
//							defer logFile.Close()
//							client.Write([]byte(V))
//							delete(CurrentAddressTable, K)
//						}
//					}
//				}
//				logFile := OpenLogFile("Error")
//				WriteLog(logFile, "error,"+err.Error())
//				defer logFile.Close()
//			}
//			fmt.Println(n)
//			defer client.Close()
//		}
//		for K, V := range ReadyAddressTable {
//			Port, err := strconv.Atoi(K)
//			if err != nil {
//				logFile := OpenLogFile("Error")
//				WriteLog(logFile, "error,"+err.Error())
//			}
//			address := strings.Split(V, ":")
//			client, err := net.Dial("tcp", address[0]+fmt.Sprint(Port+100))
//			Client = append(Client, client)
//			data := make([]byte, 4096)
//			Group := []byte("Ready")
//			client.Write(Group)
//			n, err := client.Read(data)
//			if err != nil {
//				logFile := OpenLogFile("Error")
//				WriteLog(logFile, "error,"+err.Error())
//				defer logFile.Close()
//			}
//			fmt.Println(n)
//			defer client.Close()
//		}
//		for K, V := range ZombieAddressTable {
//			Port, err := strconv.Atoi(K)
//			if err != nil {
//				logFile := OpenLogFile("Error")
//				WriteLog(logFile, "error,"+err.Error())
//			}
//			address := strings.Split(V, ":")
//			client, err := net.Dial("tcp", address[0]+fmt.Sprint(Port+100))
//			Client = append(Client, client)
//			data := make([]byte, 4096)
//			Group := []byte("Zombie")
//			client.Write(Group)
//			n, err := client.Read(data)
//			if err != nil {
//				logFile := OpenLogFile("Error")
//				WriteLog(logFile, "error,"+err.Error())
//				defer logFile.Close()
//			}
//			fmt.Println(n)
//			defer client.Close()
//		}
//	}
//
// put address and port for Connection
func TCPConnection(address string, port string) {
	client, err := net.Dial("tcp", address+":"+port)
	if err != nil {
		logFile := OpenLogFile("Error")
		WriteLog(logFile, "error,"+err.Error())
		defer logFile.Close()
	}
	Client[address+":"+port] = client
}

// 4000,, 4001 for differenciation I have to make Client is like Client map[string]net.Conn
// I made them as a map so I have to test it
func CloseConnection(Client map[string]net.Conn) {
	for _, V := range Client {
		V.Close()
	}
}
