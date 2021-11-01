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
	"bufio"
	"context"
	"io"
	"os"
	"path/filepath"
	"sync"
)

type ResourceLogFollower struct {
	Ctx        context.Context
	cancelFunc context.CancelFunc
	basePath   string
	followLock *sync.RWMutex
	following  map[string]bool
}

func NewResourceLogFollower(ctx context.Context, basePath string) *ResourceLogFollower {
	childCtx, cancelFunc := context.WithCancel(ctx)
	return &ResourceLogFollower{
		Ctx:        childCtx,
		cancelFunc: cancelFunc,
		basePath:   basePath,
		followLock: &sync.RWMutex{},
		following:  make(map[string]bool),
	}
}

func (l *ResourceLogFollower) BuildLogWriter(path string) (*os.File, error) {
	logFile := l.buildLogFilename(path)
	if err := os.MkdirAll(filepath.Dir(logFile), os.ModePerm); err != nil {
		return nil, err
	}
	if _, err := os.Stat(logFile); os.IsExist(err) {
		if err := os.Remove(logFile); err != nil {
			return nil, err
		}
	}

	return os.Create(logFile)
}

func (l *ResourceLogFollower) ConsumeLog(logWriter *os.File, stream io.ReadCloser) <-chan struct{} {
	if l.IsFollowed(logWriter.Name()) {
		return nil
	}

	finished := make(chan struct{}, 1)
	go func() {
		defer func() {
			stream.Close()
			close(finished)
		}()

		r := bufio.NewReader(stream)
		for {
			bytes, err := r.ReadBytes('\n')
			if err != nil {
				if err != io.EOF {
					return
				}
				return
			}

			l.writeFollowed(logWriter)
			if _, err := logWriter.Write(bytes); err != nil {
				return
			}
		}
	}()
	return finished
}

func (l *ResourceLogFollower) IsFollowed(path string) bool {
	l.followLock.RLock()
	defer l.followLock.RUnlock()
	return l.following[l.buildLogFilename(path)]
}

func (l *ResourceLogFollower) Close() {
	l.cancelFunc()
}

func (l *ResourceLogFollower) buildLogFilename(path string) string {
	return filepath.Join(l.basePath, path)
}

func (l *ResourceLogFollower) writeFollowed(writer *os.File) {
	l.followLock.Lock()
	defer l.followLock.Unlock()
	l.following[writer.Name()] = true
}
