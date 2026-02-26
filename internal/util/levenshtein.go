package util

// Levenshtein returns the edit distance between two strings (rune-aware).
// Uses the standard DP approach with a single rolling row to keep allocations minimal.
func Levenshtein(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	la, lb := len(ra), len(rb)

	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}

	// row[j] = distance(ra[:i], rb[:j])
	row := make([]int, lb+1)
	for j := range row {
		row[j] = j
	}

	for i := 1; i <= la; i++ {
		prev := i
		for j := 1; j <= lb; j++ {
			cost := row[j-1]
			if ra[i-1] != rb[j-1] {
				cost++ // substitute
				if row[j]+1 < cost {
					cost = row[j] + 1 // delete
				}
				if prev+1 < cost {
					cost = prev + 1 // insert
				}
			}
			row[j-1] = prev
			prev = cost
		}
		row[lb] = prev
	}
	return row[lb]
}
