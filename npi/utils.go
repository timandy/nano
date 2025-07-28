package npi

import (
	"strings"
	"unicode"
)

// JoinPaths 拼接路由路径, basePath 肯定是 clean 的
func JoinPaths(basePath, relativePath string) string {
	relativePath = strings.TrimFunc(relativePath, trimFunc)
	if relativePath == "" {
		return basePath
	}
	if basePath == "" {
		return relativePath
	}
	return basePath + "." + relativePath
}

func trimFunc(c rune) bool {
	return c == '/' || c == '.' || unicode.IsSpace(c)
}
