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

type Action interface {
	// Do performs the trigger action according to the settings,
	// and returns an error channel, the controller waits for the
	// error and if it's nil, the controller considers the action
	// is successfully scheduled, otherwise the controller considers
	// this is a failure and aborts the process.
	//
	// It's guaranteed that the error channel will receive the first
	// error, and receive at most 1 error, all following errors will
	// not be returned.
	//
	// This returning style can be used to wait for the http action
	// being normal, and then schedule it, otherwise we can interrupt
	// according to the first error.
	Do() chan error

	// Stop stops the scheduled actions.
	Stop()
}
