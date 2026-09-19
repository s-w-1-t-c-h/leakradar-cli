package api

import "context"

// PasswordRange calls GET /password-range for a k-anonymity style SHA-1
// prefix lookup (like haveibeenpwned's Pwned Passwords range API).
func (c *Client) PasswordRange(ctx context.Context, prefix string, limit int, suffixOnly bool) (*PasswordRangeResponse, error) {
	q := newQuery()
	setStr(q, "prefix", prefix)
	setInt(q, "limit", limit)
	setBool(q, "suffix_only", suffixOnly)

	var out PasswordRangeResponse
	err := c.doJSON(ctx, requestOpts{
		method: "GET",
		path:   "/password-range",
		query:  q,
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}
