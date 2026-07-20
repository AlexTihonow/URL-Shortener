package service

const base62Alphabet = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

const codeOffset = 1_000_000

func EncodeBase62(n int64) string {
	if n == 0 {
		return string(base62Alphabet[0])
	}
	buf := make([]byte, 0, 12)
	for n > 0 {
		buf = append(buf, base62Alphabet[n%62])
		n /= 62
	}
	for i, j := 0, len(buf)-1; i < j; i, j = i+1, j-1 {
		buf[i], buf[j] = buf[j], buf[i]
	}
	return string(buf)
}

func codeForID(id int64) string {
	return EncodeBase62(id + codeOffset)
}
