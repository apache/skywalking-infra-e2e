package verifier

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/apache/skywalking-infra-e2e/internal/logger"
	"github.com/apache/skywalking-infra-e2e/internal/util"
	"github.com/apache/skywalking-infra-e2e/third-party/template"

	"github.com/google/go-cmp/cmp"
	"gopkg.in/yaml.v2"
)

// MismatchError is the error type returned by the verify functions.
// Then the caller will know if there is a mismatch.
type MismatchError struct {
	Err error
}

func (e *MismatchError) Unwrap() error { return e.Err }

func (e *MismatchError) Error() string {
	if e == nil {
		return "<nil>"
	}
	return "the actual data does not match the expected data"
}

// VerifyDataFile reads the actual data from the file and verifies.
func VerifyDataFile(actualFile, expectedFile string) error {
	actualData, err := util.ReadFileContent(actualFile)
	if err != nil {
		logger.Log.Error("failed to read the actual data file")
		return err
	}

	expectedTemplate, err := util.ReadFileContent(expectedFile)
	if err != nil {
		logger.Log.Error("failed to read the expected data file")
		return err
	}

	return verify(actualData, expectedTemplate)
}

// VerifyQuery gets the actual data from the query and then verifies.
func VerifyQuery(query, expectedFile string) error {
	return errors.New("not implemented")
}

// verify checks if the actual data match the expected template.
// It will print the diff if the actual data does not match.
func verify(actualData, expectedTemplate string) error {
	var actual interface{}
	if err := yaml.Unmarshal([]byte(actualData), &actual); err != nil {
		logger.Log.Error("failed to unmarshal actual data")
		return err
	}

	tmpl, err := template.New("test").Funcs(funcMap()).Parse(expectedTemplate)
	if err != nil {
		logger.Log.Error("failed to parse template")
		return err
	}

	var b bytes.Buffer
	if err := tmpl.Execute(&b, actual); err != nil {
		logger.Log.Error("failed to execute template")
		return err
	}

	var expected interface{}
	if err := yaml.Unmarshal(b.Bytes(), &expected); err != nil {
		logger.Log.Error("failed to unmarshal expected data")
		return err
	}

	if !cmp.Equal(expected, actual) {
		diff := cmp.Diff(expected, actual)
		fmt.Println(diff)
		return &MismatchError{}
	} else {
		logger.Log.Info("the actual data matches the expected data")
	}

	return nil
}
