//go:build windows

package manage

import "os"

func openTTYInput() (*os.File, error) {
	return os.OpenFile("CONIN$", os.O_RDONLY, 0)
}
