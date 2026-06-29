package downloadservices

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

func ValidateDownloadPath(path string) error {
	if !filepath.IsAbs(path) {
		return errors.New("路径错误，请输入绝对路径")
	}
	if strings.HasPrefix(path, "/mnt") || strings.HasPrefix(path, "/root") {
		return nil
	}
	return errors.New("路径错误，必须选择硬盘的路径或者/root的路径")
}

func BuildEnsureDownloadDirCommands(path string) []string {
	return []string{
		fmt.Sprintf("if [ ! -d '%v' ]; then mkdir -p '%v'; fi", path, path),
		fmt.Sprintf("chmod 777 '%v'", path),
	}
}
