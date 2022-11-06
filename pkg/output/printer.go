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
