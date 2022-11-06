//
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

type Printer interface {
	Start(...string)
	Success(string)
	Warning(string)
	Fail(string)
	UpdateText(string)
	PrintSummary(pass, fail, skip int)
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
	p.spinner.Success(msg)
}

func (p *printer) Warning(msg string) {
	p.spinner.Warning(msg)
}

func (p *printer) Fail(msg string) {
	p.spinner.Fail(msg)
}

func (p *printer) UpdateText(text string) {
	if p.batchOutput {
		return
	}

	p.spinner.UpdateText(text)
}

// PrintSummary outputs a summary of verify result.
// The summary shows the number of the successful, failed and skipped cases.
func (p *printer) PrintSummary(pass, fail, skip int) {
	pterm.Info.Prefix = pterm.Prefix{
		Text:  "SUMMARY",
		Style: &pterm.ThemeDefault.InfoPrefixStyle,
	}
	pterm.Info.WithMessageStyle(&pterm.Style{pterm.FgGreen}).Println(fmt.Sprintf("%d passed", pass))

	pterm.Info.Prefix = pterm.Prefix{
		Text:  "       ",
		Style: &pterm.ThemeDefault.InfoPrefixStyle,
	}
	pterm.Info.WithMessageStyle(&pterm.Style{pterm.FgLightRed}).Println(fmt.Sprintf("%d failed", fail))
	pterm.Info.WithMessageStyle(&pterm.Style{pterm.FgYellow}).Println(fmt.Sprintf("%d skipped", skip))
	fmt.Println()
}
