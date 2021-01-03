package setup

import "os/exec"

// check if binary is present in PATH
func Which(binary string) error {
	_, err := exec.LookPath(binary)

	return err
}
