package mobile

import (
	v2 "internal-libcore/v2"
	_ "github.com/sagernet/gomobile"
)

func Setup(baseDir, workingDir, tempDir string, debug bool) error {
	return v2.Setup(baseDir, workingDir, tempDir, 0, debug)
}
