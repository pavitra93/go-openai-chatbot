package utils

func MakeHistoryWindow[T any](archive []T, userMessage string, keepLast int) []T {
	// how many from the tail (excluding the very first element)
	maxTail := len(archive) - 1
	if keepLast > maxTail {
		keepLast = maxTail
	}
	tail := archive[len(archive)-keepLast:]
	out := make([]T, 0, 1+len(tail))
	out = append(out, archive[0])
	out = append(out, tail...)
	return out
}
