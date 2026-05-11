package scorer

import "testing"

// TestThresholdsFor verifies that ThresholdsFor dispatches correctly for
// canonical ids, common variant spellings, unknown ids, and empty input.
func TestThresholdsFor(t *testing.T) {
	cases := []struct {
		name          string
		id            string
		wantMinComps  int
		wantMinScore  float64
		wantBGVariant bool // true when we expect bgThresholds
	}{
		// Canonical BG identifier. Per XOL-190 (founder ruling 2026-05-08),
		// MinComparables raised from 3 → 5 for γ implementation.
		{
			name:          "canonical_olxbg",
			id:            "olxbg",
			wantMinComps:  5,
			wantMinScore:  8.0,
			wantBGVariant: true,
		},
		// Common variant with hyphen.
		{
			name:          "variant_olx_dash_bg",
			id:            "OLX-BG",
			wantMinComps:  5,
			wantMinScore:  8.0,
			wantBGVariant: true,
		},
		// Dot-separated variant.
		{
			name:          "variant_olx_dot_bg",
			id:            "olx.bg",
			wantMinComps:  5,
			wantMinScore:  8.0,
			wantBGVariant: true,
		},
		// Unknown marketplace falls through to default.
		{
			name:          "unknown_marktplaats",
			id:            "marktplaats",
			wantMinComps:  6,
			wantMinScore:  8.0,
			wantBGVariant: false,
		},
		// Empty string falls through to default.
		{
			name:          "empty_string",
			id:            "",
			wantMinComps:  6,
			wantMinScore:  8.0,
			wantBGVariant: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ThresholdsFor(tc.id)
			if got.MinComparables != tc.wantMinComps {
				t.Errorf("ThresholdsFor(%q).MinComparables = %d, want %d", tc.id, got.MinComparables, tc.wantMinComps)
			}
			if got.MinScoreForBuy != tc.wantMinScore {
				t.Errorf("ThresholdsFor(%q).MinScoreForBuy = %f, want %f", tc.id, got.MinScoreForBuy, tc.wantMinScore)
			}
			// Verify the full struct matches the expected preset so we catch any
			// future partial-mutation bugs.
			if tc.wantBGVariant {
				if got != bgThresholds {
					t.Errorf("ThresholdsFor(%q) = %+v, want bgThresholds %+v", tc.id, got, bgThresholds)
				}
			} else {
				if got != defaultThresholds {
					t.Errorf("ThresholdsFor(%q) = %+v, want defaultThresholds %+v", tc.id, got, defaultThresholds)
				}
			}
		})
	}
}

// TestBGComparableBucket exercises the γ-implementation bucket boundaries
// (XOL-190 founder ruling 2026-05-08). N=0..1 → insufficient_data; N=2..4
// → low_confidence; N>=5 → full_confidence (BG MinComparables is 5).
func TestBGComparableBucket(t *testing.T) {
	cases := []struct {
		name          string
		count         int
		marketplaceID string
		wantBucket    string
	}{
		// BG bucket transitions at N=0,1,2,3,4,5,10.
		{"bg_n0_insufficient", 0, "olxbg", "insufficient_data"},
		{"bg_n1_insufficient", 1, "olxbg", "insufficient_data"},
		{"bg_n2_low_confidence", 2, "olxbg", "low_confidence"},
		{"bg_n3_low_confidence", 3, "olxbg", "low_confidence"},
		{"bg_n4_low_confidence", 4, "olxbg", "low_confidence"},
		{"bg_n5_full_confidence", 5, "olxbg", "full_confidence"},
		{"bg_n10_full_confidence", 10, "olxbg", "full_confidence"},
		// Non-BG marketplaces use defaultThresholds (MinComparables=6) for the
		// full_confidence floor; insufficient_data is still N<2 universally.
		{"default_n0_insufficient", 0, "marktplaats", "insufficient_data"},
		{"default_n1_insufficient", 1, "marktplaats", "insufficient_data"},
		{"default_n2_low_confidence", 2, "marktplaats", "low_confidence"},
		{"default_n5_low_confidence", 5, "marktplaats", "low_confidence"},
		{"default_n6_full_confidence", 6, "marktplaats", "full_confidence"},
		{"default_n10_full_confidence", 10, "marktplaats", "full_confidence"},
		// Negative count defensive — treat as insufficient (caller bug surfaces
		// as no fair-value claim, not a panic).
		{"defensive_negative", -1, "olxbg", "insufficient_data"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := BGComparableBucket(tc.count, tc.marketplaceID)
			if got != tc.wantBucket {
				t.Errorf("BGComparableBucket(%d, %q) = %q, want %q", tc.count, tc.marketplaceID, got, tc.wantBucket)
			}
		})
	}
}

// TestNormalizeMarketplaceID verifies individual normalization edge cases.
func TestNormalizeMarketplaceID(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"olxbg", "olxbg"},
		{"OLX-BG", "olxbg"},
		{"olx.bg", "olxbg"},
		{"OLX_BG", "olxbg"},
		{"  OLX-BG  ", "olxbg"},
		{"marktplaats", "marktplaats"},
		{"", ""},
	}
	for _, tc := range cases {
		got := normalizeMarketplaceID(tc.in)
		if got != tc.want {
			t.Errorf("normalizeMarketplaceID(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
