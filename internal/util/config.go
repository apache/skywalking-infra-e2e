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

package util

import (
	"path"
	"path/filepath"

	"github.com/apache/skywalking-infra-e2e/internal/logger"
)

var (
	CfgFile string
	WorkDir string
	LogDir  string
)

// ResolveAbs resolves the relative path (relative to CfgFile) to an absolute file path.
func ResolveAbs(p string) string {
	abs, err := filepath.Abs(CfgFile)
	if err != nil {
		logger.Log.Warnf("failed to resolve the absolute file path of %v\n", CfgFile)
		return p
	}
	return ResolveAbsWithBase(p, abs)
}

// ResolveAbsWithBase resolves the relative path (relative to appoint file path) to an absolute file path.
func ResolveAbsWithBase(p, baseFile string) string {
	if p == "" {
		return p
	}

	if path.IsAbs(p) {
		return p
	}

	return filepath.Join(filepath.Dir(baseFile), p)
}
