package imgconv

import "os/exec"

func (i *imgConverter) convCmd(src, dst string) *exec.Cmd {
	return i.execCommand(
		"sips",
		"--setProperty", "format", "jpeg",
		"--setProperty", "formatOptions", "best",
		"--out", dst,
		src,
	)
}
