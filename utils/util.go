package utils

import "strings"

func GenDir(u2 string) string {
	var dir string
	for _, v := range strings.Split(u2, "-") {
		dir += v[0:1] + "/"
	}
	return dir
}
