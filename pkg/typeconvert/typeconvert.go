package typeconvert

import (
	"strconv"
)

// Uint32ToStr converts uint32 to string.
func Uint32ToStr(val uint32) string {
	return strconv.FormatUint(uint64(val), 10)
}

// StrToUint32 converts string to uint32. Returns 0 if an error happens
func StrToUint32(str string) (uint32, error) {
	parsed, err := strconv.ParseUint(str, 10, 64)
	if err != nil {
		return 0, err
	}

	return uint32(parsed), nil
}
