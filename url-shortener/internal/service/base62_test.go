package service

import "testing"

func TestEncodeBase62(t *testing.T) {
	cases := []struct {
		in   int64
		want string
	}{
		{0, "0"},
		{1, "1"},
		{61, "Z"},
		{62, "10"},
		{3843, "ZZ"},
	}
	for _, c := range cases {
		if got := EncodeBase62(c.in); got != c.want {
			t.Errorf("EncodeBase62(%d) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestCodeForIDIsDeterministicAndUnique(t *testing.T) {
	seen := make(map[string]int64)
	for id := int64(1); id <= 10000; id++ {
		code := codeForID(id)
		if prev, ok := seen[code]; ok {
			t.Fatalf("collision: id %d and %d both -> %q", prev, id, code)
		}
		seen[code] = id
		if codeForID(id) != code {
			t.Fatalf("codeForID not deterministic for id %d", id)
		}
	}
}

func TestValidURL(t *testing.T) {
	valid := []string{"http://a.com", "https://a.com/x?y=1"}
	invalid := []string{"", "ftp://a.com", "not a url", "a.com"}
	for _, v := range valid {
		if !validURL(v) {
			t.Errorf("expected %q valid", v)
		}
	}
	for _, v := range invalid {
		if validURL(v) {
			t.Errorf("expected %q invalid", v)
		}
	}
}
