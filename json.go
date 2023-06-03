package main

import (
	"encoding/json"
	"os"
)

type config struct {
	M *mss
	C *customers
}

func loader() {
	cus := customers{}
	conf := config{&dic, &cus}
	bytes, err := os.ReadFile(tmbPingJson)
	if err != nil {
		stdo.Println(err)
		return
	}
	err = json.Unmarshal(bytes, &conf)
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
	defer wg.Done()
	cus := customers{}
	conf := config{&dic, &cus}
	for {
		select {
		case <-saveDone:
			stdo.Println(cus)
			bytes, err := json.Marshal(conf)
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
