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
	"fmt"
	"github.com/BurntSushi/toml"
	Logger "github.com/alecthomas/log4go"
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
	"encoding/json"
	"strconv"
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



	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {

		Logger.Exit(err)
	}

	Logger.LoadConfiguration(dir+"/log4go.xml")

	if _, err := toml.DecodeFile(dir+"/config.toml", &config.GlbCfg); err != nil {
		Logger.Exit(err)
	}

	MonitorPrc = config.GlbCfg.Monitor.PrcName
	userLimit = config.GlbCfg.Monitor.Users
	monitorDP = time.Duration(int(time.Second) / config.GlbCfg.Monitor.Frames)


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

func printInitInfo()  {
	buf, err := json.Marshal(config.GlbCfg)
	if err != nil {
		Logger.Exit(err)
	}

	scanproc.SelfPid = strconv.Itoa(os.Getpid())
	scanproc.ProcMap = make(map[string][]string)
	Logger.Info("toml config info-->", string(buf))
	Logger.Info("SelfPid------------------------------->", scanproc.SelfPid)
	Logger.Info("init success")
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

	printInitInfo()

	vouchb := lookForVoucher(config.GlbCfg.ProtectId)
	if !vouchb {
		fmt.Println("vouch not found,pid-->%s", config.GlbCfg.ProtectId)
		gService.Stop()
	}

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

	status,err := service.Manage()

	if err != nil {
		fmt.Printf("error:%v\n",err)
		os.Exit(1)
	}

	fmt.Printf("run as service status-->%s\n", status)
}

func lookForVoucher(pidstr string) bool {
	//pidstr := "32439"
	cmdstr := fmt.Sprintf("ps -p %s", pidstr)
	killbuf, err := exec.Command("/bin/sh", "-c", cmdstr).CombinedOutput()
	if err != nil {
		Logger.Info(err)
		return false
	}
	killResult := strings.TrimSpace(string(killbuf))
	for _, line := range strings.Split(killResult, "\n")[:] {
		fields := strings.Fields(line)
		fieldsMinLen := 1
		lens := len(fields)
		if lens < fieldsMinLen {
			continue
		}

		temp := fields[0]

		if pidstr == temp {
			return true
		}
	}

	Logger.Info(killResult)
	return false
}
