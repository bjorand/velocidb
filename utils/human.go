package utils

import "fmt"

func HumanSizeBytes(v int64) string {
	units := []string{"B", "KB", "MB", "GB", "TB", "PB"}
	unit := "B"
	unitIndex := 0
	for {
		if unitIndex > (len(units) - 2) {
			break
		}
		if v > 1000 {
			unitIndex = unitIndex + 1
			unit = units[unitIndex]
			v = v / 1000
			continue
		}
		break
	}
	return fmt.Sprintf("%d%s", v, unit)
}
