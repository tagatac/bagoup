package imgconvert

import (
	"testing"
)

func TestNewImgConverter(t *testing.T) {
	_ = NewImgConverter("testTempDir")
}
