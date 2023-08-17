package output

import (
	"fmt"

	"gopkg.in/yaml.v2"
)

// VerifyResult stores the result of verify
type VerifyResult struct {
	Passed  []string
	Failed  []string
	Skipped []string
}

// OutputVerifyResultInYAML outputs the verifyResult in YAML
func (verifyResult *VerifyResult) OutputVerifyResultInYAML() {
	yaml, _ := yaml.Marshal(verifyResult)
	fmt.Print(string(yaml))
}

// AddPassedCase adds the passed cases to verifyResult
func (verifyResult *VerifyResult) AddPassedCase(caseName string) {
	verifyResult.Passed = append(verifyResult.Passed, caseName)
}

// AddFailedCase adds the failed cases to verifyResult
func (verifyResult *VerifyResult) AddFailedCase(caseName string) {
	verifyResult.Failed = append(verifyResult.Failed, caseName)
}

// AddSkippedCase adds the skipped cases to verifyResult
func (verifyResult *VerifyResult) AddSkippedCase(caseName string) {
	verifyResult.Skipped = append(verifyResult.Skipped, caseName)
}
