package utils

func RemoveFromSlice[E any](s []E, i int) []E {
	s[i] = s[len(s)-1]
	return s[:len(s)-1]
}
