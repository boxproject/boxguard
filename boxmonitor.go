// Copyright 2018 The box.la Authors All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"encoding/json"
	"github.com/BurntSushi/toml"
	"github.com/boxproject/boxguard/config"
	"github.com/boxproject/boxguard/pfctlmgr"
	"github.com/boxproject/boxguard/scanproc"
	go_service "github.com/takama/daemon"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"strconv"
	Logger "log"
)

type Service struct {
	go_service.Daemon
}

var (
	MonitorPrc    = "voucher"
	ServiceName  = "boxgd"
	Description  = "boxgd-Service"
	userLimit      = -1
	monitorDP      = 666 * time.Millisecond
	gService    Service
)

func init() {
	


	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		Logger.Fatal(err)
	}

	if _, err := toml.DecodeFile(dir+"/config.toml", &config.GlbCfg); err != nil {
		Logger.Fatal(err)
	}

	MonitorPrc = config.GlbCfg.Monitor.PrcName
	userLimit = config.GlbCfg.Monitor.Users
	monitorDP = time.Duration(int(time.Second) / config.GlbCfg.Monitor.Frames)


	buf, err := json.Marshal(config.GlbCfg)
	if err != nil {
		Logger.Fatal(err)
	}
	Logger.Println("toml config info-->", string(buf))

	scanproc.SelfPid = strconv.Itoa(os.Getpid())
	Logger.Println("SelfPid------------------------------->", scanproc.SelfPid)
	scanproc.ProcMap = make(map[string][]string)
	Logger.Println("init success")
}

//get current login user count
func getUserStat() (num int) {
	count := 0
	output, _ := exec.Command("who").Output()
	for _, line := range strings.Split(string(output), "\n")[:] {
		fields := strings.Fields(line)
		fieldsMinLen := 5
		lens := len(fields)
		if lens < fieldsMinLen {
			continue
		}

		if config.GlbCfg.AllowUser==fields[0] {
			Logger.Printf("%s_%s is working",fields[0],fields[1])
			continue
		}

		count++
	}
	return count
}

//kill progme
func killPro() {
	output, err := exec.Command("killall", "-SIGINT", MonitorPrc).CombinedOutput()
	if err != nil {
		Logger.Printf("progme:voucher killed failed:" + string(output))
	} else {
		Logger.Printf("progme:voucher killed succeed:" + string(output))
	}
}

// Manage by daemon commands or run the daemon
func (service *Service) Manage() (string, error) {

	usage := "Usage: myservice install | remove | start | stop | status"
	// if received any kind of command, do it
	if len(os.Args) > 1 {
		command := os.Args[1]
		switch command {
		case "install":
			return service.Install()
		case "remove":
			return service.Remove()
		case "start":
			return service.Start()
		case "stop":
			return service.Stop()
		case "status":
			return service.Status()
		default:
			return usage, nil
		}
	}

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
	Logger.Printf("----after %ds monitor progme will work-----\n", config.GlbCfg.WaitSeconds, ",pid=", os.Getpid())
	config.GlbCfg.InitData()

	if config.GlbCfg.EnablePfctl {
		pfctlmgr.InitPfctl()
	}

	for count := 1; count <= config.GlbCfg.WaitSeconds; count++ {
		time.Sleep(time.Second)
		Logger.Printf("----%ds----\n", count)
	}

	scanproc.GetProcessList(true)

	Logger.Println("----monitor progme start-----")

	go func() {
		for {
			timerListen := time.NewTicker(monitorDP)
			select {
			case <-timerListen.C:
				scanproc.GetProcessList(false)
			}
		}

	}()

	go func() {
		for {
			timerListen := time.NewTicker(monitorDP)
			select {
			case <-timerListen.C:
				if userLimit >= 0 {
					userCount := getUserStat()
					Logger.Println("cur valid usr cnt -->", userCount)
					if userCount > userLimit {
						Logger.Printf("----%d users login,begin to kill progme----\n", userCount)
						//kill progme
						killPro()
						gService.Stop()
					}
				}
			}
		}

	}()



	// loop work cycle with accept interrupt by system signal
	for {
		select {
		case killSignal := <-interrupt:
			Logger.Println("Got signal:", killSignal)
			if killSignal == syscall.SIGINT {
				Logger.Printf("interrupted by system signal,%s  was killed\n", ServiceName)
				os.Exit(0)
			}
		}
	}
}

func main() {

	srv, err := go_service.New(ServiceName, Description, []string{""}...)
	if err != nil {
		Logger.Println("Error: ", err)
		os.Exit(1)
	}

	service := &Service{srv}
	gService = *service
	status, err := service.Manage()
	if err != nil {
		Logger.Println(status, "\nError: ", err)
		os.Exit(1)
	}
	Logger.Println(status)
}
