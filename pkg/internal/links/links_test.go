package links

import "testing"

func TestParseBundleID(t *testing.T) {
	cases := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{"stripe/api", "stripe/api", false},
		{"hub://stripe/api", "stripe/api", false},
		{"/stripe/api/", "stripe/api", false},
		{"stripe", "", true},
		{"stripe/api/extra", "", true},
		{"/api", "", true},
		{"", "", true},
	}
	for _, tc := range cases {
		got, err := ParseBundleID(tc.in)
		if (err != nil) != tc.wantErr {
			t.Errorf("ParseBundleID(%q) err=%v, wantErr=%v", tc.in, err, tc.wantErr)
		}
		if got != tc.want {
			t.Errorf("ParseBundleID(%q)=%q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestParseHubURI(t *testing.T) {
	cases := []struct {
		in          string
		wantBundle  string
		wantConcept string
		wantErr     bool
	}{
		{"hub://stripe/api", "stripe/api", "", false},
		{"hub://stripe/api/charges/create", "stripe/api", "charges/create", false},
		{"hub://stripe/api/charges/create.md#params", "stripe/api", "charges/create", false},
		{"hub://onlyone", "", "", true},
		{"hub://stripe/api/", "", "", true},
		{"hub://stripe/../secret", "", "", true},
		{"stripe/api", "", "", true},
	}
	for _, tc := range cases {
		b, c, err := ParseHubURI(tc.in)
		if (err != nil) != tc.wantErr {
			t.Errorf("ParseHubURI(%q) err=%v, wantErr=%v", tc.in, err, tc.wantErr)
		}
		if b != tc.wantBundle || c != tc.wantConcept {
			t.Errorf("ParseHubURI(%q)=(%q, %q), want (%q, %q)", tc.in, b, c, tc.wantBundle, tc.wantConcept)
		}
	}
}
