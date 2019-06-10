package utils

import "fmt"

func lpopString(pArr *[]string) (string, error) {
	arr := *pArr
	if len(arr) > 0 {
		*pArr = arr[1:]
		return arr[0], nil
	}
	*pArr = []string{}
	return "", fmt.Errorf("Cannot pop empty list")

}
