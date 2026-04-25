package imgconv

import "os/exec"

func (i *imgConverter) convCmd(src, dst string) *exec.Cmd {
	return i.execCommand("magick", "convert", src, dst)
}
