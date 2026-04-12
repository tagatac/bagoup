package imgconvert

import "github.com/tagatac/goheif/heic2jpg"

type imgConverter struct {
	heic2jpg.Converter
	tempDir string
}

func NewImgConverter(tempDir string) ImgConverter {
	return &imgConverter{
		Converter: heic2jpg.NewConverter(),
		tempDir:   tempDir,
	}
}

func (i *imgConverter) convert(src, dst string) error {
	return i.HEIC2JPG(src, dst)
}
