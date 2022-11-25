package main

import (
	"log"
	"os"
	"runtime/debug"
	"strconv"
	"time"
)

// Channels is for log. Prevent async logging
var statusQue chan string
var performanceQue chan string
var warningQue chan string

func main() {
	// Fisrt argument is for node's group
	arg_first := os.Args[1]
	// Second argument is for node's port
	arg_second := os.Args[2]
	// Third argument is for node's name
	arg_third := os.Args[3]

	// Check msgQue channel and write log
	go func() {
		statusQue = make(chan string, 100)
		performanceQue = make(chan string, 100)
		warningQue = make(chan string, 100)
		for {
			select {
			case msg := <-statusQue:
				logFile := OpenLogFile(InitValue.NodeName + "-Status")
				defer logFile.Close()
				WriteLog(logFile, msg)
			case msg := <-performanceQue:
				logFile := OpenLogFile(InitValue.NodeName + "-Performance")
				defer logFile.Close()
				WriteLog(logFile, msg)
			case msg := <-warningQue:
				logFile := OpenLogFile(InitValue.NodeName + "-Warning")
				defer logFile.Close()
				WriteLog(logFile, msg)
			}
		}
	}()

	// Check if config file is modified with GoRoutine
	go func() {
		check := true
		for {
			conf := LoadConfig()
			check = checkConfig(conf)
			if !check {
				log.Println("Config file was modified!!")
				ConfigData = conf
				Hash = MakeHashOfConfig(conf)
			}
			time.Sleep(time.Millisecond * 3000)
		}
	}()

	// Clean node's buffer when node is zombie
	go func() {
		for {
			if InitValue.Group == "Zombie" {
				cpu, _, _ := GetMemoryUsage()
				cpuPer, err := strconv.ParseFloat(cpu, 32)
				if err != nil {
					log.Println("Cpu parsing error", err)
				}
				if cpuPer >= 20 {
					log.Println("Free Memory")
					debug.FreeOSMemory()
				} else {
					time.Sleep(time.Millisecond * 1000)
				}
			} else {
				time.Sleep(time.Millisecond * 3000)
			}
		}
	}()

	go TcpStart(arg_second)
	NewServer(arg_second, arg_first, arg_third)
	ServerStart(arg_second)
}
