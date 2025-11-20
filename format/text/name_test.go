package text

import "testing"

func TestFormatName(t *testing.T) {
	cases := []struct {
		in, to, want string
	}{
		{"UserID", "snake", "user_id"},
		{"userID", "lc", "userId"},
		{"UserName", "uu", "USER_NAME"},
		{"user_name", "upperCamel", "UserName"},
	}
	for _, c := range cases {
		if got := FormatName(c.in, c.to); got != c.want {
			t.Fatalf("FormatName(%q,%q): got %q, want %q", c.in, c.to, got, c.want)
		}
	}
}
