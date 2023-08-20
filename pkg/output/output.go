package output

import (
	"fmt"
	"gopkg.in/yaml.v2"
)

var (
	Format  string
	Formats = []string{"yaml"}
)

type YamlCaseResult struct {
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

func PrintResult(caseRes []*CaseResult) {
	switch Format {
	case "yaml":
		PrintResultInYAML(caseRes)
	}
}

func PrintResultInYAML(caseRes []*CaseResult) {
	var yamlCaseResult YamlCaseResult
	for _, cr := range caseRes {
		if !cr.Skip {
			if cr.Err == nil {
				yamlCaseResult.Passed = append(yamlCaseResult.Passed, cr.CaseName)
			} else {
				yamlCaseResult.Failed = append(yamlCaseResult.Failed, cr.CaseName)
			}
		} else {
			yamlCaseResult.Skipped = append(yamlCaseResult.Skipped, cr.CaseName)
		}
	}

	yaml, _ := yaml.Marshal(yamlCaseResult)
	fmt.Print(string(yaml))
}
