// Licensed to Apache Software Foundation (ASF) under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Apache Software Foundation (ASF) licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.
//

package config

// E2EConfig corresponds to configuration file e2e.yaml.
type E2EConfig struct {
	Setup  Setup        `yaml:"setup"`
	Verify []VerifyCase `yaml:"verify"`
}

type Setup struct {
	Env       string     `yaml:"env"`
	File      string     `yaml:"file"`
	Manifests []Manifest `yaml:"manifests"`
	// Run is not supported yet
	Run     []Run `yaml:"run"`
	Timeout int   `yaml:"timeout"`
}

type Manifest struct {
	Path  string `yaml:"path"`
	Waits []Wait `yaml:"wait"`
}

type Run struct {
	Command string `yaml:"command"`
	Waits   []Wait `yaml:"wait"`
}

type Wait struct {
	Namespace     string `yaml:"namespace"`
	Resource      string `yaml:"resource"`
	LabelSelector string `yaml:"label-selector"`
	For           string `yaml:"for"`
}

type VerifyCase struct {
	Query    string `yaml:"query"`
	Actual   string `yaml:"actual"`
	Expected string `yaml:"expected"`
}
