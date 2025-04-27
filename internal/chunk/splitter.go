package chunk

// Split300 slices the string into ≤300-어절 chunks
// without decoding UTF-8 runes.  ZERO copies, 1 slice alloc.
func Split300(s string) []string {
	const max = 300

	// Capacity hint: assume “avg 5-bytes‐word + 1 space”.
	// For 5 000 어절 ≈ 30 000 B ⇒ hint ≈ 3.  Good enough.
	hint := len(s)/(max*6) + 1
	res := make([]string, 0, hint)

	start, words := 0, 0
	for i := 0; i < len(s); i++ {
		b := s[i]
		if b == ' ' || b == '\n' {
			words++
			if words == max {
				res = append(res, s[start:i])
				start, words = i+1, 0
			}
		}
	}
	// trailing slice (never empty because start ≤ len(s))
	res = append(res, s[start:])
	return res
}
