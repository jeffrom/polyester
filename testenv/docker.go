package testenv

import "os/exec"

func DockerAvailable() error {
	if _, err := exec.LookPath("docker"); err != nil {
		return err
	}
	return nil
}
