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


package config

import (
	"github.com/boxproject/boxguard/scanproc"
)


var(
	GlbCfg      GlobalConfig
)

type GlobalConfig struct {
	//Basedir	 string		`toml:"basedir"`
	Monitor         MonitorInfo
	WaitSeconds     int
	EnablePfctl     bool
	EnableProcGuard bool
	WhiteList       []string
	WhiteMap        map[string]bool
	AllowUser		string
}

func (glbcfg *GlobalConfig) InitData() {
	if !GlbCfg.EnableProcGuard {
		return
	}
	scanproc.Logger.Println("Global config init begin")
	glbcfg.WhiteMap = make(map[string]bool)
	for _, proc := range glbcfg.WhiteList {
		glbcfg.WhiteMap[proc] = true
	}
	scanproc.Logger.Printf("Global config init end,white proc data---->%v",glbcfg.WhiteMap)
}

func (glbcfg *GlobalConfig) InWhite(fullPath string) bool {
	_, ok := glbcfg.WhiteMap[fullPath]
	return ok
}

type MonitorInfo struct {
	Users   int
	Frames  int
	PrcName string
}
