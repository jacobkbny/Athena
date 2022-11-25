package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type Data struct {
	Data string `json:"data"`
}

func SwtichStrategy(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Switch Called")
	data := new(Data)
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		fmt.Println(err)
	}
	if data.Data == "0" {
		Strategy = "NORMAL"
		fmt.Println("Strategy", Strategy)
		ChangeToNormal()
	}
	if data.Data == "1" {
		ChangeStrategy()
		ScaleUp()
		fmt.Println("ChangeStrategy")
	}
}
