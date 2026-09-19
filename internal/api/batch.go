package api

import "context"

// MaxBatchSize is the API's per-request cap for locked-exists checks.
const MaxBatchSize = 100

// EmailsLockedExists calls POST /search/emails/locked-exists. Callers must
// chunk input to MaxBatchSize entries; see cmd batch helpers for chunking.
func (c *Client) EmailsLockedExists(ctx context.Context, emails []string, includeCounts bool) ([]EmailMassResult, error) {
	var out EmailMassResponse
	err := c.doJSON(ctx, requestOpts{
		method: "POST",
		path:   "/search/emails/locked-exists",
		body:   EmailMassRequest{Emails: emails, IncludeCounts: includeCounts},
	}, &out)
	if err != nil {
		return nil, err
	}
	return out.Results, nil
}

// DomainsLockedExists calls POST /search/domains/locked-exists.
func (c *Client) DomainsLockedExists(ctx context.Context, domains []string, includeCounts bool) ([]DomainMassResult, error) {
	var out DomainMassResponse
	err := c.doJSON(ctx, requestOpts{
		method: "POST",
		path:   "/search/domains/locked-exists",
		body:   DomainMassRequest{Domains: domains, IncludeCounts: includeCounts},
	}, &out)
	if err != nil {
		return nil, err
	}
	return out.Results, nil
}
