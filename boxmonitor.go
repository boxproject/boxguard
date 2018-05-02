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
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

type Service struct {
	go_service.Daemon
}

var (
	MonitorPrc  = "voucher"
	ServiceName = "boxgd"
	Description = "boxgd-Service"
	stdlog      *log.Logger
	userLimit   = -1
	monitorDP   = 100 * time.Microsecond
	gService    Service
)

func init() {

	stdlog = log.New(os.Stdout, "", log.Ldate|log.Ltime)

	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
	}

	if _, err := toml.DecodeFile(dir+"/config.toml", &config.GlbCfg); err != nil {
		//return err
		log.Fatal(err)
	}
	MonitorPrc = config.GlbCfg.Monitor.PrcName
	userLimit = config.GlbCfg.Monitor.Users
	if config.GlbCfg.Monitor.Frames <= 0 {
		monitorDP = time.Second / 10
	} else if config.GlbCfg.Monitor.Frames > 1000 {
		monitorDP = time.Second / 1000
	} else {
		monitorDP = time.Duration(int(time.Second) / config.GlbCfg.Monitor.Frames)
	}

	buf, err := json.Marshal(config.GlbCfg)
	if err != nil {
		log.Fatal(err)
	}
	stdlog.Println("toml config info-->", string(buf))


	config.GlbCfg.InitData()

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
		stdlog.Printf("progme:voucher killed failed:" + string(output))
	} else {
		stdlog.Printf("progme:voucher killed succeed:" + string(output))
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
	stdlog.Printf("----after %ds monitor progme will work-----\n", config.GlbCfg.WaitSeconds, ",pid=", os.Getpid())
	for count := 1; count <= config.GlbCfg.WaitSeconds; count++ {
		time.Sleep(time.Second)
		stdlog.Printf("----%ds----\n", count)
	}
	stdlog.Println("----monitor progme start-----")

	if config.GlbCfg.EnablePfctl {
		pfctlmgr.InitPfctl()
	}

	go func() {
		for {
			scanproc.GetProcessList(false)
			if userLimit >= 0 {
				userCount := getUserStat()
				stdlog.Println("cur usr cnt -->", userCount)
				if userCount > userLimit {
					stdlog.Printf("----%d users login,begin to kill progme----\n", userCount)
					//kill progme
					killPro()
					gService.Stop()
				}
			}
			//stdlog.Println("----work-----")
			time.Sleep(monitorDP)
		}
	}()

	// loop work cycle with accept interrupt by system signal
	for {
		select {
		case killSignal := <-interrupt:
			stdlog.Println("Got signal:", killSignal)
			if killSignal == syscall.SIGINT {
				stdlog.Println("interrupted by system signal")
				stdlog.Printf("%s  was killed\n", ServiceName)
				os.Exit(0)
			}
			stdlog.Printf("%s  was killed\n", ServiceName)
			os.Exit(0)
		}
	}
}

func main() {
	scanproc.GetProcessList(true)
	srv, err := go_service.New(ServiceName, Description, []string{""}...)
	if err != nil {
		stdlog.Println("Error: ", err)
		os.Exit(1)
	}

	service := &Service{srv}
	gService = *service
	status, err := service.Manage()
	if err != nil {
		stdlog.Println(status, "\nError: ", err)
		os.Exit(1)
	}
	stdlog.Println(status)
}
