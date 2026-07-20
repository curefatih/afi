package controlplane

import "testing"

func TestShouldWipeSchema(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name          string
		metaExists    bool
		metaVersion   int
		metaVersionOK bool
		orgExists     bool
		orgIDType     string
		wantWipe      bool
	}{
		{
			name:     "empty database",
			wantWipe: false,
		},
		{
			name:      "legacy uuid organizations",
			orgExists: true,
			orgIDType: "uuid",
			wantWipe:  true,
		},
		{
			name:      "text orgs without meta is wipe",
			orgExists: true,
			orgIDType: "text",
			wantWipe:  true, // unknown/incomplete install → reset once
		},
		{
			name:          "matching schema version never wipes",
			metaExists:    true,
			metaVersion:   schemaVersion,
			metaVersionOK: true,
			orgExists:     true,
			orgIDType:     "text",
			wantWipe:      false,
		},
		{
			name:          "version bump does not wipe",
			metaExists:    true,
			metaVersion:   schemaVersion - 1,
			metaVersionOK: true,
			orgExists:     true,
			orgIDType:     "text",
			wantWipe:      false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := shouldWipeSchema(schemaDecision{
				MetaExists:    tc.metaExists,
				MetaVersion:   tc.metaVersion,
				MetaVersionOK: tc.metaVersionOK,
				OrgExists:     tc.orgExists,
				OrgIDDataType: tc.orgIDType,
			})
			if got != tc.wantWipe {
				t.Fatalf("shouldWipeSchema=%v want %v", got, tc.wantWipe)
			}
		})
	}
}
