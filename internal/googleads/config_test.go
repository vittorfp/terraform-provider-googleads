package googleads

import (
	"reflect"
	"sort"
	"testing"
)

func TestConfigMissing(t *testing.T) {
	base := Config{}
	cases := []struct {
		name string
		cfg  Config
		want []string
	}{
		{
			name: "fully empty — ADC path with optional request headers",
			cfg:  base,
			want: nil,
		},
		{
			name: "full refresh-token triplet — no missing",
			cfg:  Config{ClientID: "c", ClientSecret: "s", RefreshToken: "r"},
			want: nil,
		},
		{
			name: "partial refresh-token triplet — flags the gaps",
			cfg:  Config{ClientID: "c"},
			want: []string{"client_secret", "refresh_token"},
		},
		{
			name: "full service-account pair — no missing",
			cfg:  Config{ServiceAccountJSONPath: "/sa.json", ImpersonateEmail: "u@example.com"},
			want: nil,
		},
		{
			name: "partial service-account: path without email",
			cfg:  Config{ServiceAccountJSONPath: "/sa.json"},
			want: []string{"impersonate_email"},
		},
		{
			name: "partial service-account: email without path",
			cfg:  Config{ImpersonateEmail: "u@example.com"},
			want: []string{"service_account_json_path"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.cfg.Missing()
			sort.Strings(got)
			sort.Strings(tc.want)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestConfigAuthPrecedence(t *testing.T) {
	cases := []struct {
		name               string
		cfg                Config
		wantServiceAccount bool
		wantRefreshToken   bool
		wantADC            bool
	}{
		{
			name:    "no auth fields → ADC",
			cfg:     Config{},
			wantADC: true,
		},
		{
			name: "refresh token only",
			cfg: Config{
				ClientID: "c", ClientSecret: "s", RefreshToken: "r",
			},
			wantRefreshToken: true,
		},
		{
			name: "service account only",
			cfg: Config{
				ServiceAccountJSONPath: "/sa.json", ImpersonateEmail: "u@example.com",
			},
			wantServiceAccount: true,
		},
		{
			name: "service account AND refresh token — SA wins",
			cfg: Config{
				ServiceAccountJSONPath: "/sa.json", ImpersonateEmail: "u@example.com",
				ClientID: "c", ClientSecret: "s", RefreshToken: "r",
			},
			wantServiceAccount: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got, want := tc.cfg.UsesServiceAccount(), tc.wantServiceAccount; got != want {
				t.Errorf("UsesServiceAccount() = %v, want %v", got, want)
			}
			if got, want := tc.cfg.UsesRefreshToken(), tc.wantRefreshToken; got != want {
				t.Errorf("UsesRefreshToken() = %v, want %v", got, want)
			}
			if got, want := tc.cfg.UsesADC(), tc.wantADC; got != want {
				t.Errorf("UsesADC() = %v, want %v", got, want)
			}
		})
	}
}

func TestRequestHeaders(t *testing.T) {
	cases := []struct {
		name string
		cfg  Config
		want []string
	}{
		{
			name: "direct access omits optional headers",
			cfg:  Config{},
			want: nil,
		},
		{
			name: "manager access sends login customer only",
			cfg:  Config{LoginCustomerID: "1111111111"},
			want: []string{"login-customer-id", "1111111111"},
		},
		{
			name: "legacy API sends both configured headers",
			cfg: Config{
				DeveloperToken:  "legacy-token",
				LoginCustomerID: "1111111111",
			},
			want: []string{
				"developer-token", "legacy-token",
				"login-customer-id", "1111111111",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := requestHeaders(tc.cfg); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("requestHeaders() = %v, want %v", got, tc.want)
			}
		})
	}
}
