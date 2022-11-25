package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"log"
	"os"
	"sort"
)

var Hash [32]byte

// Load config.json file
func LoadConfig() Config {
	file, err := os.Open("config.json")
	defer file.Close()
	if err != nil {
		log.Println(err)
	}

	var config Config
	json.NewDecoder(file).Decode(&config)

	return config
}

// Check if config file is modified by Hash
func checkConfig(config Config) bool {
	tmpHash := MakeHashOfConfig(config)

	if tmpHash == Hash {
		return true
	}
	return false
}

// Create data of config file to make hash
func prepareData(config Config) []byte {
	var tmp [][]byte
	for k, v := range config.URL {
		tmp = append(tmp, []byte(k))
		tmp = append(tmp, []byte(v))
	}
	tmp = append(tmp, []byte(config.MspPort))
	tmp = append(tmp, []byte(config.Public))

	data := bytes.Join(tmp, []byte{})
	sort.Slice(data, func(i, j int) bool {
		return data[i] < data[j]
	})

	return data
}

// Create hash of config file
func MakeHashOfConfig(config Config) [32]byte {
	data := prepareData(config)
	hash := sha256.Sum256(data)

	return hash
}
