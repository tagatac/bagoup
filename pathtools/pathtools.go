package pathtools

import (
	"os/user"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

func ReplaceTilde(filePath string) (string, error) {
	if strings.HasPrefix(filePath, "~") {
		usr, err := user.Current()
		if err != nil {
			return "", errors.Wrap(err, "get current user")
		}
		filePath = filepath.Join(usr.HomeDir, filePath[1:])
	}
	return filePath, nil
}
