package utils

func StrInArray(a []string, s string) bool {
	for _, i := range a {
		if i == s {
			return true
		}
	}
	return false
}
