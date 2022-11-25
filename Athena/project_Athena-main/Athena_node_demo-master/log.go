package main

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

// Open or make directory and file for log
func OpenLogFile(fileName string) *os.File {
	// log file name is current day
	date := time.Now().Format("2006-01-02")
	logFolderPath := "./log/" + InitValue.NodeName
	logFilePath := fmt.Sprintf("%s/%s-%s.txt", logFolderPath, date, fileName)
	if _, err := os.Stat(logFolderPath); os.IsNotExist(err) {

		os.MkdirAll(logFolderPath, 0777)
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
		WriteLog(logFile, ",")
	} else {
		WriteLog(logFile, "[")
	}

	deleteLine(logFilePath, "]")
	return logFile
}

// Write log
func WriteLog(logFile *os.File, logData string) {
	//log.SetOutput(logFile)

	logger := log.New(logFile, "", 0)
	// logger := log.New(logFile, "", log.Ldate|log.Ltime)

	if logData != "," && logData != "[" {
		strs := strings.Split(logData, ",")
		date := time.Now().Format("2006-01-02 15:04:05")
		logger.Println("{\n\"timestamp\":\"" + date + "\",")
		for i := 0; i < len(strs); i += 2 {
			if i+2 >= len(strs) {
				if strs[i+1] == "null" {
					logger.Print("\"" + strs[i] + "\":null")
				} else {
					logger.Print("\"" + strs[i] + "\":\"" + strs[i+1] + "\"")
				}
			} else {
				if strs[i+1] == "null" {
					logger.Print("\"" + strs[i] + "\":null,")
				} else {
					logger.Print("\"" + strs[i] + "\":\"" + strs[i+1] + "\",")
				}
			}
		}
		logger.Println("}\n]")
	} else {
		logger.Println(logData)
	}
}

func deleteLine(path string, line string) {
	fpath := path

	f, err := os.Open(fpath)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	var bs []byte
	buf := bytes.NewBuffer(bs)

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if scanner.Text() != line {
			_, err := buf.Write(scanner.Bytes())
			if err != nil {
				log.Fatal(err)
			}
			_, err = buf.WriteString("\n")
			if err != nil {
				log.Fatal(err)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	err = os.WriteFile(fpath, buf.Bytes(), 0666)
	if err != nil {
		log.Fatal(err)
	}
}
