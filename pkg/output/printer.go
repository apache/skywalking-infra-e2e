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

package output

import (
	"fmt"

	"github.com/pterm/pterm"
)

// CaseResult represents the result of a verification case.
type CaseResult struct {
	Msg  string
	Err  error
	Skip bool
}

type Printer interface {
	Start(...string)
	Success(string)
	Warning(string)
	Fail(string)
	UpdateText(string)
	PrintResult([]*CaseResult) (int, int, int)
}

type printer struct {
	spinner     *pterm.SpinnerPrinter
	batchOutput bool
}

var _ Printer = &printer{}

func NewPrinter(batchOutput bool) Printer {
	spinner := pterm.DefaultSpinner.WithShowTimer(false)
	pterm.Error.Prefix = pterm.Prefix{
		Text:  "DETAILS",
		Style: &pterm.ThemeDefault.ErrorPrefixStyle,
	}

	return &printer{
		spinner:     spinner,
		batchOutput: batchOutput,
	}
}

func (p *printer) Start(msg ...string) {
	if p.batchOutput {
		return
	}

	p.spinner, _ = p.spinner.Start(msg)
}

func (p *printer) Success(msg string) {
	if p.batchOutput {
		return
	}

	p.spinner.Success(msg)
}

func (p *printer) Warning(msg string) {
	if p.batchOutput {
		return
	}

	p.spinner.Warning(msg)
}

func (p *printer) Fail(msg string) {
	if p.batchOutput {
		return
	}

	p.spinner.Fail(msg)
}

func (p *printer) UpdateText(text string) {
	if p.batchOutput {
		return
	}

	p.spinner.UpdateText(text)
}

// PrintResult prints the result of verification and the summary.
// If bathOutput is false, will only print the summary.
func (p *printer) PrintResult(caseRes []*CaseResult) (passNum, failNum, skipNum int) {
	// Count the number of passed and failed.
	// If batchOutput is true, print the result of all cases in a batch.
	for _, cr := range caseRes {
		if !cr.Skip {
			if cr.Err == nil {
				passNum++
				if p.batchOutput {
					p.spinner.Success(cr.Msg)
				}
			} else {
				failNum++
				if p.batchOutput {
					p.spinner.Warning(cr.Msg)
					p.spinner.Fail(cr.Err.Error())
				}
			}
		} else {
			skipNum++
		}
	}

	// Print the summary.
	pterm.Info.Prefix = pterm.Prefix{
		Text:  "SUMMARY",
		Style: &pterm.ThemeDefault.InfoPrefixStyle,
	}
	pterm.Info.WithMessageStyle(&pterm.Style{pterm.FgGreen}).Println(fmt.Sprintf("%d passed", passNum))
	pterm.Info.Prefix = pterm.Prefix{
		Text:  "       ",
		Style: &pterm.ThemeDefault.InfoPrefixStyle,
	}
	pterm.Info.WithMessageStyle(&pterm.Style{pterm.FgLightRed}).Println(fmt.Sprintf("%d failed", failNum))
	pterm.Info.WithMessageStyle(&pterm.Style{pterm.FgYellow}).Println(fmt.Sprintf("%d skipped", skipNum))
	fmt.Println()

	return passNum, failNum, skipNum
}
