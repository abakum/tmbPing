package main

import (
	"encoding/json"
	"os"
)

type config struct {
	M *mss
	C *customers
}

func loader() (err error) {
	cus := customers{}
	conf := config{&dic, &cus}
	bytes, err := os.ReadFile(tmbPingJson)
	if err != nil {
		let.Println(src(8), err)
		return
	}
	err = json.Unmarshal(bytes, &conf)
	if err != nil {
		let.Println(src(8), err)
		return
	}
	for _, cu := range cus {
		ltf.Println(cu)
		ips.write(cu.Cmd, cu)
	}
	return
}

func saver() {
	defer wg.Done()
	cus := customers{}
	conf := config{&dic, &cus}
	for {
		select {
		case <-saveDone:
			ltf.Println(cus)
			bytes, err := json.Marshal(conf)
			if err != nil {
				let.Println(src(8), err)
				return
			}
			err = os.WriteFile(tmbPingJson, bytes, 0644)
			if err != nil {
				let.Println(src(8), err)
				return
			}
			ltf.Println("saver done")
			return
		case cu, ok := <-save:
			if ok {
				cus = append(cus, cu)
				// bytes, _ := json.Marshal(cu)
				// stdo.Printf("%s\n", bytes)
			} else {
				ltf.Println("saver channel closed")
				saveDone <- true
			}
		}
	}
}
