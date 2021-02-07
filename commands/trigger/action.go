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

package trigger

import (
	"strconv"
	"time"
)

type Action interface {
	Do() error
}

type action struct {
	interval time.Duration
	times    int
}

func ParseInterval(s string) time.Duration {
	if len(s) >= 1 {
		var base time.Duration
		switch s[len(s)-1:] {
		case "s":
			base = time.Second
		case "m":
			base = time.Minute
		case "h":
			base = time.Hour
		case "d":
			base = time.Hour * 24
		default:
			base = time.Second
		}

		i, e := strconv.Atoi(s[:len(s)-1])
		if e != nil {
			return time.Second
		}

		return time.Duration(i) * base
	}
	return time.Second
}
