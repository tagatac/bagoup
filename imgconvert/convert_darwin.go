package imgconvert

import "os/exec"

type imgConverter struct {
	execCommand func(string, ...string) *exec.Cmd
	tempDir     string
}

func NewImgConverter(tempDir string) ImgConverter {
	return &imgConverter{
		execCommand: exec.Command,
		tempDir:     tempDir,
	}
}

func (i *imgConverter) convert(src, dst string) error {
	cmd := i.execCommand(
		"sips",
		"--setProperty", "format", "jpeg",
		"--setProperty", "formatOptions", "best",
		"--out", dst,
		src,
	)
	return cmd.Run()
}
