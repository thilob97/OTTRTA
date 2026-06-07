package agent

import "os/exec"

func OmpExists() bool {
	_, err := exec.LookPath("omp")
	return err == nil
}
