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

package scanproc

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/boxproject/boxguard/config"
	"io"
	Logger "log"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

var (
	dot               = []byte{46}
	ProcMap map[string][]string
	SelfPid string
)

func writeToFile(execPath, pid string) {
	f, err := os.OpenFile("./proclog_snap", os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		fmt.Println(err)
	}
	defer f.Close()

	f.WriteString(time.Now().Format("2006-01-02 15:04:05.999999999"))
	f.WriteString("\t\tpid=")
	f.WriteString(pid)
	f.WriteString("\t")
	f.WriteString(execPath)
	f.WriteString("\n")
}

func GetProcessList(init bool) {
	if !config.GlbCfg.EnableProcGuard {
		return
	}

	if init {
		Logger.Println("get process list begin,please wait for a moment")
	}else{
		Logger.Println("get process list again••••••••••••••")
	}


	timebg := time.Now().UnixNano()

	cmd := exec.Command("/bin/sh", "-c", "ps -A")

	result, err := cmd.Output()

	buf := bytes.NewBuffer(result)

	readerout := bufio.NewReader(buf)

	reg, err := regexp.Compile("(\\d+).*?(\\.?/.*?)$")
	if err != nil {
		Logger.Printf("%v\n", err)
		return
	}

	if _, _, err = readerout.ReadLine(); err != nil {
		Logger.Printf("read buffer failed. cause:%v\n",err)
		return
	}

	for {
		line, _, err := readerout.ReadLine()
		if err != nil {
			if io.EOF == err {
				Logger.Printf("current proc count:%d", len(ProcMap))
				return
			}
			Logger.Printf("read line failed. cause: %v\n", err)
			return
		}

		matches := reg.FindAllSubmatch(line, -1)

		if len(matches) > 0 {
			for _, test := range matches {
				if len(test) == reg.NumSubexp()+1 {
					pid := test[1]
					pidstr := string(pid)
					exePath := test[2]
					exePathStr := string(exePath)
					if bytes.Index(exePath, dot) == 0 {

						fpstr, err := GetFullPath(string(pid))
						if err != nil {
							Logger.Printf("error occured when get full path,%v\n",err)
							continue
						}

						fields := strings.Fields(fpstr)
						lenth := len(fields)
						if lenth < 9 {
							Logger.Printf("fpstr lenth less than 9,what is:%v\n",fpstr)
							continue
						}
						pidstr = fields[1]
						exePathStr = fields[8]
					}

					if init {
						Logger.Println("===============init proc-------->", exePathStr)
					}

					//check if the proc already in memory,if not in kill
					toggle(exePathStr, pidstr, init)
				}
			}
		} else {
			fpstr := string(line)
			fields := strings.Fields(fpstr)
			lenth := len(fields)

			pidstr := fields[0]

			logstr := fields[lenth-1]

			toggle(logstr,pidstr,init)
		}

	}

	timett := time.Now().UnixNano()-timebg

	Logger.Printf("get process list end, cost  unix nano mills--> %v",timett)

}

func doKill(exePathStr string, pidstr string) {

	if SelfPid != pidstr && !config.GlbCfg.InWhite(exePathStr) {
		Logger.Println("====do kill==========for not in memory--------》", exePathStr)
		cmdstr := "kill -9 " + pidstr
		killbuf, err := exec.Command("/bin/sh", "-c", cmdstr).CombinedOutput()

		killResult := string(killbuf)
		writeToFile(exePathStr, pidstr)
		if err != nil {
			Logger.Println("kill process ", exePathStr, " failed,pid=", pidstr, ",exec out:", killResult)
		}
		Logger.Println("kill process ", exePathStr, " success,pid=", pidstr, ",exec out:", killResult)
	}
}

func GetFullPath(pid string) (string, error) {
	lsofcmd := "lsof -p " + pid + " | head -n 3 | tail -n 1"
	cmd := exec.Command("/bin/sh", "-c", lsofcmd)

	result, err := cmd.Output()
	if err != nil {
		return "", err
	}

	buf := bytes.NewBuffer(result)

	reader := bufio.NewReader(buf)

	for {
		line, _, err := reader.ReadLine()
		if err != nil {
			if io.EOF == err {
				return "", err
			}
			Logger.Printf("read line failed. cause: %v\n", err)
			return "", err
		}
		return string(line), nil
	}

}

func toggle(exePathStr string, pidstr string, init bool) {
	subPids, ok := ProcMap[exePathStr]
	//add it to memory
	subPids = append(subPids, pidstr)
	if !ok {
		if init {
			ProcMap[exePathStr] = subPids
		} else {
			doKill(exePathStr, pidstr)
		}
	}
}