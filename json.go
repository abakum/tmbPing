package main

import (
	"encoding/json"
	"os"
)

func loader() {
	cus := customers{}
	bytes, err := os.ReadFile(tmbPingJson)
	if err != nil {
		stdo.Println(err)
		return
	}
	err = json.Unmarshal(bytes, &cus)
	if err != nil {
		stdo.Println(err)
		return
	}
	for _, cu := range cus {
		stdo.Println(cu)
		ips.write(cu.Cmd, cu)
	}
}

func saver() {
	save = make(cCustomer, 1)
	saveDone = make(chan bool, 1)
	cus := customers{}
	for {
		select {
		case <-saveDone:
			stdo.Println(cus)
			bytes, err := json.Marshal(cus)
			if err != nil {
				stdo.Println(err)
				return
			}
			err = os.WriteFile(tmbPingJson, bytes, 0644)
			if err != nil {
				stdo.Println(err)
				return
			}
			stdo.Println("saver done")
			return
		case cu, ok := <-save:
			if ok {
				cus = append(cus, cu)
				// bytes, _ := json.Marshal(cu)
				// stdo.Printf("%s\n", bytes)
			} else {
				stdo.Println("saver channel closed")
				saveDone <- true
			}
		}
	}
}
