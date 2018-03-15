package testfuncs

import "strconv"

// DoubleUint64 uses a uint64 as an argument for testing purposes. It returns 2x
// the argument.
func DoubleUint64(a uint64) uint64 {
	return a * 2
}

// MergeInts uses a variadic argument for testing purposes.  It also contains 
// another argument of the same type to test for deduplication of converters.
// It simply mashes all the arguments together into a giant string.
func MergeInts(first int, ints ...int) string {
	result := strconv.Itoa(first)
	for _, i := range ints {
		result += strconv.Itoa(i)
	}
	return result
}
