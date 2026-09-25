package googleads

import "testing"

func TestParseResourceName(t *testing.T) {
	cases := []struct {
		in       string
		kind     string
		wantCid  string
		wantRest string
		wantErr  bool
	}{
		{"customers/123/campaigns/456", "campaigns", "123", "456", false},
		{"customers/123/adGroupAds/789~111", "adGroupAds", "123", "789~111", false},
		{"customers/123/campaigns/456", "adGroups", "", "", true},
		{"customers//campaigns/456", "campaigns", "", "", true},
		{"customers/123/campaigns/", "campaigns", "", "", true},
		{"garbage", "campaigns", "", "", true},
		{"customers/123/campaigns/456/extra", "campaigns", "", "", true},
	}
	for _, tc := range cases {
		cid, rest, err := ParseResourceName(tc.in, tc.kind)
		if (err != nil) != tc.wantErr {
			t.Fatalf("ParseResourceName(%q, %q) err=%v wantErr=%v", tc.in, tc.kind, err, tc.wantErr)
		}
		if !tc.wantErr {
			if cid != tc.wantCid || rest != tc.wantRest {
				t.Fatalf("ParseResourceName(%q, %q) = (%q, %q), want (%q, %q)", tc.in, tc.kind, cid, rest, tc.wantCid, tc.wantRest)
			}
		}
	}
}

func TestBuildResourceName(t *testing.T) {
	got := BuildResourceName("123", "campaigns", "456")
	if got != "customers/123/campaigns/456" {
		t.Fatalf("BuildResourceName = %q", got)
	}
}

func TestCompositeID(t *testing.T) {
	got := CompositeID("789", "111")
	if got != "789~111" {
		t.Fatalf("CompositeID = %q", got)
	}
	parent, child, err := SplitComposite("789~111")
	if err != nil || parent != "789" || child != "111" {
		t.Fatalf("SplitComposite returned (%q, %q, %v)", parent, child, err)
	}
	if _, _, err := SplitComposite("nope"); err == nil {
		t.Fatal("SplitComposite expected error")
	}
}
