package provider

import (
	"testing"
)

func TestParseCompositeID(t *testing.T) {
	tests := []struct {
		input        string
		wantSite     string
		wantResource string
		wantErr      bool
	}{
		{"site-123/resource-456", "site-123", "resource-456", false},
		{"abc/def", "abc", "def", false},
		{"site/with/slash", "site", "with/slash", false},
		{"noslash", "", "", true},
		{"/missingsite", "", "", true},
		{"missingresource/", "", "", true},
		{"", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			siteID, resourceID, err := parseCompositeID(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error for input %q", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if siteID != tt.wantSite {
				t.Errorf("expected siteID %q, got %q", tt.wantSite, siteID)
			}
			if resourceID != tt.wantResource {
				t.Errorf("expected resourceID %q, got %q", tt.wantResource, resourceID)
			}
		})
	}
}
