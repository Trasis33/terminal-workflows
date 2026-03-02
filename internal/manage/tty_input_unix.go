//go:build !windows

package manage

import "os"

func openTTYInput() (*os.File, error) {
	return os.OpenFile("/dev/tty", os.O_RDONLY, 0)
}
