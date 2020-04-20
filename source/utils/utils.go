package utils

func CopySlice(slice [][]bool) [][]bool {
	ns := make([][]bool, len(slice))
	for i := range slice {
		ns[i] = make([]bool, len(slice[i]))
		copy(ns[i], slice[i])
	}
	return ns
}
