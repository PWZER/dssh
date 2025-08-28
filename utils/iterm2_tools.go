package utils

import (
	"fmt"
	"os"
)

func SetWindowTitle(title string) (err error) {
	if os.Getenv("LC_TERMINAL") == "iTerm2" {
		if title == "" {
			if title, err = os.Getwd(); err != nil {
				return err
			}
		}
		_, err = os.Stdin.WriteString(fmt.Sprintf("\033]0;%s\007", title))
	}
	return err
}
