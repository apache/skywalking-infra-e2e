package output

import (
	"fmt"
	"gopkg.in/yaml.v2"
)

var (
	Format  string
	Formats = []string{"yaml"}
)

type CaseInfo struct {
	Passed  []string
	Failed  []string
	Skipped []string
}

func FormatIsNotExist() bool {
	for _, format := range Formats {
		if Format == format {
			return false
		}
	}

	return true
}

func PrintTheOutput(caseInfo CaseInfo) {
	switch Format {
	case "yaml":
		caseInfo.PrintInYAML()
	}
}

func (caseInfo *CaseInfo) PrintInYAML() {
	yaml, _ := yaml.Marshal(caseInfo)
	fmt.Print(string(yaml))
}

func (caseInfo *CaseInfo) AddPassedCase(caseName string) {
	caseInfo.Passed = append(caseInfo.Passed, caseName)
}

func (caseInfo *CaseInfo) AddFailedCase(caseName string) {
	caseInfo.Failed = append(caseInfo.Failed, caseName)
}

func (caseInfo *CaseInfo) AddSkippedCase(caseName string) {
	caseInfo.Skipped = append(caseInfo.Skipped, caseName)
}
