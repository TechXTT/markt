package scorer

import "strings"

// Thresholds bundles the knobs ComputeVerdict uses. Kept in a struct so the
// dispatch layer can return a marketplace-specific set without scattering
// literals through the codebase.
type Thresholds struct {
	// MinComparables is the minimum comparables count required for BUY and to
	// exit the low-comparables ASK SELLER gate.
	MinComparables int
	// MinScoreForBuy is the minimum score required for BUY.
	MinScoreForBuy float64
	// MaxPriceRatioSkip triggers SKIP when priceRatio is strictly above this.
	MaxPriceRatioSkip float64
	// MaxPriceRatioNegotiate is the upper bound for the NEGOTIATE price band.
	MaxPriceRatioNegotiate float64
	// FreshnessDaysDefault is the comparable freshness limit for BUY in
	// non-low-liquidity niches.
	FreshnessDaysDefault int
	// FreshnessDaysLowLiquidity is the comparable freshness limit for BUY in
	// low-liquidity niches (camera bodies, discontinued laptop lines).
	FreshnessDaysLowLiquidity int
}

// defaultThresholds preserves the pre-XOL-36 Marktplaats-tuned behavior.
// All non-BG marketplaces fall through to this. Do not change these values
// without justifying the NL regression impact in the PR body.
var defaultThresholds = Thresholds{
	MinComparables:            6,
	MinScoreForBuy:            8.0,
	MaxPriceRatioSkip:         1.30,
	MaxPriceRatioNegotiate:    1.30,
	FreshnessDaysDefault:      60,
	FreshnessDaysLowLiquidity: 90,
}

// bgThresholds reflects OLX BG's thinner query liquidity. Per XOL-190
// (founder ruling 2026-05-08, γ implementation), the comparables floor is
// raised from 3 → 5 because <3 BG comps was producing fair=ask tautologies
// that masked verdict differentiation (audit 2026-05-07: 8 listings, 0/0/8/0
// distribution). N≥5 is the minimum for reliable Buy / Negotiate / Skip
// differentiation. N=2-4 still calculates a fair-value (informational) but
// routes through the existing MinComparables → ActionAskSeller gate at line
// 197 of verdict.go. N=0-1 gets a separate "insufficient_data" classification
// via BGComparableBucket below — fair-value should be hidden by callers in
// that bucket.
var bgThresholds = Thresholds{
	MinComparables:            5,
	MinScoreForBuy:            8.0,
	MaxPriceRatioSkip:         1.30,
	MaxPriceRatioNegotiate:    1.30,
	FreshnessDaysDefault:      60,
	FreshnessDaysLowLiquidity: 90,
}

// ThresholdsFor returns the Thresholds for the given marketplace id. Unknown
// or empty ids fall through to defaultThresholds.
func ThresholdsFor(marketplaceID string) Thresholds {
	switch normalizeMarketplaceID(marketplaceID) {
	case "olxbg":
		return bgThresholds
	default:
		return defaultThresholds
	}
}

// normalizeMarketplaceID lowercases, trims, and collapses common variants
// (olx-bg, olx.bg → olxbg). Kept defensive because ingest is not strictly
// normalized yet.
func normalizeMarketplaceID(id string) string {
	s := strings.ToLower(strings.TrimSpace(id))
	s = strings.ReplaceAll(s, "-", "")
	s = strings.ReplaceAll(s, ".", "")
	s = strings.ReplaceAll(s, "_", "")
	return s
}

// BGComparableBucket classifies a BG-marketplace verdict by comparable count.
// Per XOL-190 γ implementation (founder ruling 2026-05-08), three buckets:
//
//	"insufficient_data"  → N = 0..1   — no fair-value claim; callers should
//	                                    hide the "X% of fair value" line and
//	                                    surface "Insufficient market data —
//	                                    manual price check recommended" badge
//	"low_confidence"     → N = 2..(MinComparables-1)
//	                                  — fair-value calculated and displayed
//	                                    as informational; verdict still routes
//	                                    to ASK SELLER via the existing
//	                                    MinComparables gate at verdict.go:197;
//	                                    callers surface "Estimated fair value
//	                                    (low confidence — N comparables)" badge
//	"full_confidence"    → N >= MinComparables
//	                                  — full Buy / Negotiate / Ask / Skip
//	                                    differentiation per existing thresholds
//
// The bucket is independent of the verdict string itself — both the
// "insufficient_data" and "low_confidence" buckets resolve to ActionAskSeller
// via existing logic. The bucket exists for the wire layer to drive distinct
// dash UX per the founder's display table.
//
// Insufficient-floor is hard-coded at 2 (N=0..1 bucket) because the founder
// ruling pinned both buckets explicitly and small-N statistics are not robust
// enough to make this configurable at this point. Revisit if the BG
// comparables DB warms past expected baseline.
func BGComparableBucket(comparableCount int, marketplaceID string) string {
	t := ThresholdsFor(marketplaceID)
	if comparableCount < 2 {
		return "insufficient_data"
	}
	if comparableCount < t.MinComparables {
		return "low_confidence"
	}
	return "full_confidence"
}
