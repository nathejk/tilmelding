package payment

import (
	"crypto/rand"
	"fmt"

	"github.com/nathejk/shared-go/types"
)

// Payment references are shown to humans: they appear in the MobilePay portal
// when an operator reconciles a payment, in the /callback/mobilepay/:ref URL and
// in the /betaling/:ref page. A raw UUID is 36 characters of noise there — it
// fills the portal's column and says nothing about what it is.
//
// The format is a namespace, the season, and 40 random bits:
//
//	NH26-7K3M9F2T
//
// so the portal shows at a glance that a payment is Nathejk's and which year it
// belongs to, in 13 characters instead of 36.
//
// # Alphabet
//
// Crockford base32 — the digits plus the uppercase letters except I, L, O and U.
// Excluding those means a reference cannot be misread between 1/I/L or 0/O when
// it is read off a screen, copied into a support ticket or spelled out over the
// phone, and dropping U keeps accidental profanity out of a customer-visible
// identifier.
const referenceAlphabet = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

// referenceRandomChars is the length of the random tail, and the number worth
// arguing about.
//
// A duplicate reference does not fail loudly: the projector upserts on the
// primary key, so a second payment with the same reference silently overwrites
// the first. And because GET /api/payment/:ref is unauthenticated, the reference
// is effectively a bearer token for one payment's details — so it must also be
// far too sparse to enumerate.
//
// 8 characters is 40 bits, ~1.1e12 values. At the observed ~700 payments a year
// the chance of any collision is about 1 in 4.5 million per year (1 in ~90,000
// over fifty years), and finding a valid reference by guessing takes ~1e9
// attempts. Six characters would be 1 in 4,400 per year, which is not a risk
// worth taking against a silent overwrite.
const referenceRandomChars = 8

// newReference returns a fresh payment reference for the given season.
//
// It is unpredictable by construction: the tail comes from crypto/rand, never
// from a counter, a timestamp or anything derived from the payment. A guessable
// reference would expose the payer's email through the unauthenticated show
// endpoint and let a stranger trigger the capture callback.
func newReference(year types.YearSlug) (string, error) {
	tail, err := randomBase32(referenceRandomChars)
	if err != nil {
		return "", err
	}
	return "NH" + seasonPart(year) + "-" + tail, nil
}

// seasonPart shortens a year slug to the two digits that distinguish it. A slug
// that is not the expected four digits is passed through rather than truncated
// blindly, so a malformed year shows up in the reference instead of silently
// becoming something else.
func seasonPart(year types.YearSlug) string {
	if len(year) == 4 {
		return string(year[2:])
	}
	return string(year)
}

// randomBase32 returns n characters drawn uniformly from referenceAlphabet.
//
// Bits are consumed exactly: five random bytes yield forty bits, which is eight
// five-bit indices with nothing left over and no modulo bias to reason about.
func randomBase32(n int) (string, error) {
	bits := n * 5
	buf := make([]byte, (bits+7)/8)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("payment: reading random bytes for reference: %w", err)
	}

	out := make([]byte, n)
	for i := range n {
		// The i-th five-bit group, read big-endian across buf.
		start := i * 5
		hi := start / 8
		shift := start % 8
		chunk := uint16(buf[hi]) << 8
		if hi+1 < len(buf) {
			chunk |= uint16(buf[hi+1])
		}
		idx := (chunk >> (11 - shift)) & 0x1f
		out[i] = referenceAlphabet[idx]
	}
	return string(out), nil
}
