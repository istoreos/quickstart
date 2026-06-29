package utils

import (
	"fmt"
	"strconv"
)

// func ByteCountBinary(b uint64) string {
// 	const unit = 1024
// 	if b < unit {
// 		return strconv.FormatUint(b, 10) + "B"
// 	}
// 	div, exp := int64(unit), 0
// 	for n := b / unit; n >= unit; n /= unit {
// 		div *= unit
// 		exp++
// 	}
// 	//return strconv.FormatInt(b/div, 10) + "KMGTPE"[exp:exp+1]
// 	// Fix display bug
// 	return strconv.FormatFloat(float64(b)/float64(div), 'f', 1, 32) + "KMGTPE"[exp:exp+1]
// }

func ByteCountDecimal(b uint64) string {
	const unit = 1000
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

func ByteCountBinary(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}

func PartedByteCountBinary(b uint64) string {
	return strconv.FormatUint(b, 10) + "B"
}
