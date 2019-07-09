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

func SliceEquals(arr1 interface{}, arr2 interface{}) bool {
	switch arr1.(type) {
	case []string:
		arr1S := arr1.([]string)
		arr2S := arr2.([]string)
		if len(arr1S) != len(arr2S) {
			return false
		}
		for i, v1S := range arr1S {
			if v1S != arr2S[i] {
				return false
			}
		}
		return true
	default:
		return false
	}
}
