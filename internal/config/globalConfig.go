package config

import (
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v2"

	"github.com/apache/skywalking-infra-e2e/internal/util"
)

// GlobalE2EConfig store E2EConfig which can be used globally.
type GlobalE2EConfig struct {
	Ready     bool
	E2EConfig E2EConfig
}

var GlobalConfig GlobalE2EConfig

func ReadGlobalConfigFile(configFilePath string) error {
	if GlobalConfig.Ready {
		return fmt.Errorf("can not read e2e config repeatedly")
	}

	e2eFile := configFilePath
	if util.PathExist(e2eFile) {
		// other command should check if global config is ready.
		data, err := ioutil.ReadFile(e2eFile)
		if err != nil {
			return fmt.Errorf("read e2e config file %s error: %s", e2eFile, err)
		}
		e2eConfigObject := E2EConfig{}
		err = yaml.Unmarshal(data, &e2eConfigObject)
		if err != nil {
			return fmt.Errorf("unmarshal e2e config file %s error: %s", e2eFile, err)
		}
		GlobalConfig.E2EConfig = e2eConfigObject
		GlobalConfig.Ready = true
	} else {
		return fmt.Errorf("e2e config file %s not exist", e2eFile)
	}

	if !GlobalConfig.Ready {
		return fmt.Errorf("e2e config read failed")
	}

	return nil
}
