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
	"strconv"
	"strings"
	"syscall"
	"time"
	Logger "github.com/alecthomas/log4go"
)

type Service struct {
	go_service.Daemon
}

var (
	MonitorPrc  = "voucher"
	ServiceName = "boxgd"
	Description = "boxgd-Service"
	userLimit   = -1
	monitorDP   = 1000 * time.Millisecond
	gService    Service
)

func init() {

	Logger.LoadConfiguration("./log4go.xml")

	//defer Logger.Close()

	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {

		Logger.Exit(err)
	}

	if _, err := toml.DecodeFile(dir+"/config.toml", &config.GlbCfg); err != nil {
		Logger.Exit(err)
	}

	MonitorPrc = config.GlbCfg.Monitor.PrcName
	userLimit = config.GlbCfg.Monitor.Users
	monitorDP = time.Duration(int(time.Second) / config.GlbCfg.Monitor.Frames)

	buf, err := json.Marshal(config.GlbCfg)
	if err != nil {
		Logger.Exit(err)
	}
	Logger.Info("toml config info-->", string(buf))

	scanproc.SelfPid = strconv.Itoa(os.Getpid())
	Logger.Info("SelfPid------------------------------->", scanproc.SelfPid)
	scanproc.ProcMap = make(map[string][]string)
	Logger.Info("init success")
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
		count++
	}
	return count
}

//kill progme
func killPro() {
	output, err := exec.Command("killall", "-SIGINT", MonitorPrc).CombinedOutput()
	if err != nil {
		Logger.Info("progme:voucher killed failed:" + string(output))
	} else {
		Logger.Info("progme:voucher killed succeed:" + string(output))
	}
}

//if you want to run this program as a service,you can call this method before call Manage
func runAsService(service *Service) (string, error) {
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

	return usage, nil
}

// Manage by daemon commands or run the daemon
func (service *Service) Manage() {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
	Logger.Info("----after %ds monitor progme will work-----\n", config.GlbCfg.WaitSeconds, ",pid=", os.Getpid())
	config.GlbCfg.InitData()

	if config.GlbCfg.EnablePfctl {
		pfctlmgr.InitPfctl()
	}

	for count := 1; count <= config.GlbCfg.WaitSeconds; count++ {
		time.Sleep(time.Second)
		Logger.Info("----%ds----\n", count)
	}

	scanproc.GetProcessList(true)

	Logger.Info("----monitor progme start-----")

	go func() {
		timerListen := time.NewTicker(monitorDP)
		for {
			select {
			case <-timerListen.C:
				go scanproc.GetProcessList(false)
			}
		}
	}()

	go func() {
		timerListen2 := time.NewTicker(monitorDP)
		for {
			select {
			case <-timerListen2.C:
				if userLimit >= 0 {
					userCount := getUserStat()
					Logger.Info("cur valid usr cnt -->", userCount)
					if userCount > userLimit {
						Logger.Info("----%d users login,begin to kill progme----\n", userCount)
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
			Logger.Info("Got signal:", killSignal)
			if killSignal == syscall.SIGINT {
				Logger.Info("interrupted by system signal,%s  was killed\n", ServiceName)
				os.Exit(0)
			}
			Logger.Info("%s  was killed\n", ServiceName)
			os.Exit(0)
		}
	}
}

func main() {

	srv, err := go_service.New(ServiceName, Description, []string{""}...)
	if err != nil {
		Logger.Info("Error: ", err)
		os.Exit(1)
	}

	service := &Service{srv}
	gService = *service
	service.Manage()

}
