package clipboard

import (
	"fmt"
	"os/exec"
	"strings"
)

func IsPbCopyAvailable() bool {
	cmd := exec.Command("which", "pbcopy")
	if err := cmd.Run(); err != nil {
		return false
	}
	return true
}

func Copy(text string) error {
	cmd := exec.Command("pbcopy")
	cmd.Stdin = strings.NewReader(text)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("could not copy to clipboard: %s", err)
	}
	return nil
}
