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

package pfctlmgr

import (
	Logger "log"
	"os"
	"os/exec"
	"path/filepath"
)



func InitPfctl() {
	pfconfPath := "./pf.conf"
	_, err := os.Open(pfconfPath)
	if err != nil {
		Logger.Print("init pf.conf failed",err)
	}

	output, err := exec.Command("/bin/sh", "-c", "pfctl -e").CombinedOutput()
	rt := string(output)
	if err != nil {
		Logger.Printf("progme:pfctl start failed:%s", rt)
	} else {
		Logger.Printf("progme:pfctl start success:%s", rt)
	}


	execPath, err := exec.LookPath(os.Args[0])
	if err != nil {
		Logger.Println("get exec path error:",err)
	}
	Logger.Println("exec path :",execPath)
	execDir := filepath.Dir(execPath)
	if execDir == "." {
		execDir, err = os.Getwd()
		if err != nil {
			Logger.Println("get file path error:",err)
		}
	}

	Logger.Println("execpath~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~:",execPath)


	output, err = exec.Command("/bin/sh", "-c","pfctl -f "+execDir+"/pf.conf").CombinedOutput()
	if err != nil {
		Logger.Printf("progme:pfctl reload rules failed:%s" , string(output))
	} else {
		Logger.Printf("progme:pfctl reload rules success:%s" , string(output))
	}




}
