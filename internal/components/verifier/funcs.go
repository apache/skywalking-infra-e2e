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

package verifier

import (
	"encoding/base64"
	"fmt"
	"regexp"
	"strings"

	"github.com/apache/skywalking-infra-e2e/third-party/go/template"
)

// funcMap produces the custom function map.
// Use this to pass the functions into the template engine:
// 	tpl := template.New("foo").Funcs(funcMap()))
func funcMap() template.FuncMap {
	fm := make(map[string]interface{}, len(customFuncMap))
	for k, v := range customFuncMap {
		fm[k] = v
	}
	return template.FuncMap(fm)
}

var customFuncMap = map[string]interface{}{
	// Basic:
	"notEmpty": notEmpty,

	// Encoding:
	"b64enc": base64encode,

	// Regex:
	"regexp": regexpMatch,
}

func base64encode(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}

func notEmpty(s string) string {
	if len(strings.TrimSpace(s)) > 0 {
		return s
	}
	return fmt.Sprintf("<%q is empty, wanted is not empty>", s)
}

func regexpMatch(s, pattern string) string {
	matched, err := regexp.MatchString(pattern, s)
	if err != nil {
		return fmt.Sprintf(`<%q>`, err)
	}
	if !matched {
		// Note: Changing %s to %q for s would throw yaml parsing error
		return fmt.Sprintf("<%s does not match the pattern %q>", s, pattern)
	}
	return s
}
