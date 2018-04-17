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
	"io"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"github.com/boxproject/boxguard/config"
)

var (
	dot     = []byte{46}
	procMap map[string][]string
	SelfPid string
)


func init() {
	SelfPid = strconv.Itoa(os.Getpid())
	log.Println("SelfPid------------------------------->", SelfPid)
	procMap = make(map[string][]string)
	log.Println("init success")
}

func writeToFile(execPath,pid string,){
	//f,err := os.OpenFile("./build/bin/proclog_snap",os.O_WRONLY|os.O_APPEND|os.O_CREATE,0666)
	f,err := os.OpenFile("./build/bin/proclog_snap",os.O_WRONLY|os.O_APPEND|os.O_CREATE,0666)
	if err != nil {
		log.Println("err-->",err)
	}
	defer f.Close()
	f.WriteString(pid)
	f.WriteString("\t")
	f.WriteString(execPath)
	f.WriteString("\n")
}


func GetProcessList(init bool) {
	if !config.GlbCfg.EnableProcGuard {
		return
	}
	cmd := exec.Command("ps", "-A")
	out, err := cmd.StdoutPipe()
	if err != nil {
		log.Printf("STD out pipe failed. cause: %v\n", err)
		return
	}

	if err := cmd.Start(); err != nil {
		log.Printf("run system command failed. cause: %v\n", err)
		return
	}

	reader := bufio.NewReader(out)
	reg, err := regexp.Compile("(\\d+).*?(\\.?/.*?)$")
	if err != nil {
		log.Printf("%v\n", err)
		return
	}

	if _, _, err = reader.ReadLine(); err != nil {
		return
	}

	for {
		line, _, err := reader.ReadLine()
		if err != nil {
			if io.EOF == err {
				fmt.Println("current proc count:", len(procMap))
				return
			}
			log.Printf("read line failed. cause: %v\n", err)
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
							continue
						}

						fields := strings.Fields(fpstr)
						lenth := len(fields)
						if lenth < 9 {
							continue
						}
						pidstr = fields[1]
						exePathStr = fields[8]
					}

					if init {
						log.Println("===============init proc-------->", exePathStr)
					}

					//check if the proc already in memory
					subPids, ok := procMap[exePathStr]

					//add it to memory
					subPids = append(subPids, pidstr)

					if !ok {
						if init {
							procMap[exePathStr] = subPids
						} else {
							doKill(exePathStr, pidstr)
						}
					}
				}
			}
		}else{
			fpstr := string(line)
			fields := strings.Fields(fpstr)
			lenth := len(fields)

			pidstr:= fields[0]

			logstr := fields[lenth-1]

			log.Printf("no match --> %s,log=%s",pidstr,logstr)

			doKill(logstr,pidstr)
		}

	}
}


func doKill(exePathStr string, pidstr string) {

	if SelfPid != pidstr && !config.GlbCfg.InWhite(exePathStr) {
		log.Println("====do kill==========for not in memory--------》", exePathStr)
		cmdstr := "kill -9 " + pidstr
		killbuf, err := exec.Command("/bin/sh", "-c", cmdstr).CombinedOutput()
		killResult := string(killbuf)
		writeToFile(exePathStr,pidstr)
		if err != nil {
			log.Println("kill process ", exePathStr, " failed,pid=", pidstr, ",exec out:", killResult)
		}
		log.Println("kill process ", exePathStr, " success,pid=", pidstr, ",exec out:", killResult)
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
			log.Printf("read line failed. cause: %v\n", err)
			return "", err
		}
		return string(line), nil
	}

}
