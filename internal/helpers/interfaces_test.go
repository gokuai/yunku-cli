package helpers

import "testing"

func TestValidateNaming(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		vendor  string
		extName string
		wantErr bool
	}{
		{name: "valid", vendor: "yunku", extName: "oa-plus"},
		{name: "short-vendor", vendor: "y", extName: "oa-plus", wantErr: true},
		{name: "invalid-vendor", vendor: "1yunku", extName: "oa-plus", wantErr: true},
		{name: "invalid-name", vendor: "yunku", extName: "oa_plus", wantErr: true},
		{name: "empty-name", vendor: "yunku", extName: "", wantErr: true},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateNaming(tc.vendor, tc.extName)
			if tc.wantErr && err == nil {
				t.Fatalf("ValidateNaming(%q, %q) error = nil, want failure", tc.vendor, tc.extName)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("ValidateNaming(%q, %q) error = %v, want nil", tc.vendor, tc.extName, err)
			}
		})
	}
}
