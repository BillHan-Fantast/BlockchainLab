package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strconv"
)

func main() {
	data := make(map[string]string)
	for i := 0; i < 2; i++ {
		data["node"] = "http://127.0.0.1:800" + strconv.Itoa(i)
		bytesData, _ := json.Marshal(data)
		for j := 0; j < 2; j++ {
			url := "http://127.0.0.1:800" + strconv.Itoa(j) + "/nodes/register"
			http.Post(url, "", bytes.NewReader(bytesData))
		}
	}
}
