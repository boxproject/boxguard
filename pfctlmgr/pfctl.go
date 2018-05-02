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
	"log"
	"os"
	"os/exec"
)

var stdlog *log.Logger

func InitPfctl() {
	stdlog = log.New(os.Stdout, "", log.Ldate|log.Ltime)
	pfconfPath := "./pf.conf"
	_, err := os.Open(pfconfPath)
	if err != nil {
		log.Fatal(err)
	}

	output, err := exec.Command("/bin/sh", "-c", "pfctl -e").CombinedOutput()
	rt := string(output)
	if err != nil {
		stdlog.Printf("progme:pfctl start failed:%s", rt)
	} else {
		stdlog.Printf("progme:pfctl start success:%s", rt)
	}

	output, err = exec.Command("/bin/sh", "-c","sudo pfctl -f ./etc/pf.conf").CombinedOutput()
	if err != nil {
		stdlog.Printf("progme:pfctl reload rules failed:%s" , string(output))
	} else {
		stdlog.Printf("progme:pfctl reload rules success:%s" , string(output))
	}
}
