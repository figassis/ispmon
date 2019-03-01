package main

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/figassis/ispmon/util"
)

func main() {
	errs := make(chan error, 100)
	var Wg sync.WaitGroup

	if err := util.LoadConfig(); err != nil {
		fmt.Println(err.Error())
		return
	}

	Wg.Add(1)
	go util.Run(&Wg, errs)
	for {
		select {
		case err := <-errs:
			util.Log(3, err)
		}
		time.Sleep(5 * time.Minute)
	}
	Wg.Wait()
	util.Log(1, fmt.Sprint("Done. Exiting"))
	os.Exit(0)
}
