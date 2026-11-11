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

func TestFormatNameNumericAcronymBoundary(t *testing.T) {
	cases := []struct {
		input, camel, snake string
	}{
		{"ChannelV2Id", "channelV2Id", "channel_v2_id"},
		{"ChannelV2ID", "channelV2Id", "channel_v2_id"},
		{"channelV2Id", "channelV2Id", "channel_v2_id"},
		{"ChannelID", "channelId", "channel_id"},
		{"AdvertiserId", "advertiserId", "advertiser_id"},
		{"HTTP2Server", "http2Server", "http2_server"},
		{"14Day", "14Day", "14_day"},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			for _, format := range []struct{ name, want string }{{"lc", tc.camel}, {"snake", tc.snake}} {
				if got := FormatName(tc.input, format.name); got != format.want {
					t.Errorf("FormatName(%q, %q) = %q, want %q", tc.input, format.name, got, format.want)
				}
			}
			if got := FormatName(tc.camel, "lc"); got != tc.camel {
				t.Errorf("lowerCamel is not idempotent: %q becomes %q", tc.camel, got)
			}
		})
	}
}
