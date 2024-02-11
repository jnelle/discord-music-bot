package utils

func ArrayContains[T comparable](haystack []T, needle T) bool {
	for _, val := range haystack {
		if val == needle {
			return true
		}
	}
	return false
}

func ToPtr[T comparable](value T) *T {
	return &value
}
