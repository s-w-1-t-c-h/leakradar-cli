package commands

import (
	"fmt"

	"github.com/spf13/pflag"

	"leakradar-cli/internal/api"
)

// advFilterFlags holds the full /search/advanced filter surface, shared by
// `leakradar-cli advanced`, `leakradar-cli unlock advanced` and `leakradar-cli export
// advanced` via the same underlying pflag.FlagSet (registerAdvancedFlags),
// so all three stay in lockstep with the real LeakSearchFilters schema.
var adv struct {
	username, usernameNot                 []string
	usernameMatch, usernameNotMatch       string
	password, passwordNot                 []string
	passwordMatch, passwordNotMatch       string
	url, urlNot                           []string
	urlMatch, urlNotMatch                 string
	urlDomain, urlDomainNot               []string
	urlDomainMatch, urlDomainNotMatch     string
	urlHost, urlHostNot                   []string
	urlHostMatch, urlHostNotMatch         string
	usernameHash, passwordHash            []string
	urlScheme, urlSchemeNot               []string
	urlPort, urlPortNot                   []int
	urlTLD, urlTLDNot                     []string
	emailDomain, emailDomainNot           []string
	emailDomainMatch, emailDomainNotMatch string
	emailHost, emailHostNot               []string
	emailHostMatch, emailHostNotMatch     string
	emailTLD, emailTLDNot                 []string
	passwordStrength                      string
	addedFrom, addedTo                    string
	forceAnd                              bool
	isEmail, isUsername                   bool
}

var validMatchTypes = map[string]bool{"contains": true, "starts_with": true, "ends_with": true}
var validStrengths = map[string]bool{"too_weak": true, "weak": true, "medium": true, "strong": true}

// registerAdvancedFlags attaches the full filter flag set to cmd.
func registerAdvancedFlags(f *pflag.FlagSet) {
	f.StringArrayVar(&adv.username, "username", nil, "match username (repeatable)")
	f.StringArrayVar(&adv.usernameNot, "username-not", nil, "exclude username (repeatable)")
	f.StringVar(&adv.usernameMatch, "username-match", "", "contains|starts_with|ends_with (default contains)")
	f.StringVar(&adv.usernameNotMatch, "username-not-match", "", "contains|starts_with|ends_with (default contains)")

	f.StringArrayVar(&adv.password, "password", nil, "match password (repeatable)")
	f.StringArrayVar(&adv.passwordNot, "password-not", nil, "exclude password (repeatable)")
	f.StringVar(&adv.passwordMatch, "password-match", "", "contains|starts_with|ends_with (default contains)")
	f.StringVar(&adv.passwordNotMatch, "password-not-match", "", "contains|starts_with|ends_with (default contains)")

	f.StringArrayVar(&adv.url, "url", nil, "match URL (repeatable)")
	f.StringArrayVar(&adv.urlNot, "url-not", nil, "exclude URL (repeatable)")
	f.StringVar(&adv.urlMatch, "url-match", "", "contains|starts_with|ends_with (default contains)")
	f.StringVar(&adv.urlNotMatch, "url-not-match", "", "contains|starts_with|ends_with (default contains)")

	f.StringArrayVar(&adv.urlDomain, "url-domain", nil, "match URL domain (repeatable)")
	f.StringArrayVar(&adv.urlDomainNot, "url-domain-not", nil, "exclude URL domain (repeatable)")
	f.StringVar(&adv.urlDomainMatch, "url-domain-match", "", "contains|starts_with|ends_with (default contains)")
	f.StringVar(&adv.urlDomainNotMatch, "url-domain-not-match", "", "contains|starts_with|ends_with (default contains)")

	f.StringArrayVar(&adv.urlHost, "url-host", nil, "match URL host (repeatable)")
	f.StringArrayVar(&adv.urlHostNot, "url-host-not", nil, "exclude URL host (repeatable)")
	f.StringVar(&adv.urlHostMatch, "url-host-match", "", "contains|starts_with|ends_with (default contains)")
	f.StringVar(&adv.urlHostNotMatch, "url-host-not-match", "", "contains|starts_with|ends_with (default contains)")

	f.StringArrayVar(&adv.usernameHash, "username-hash", nil, "username SHA-1 hash prefix (repeatable)")
	f.StringArrayVar(&adv.passwordHash, "password-hash", nil, "password SHA-1 hash prefix (repeatable)")

	f.StringArrayVar(&adv.urlScheme, "url-scheme", nil, "match URL scheme, e.g. https (repeatable)")
	f.StringArrayVar(&adv.urlSchemeNot, "url-scheme-not", nil, "exclude URL scheme (repeatable)")
	f.IntSliceVar(&adv.urlPort, "url-port", nil, "match URL port (repeatable)")
	f.IntSliceVar(&adv.urlPortNot, "url-port-not", nil, "exclude URL port (repeatable)")
	f.StringArrayVar(&adv.urlTLD, "url-tld", nil, "match URL TLD (repeatable)")
	f.StringArrayVar(&adv.urlTLDNot, "url-tld-not", nil, "exclude URL TLD (repeatable)")

	f.StringArrayVar(&adv.emailDomain, "email-domain", nil, "match email domain (repeatable)")
	f.StringArrayVar(&adv.emailDomainNot, "email-domain-not", nil, "exclude email domain (repeatable)")
	f.StringVar(&adv.emailDomainMatch, "email-domain-match", "", "contains|starts_with|ends_with (default contains)")
	f.StringVar(&adv.emailDomainNotMatch, "email-domain-not-match", "", "contains|starts_with|ends_with (default contains)")

	f.StringArrayVar(&adv.emailHost, "email-host", nil, "match email host (repeatable)")
	f.StringArrayVar(&adv.emailHostNot, "email-host-not", nil, "exclude email host (repeatable)")
	f.StringVar(&adv.emailHostMatch, "email-host-match", "", "contains|starts_with|ends_with (default contains)")
	f.StringVar(&adv.emailHostNotMatch, "email-host-not-match", "", "contains|starts_with|ends_with (default contains)")

	f.StringArrayVar(&adv.emailTLD, "email-tld", nil, "match email TLD (repeatable)")
	f.StringArrayVar(&adv.emailTLDNot, "email-tld-not", nil, "exclude email TLD (repeatable)")

	f.StringVar(&adv.passwordStrength, "password-strength", "", "too_weak|weak|medium|strong")
	f.StringVar(&adv.addedFrom, "added-from", "", "only leaks indexed on/after this RFC3339 datetime")
	f.StringVar(&adv.addedTo, "added-to", "", "only leaks indexed on/before this RFC3339 datetime")
	f.BoolVar(&adv.forceAnd, "force-and", false, "require all values within each field (AND within field, default OR)")
	f.BoolVar(&adv.isEmail, "is-email", false, "restrict identifier matches to emails only")
	f.BoolVar(&adv.isUsername, "is-username", false, "restrict identifier matches to usernames only")
}

func matchTypePtr(name, val string) (*api.TextMatchType, error) {
	if val == "" {
		return nil, nil
	}
	if !validMatchTypes[val] {
		return nil, fmt.Errorf("invalid --%s %q (must be contains, starts_with, or ends_with)", name, val)
	}
	t := api.TextMatchType(val)
	return &t, nil
}

// buildAdvancedFilters validates the shared `adv` flag values and builds
// the LeakSearchFilters body sent to /search/advanced and its unlock/export
// variants.
func buildAdvancedFilters() (api.LeakSearchFilters, error) {
	var filters api.LeakSearchFilters

	if adv.isEmail && adv.isUsername {
		return filters, fmt.Errorf("--is-email and --is-username are mutually exclusive")
	}
	if adv.passwordStrength != "" && !validStrengths[adv.passwordStrength] {
		return filters, fmt.Errorf("invalid --password-strength %q (must be too_weak, weak, medium, or strong)", adv.passwordStrength)
	}

	type matchField struct {
		name string
		val  string
		dst  **api.TextMatchType
	}
	for _, mf := range []matchField{
		{"username-match", adv.usernameMatch, &filters.UsernameMatchType},
		{"username-not-match", adv.usernameNotMatch, &filters.UsernameNotMatchType},
		{"password-match", adv.passwordMatch, &filters.PasswordMatchType},
		{"password-not-match", adv.passwordNotMatch, &filters.PasswordNotMatchType},
		{"url-match", adv.urlMatch, &filters.URLMatchType},
		{"url-not-match", adv.urlNotMatch, &filters.URLNotMatchType},
		{"url-domain-match", adv.urlDomainMatch, &filters.URLDomainMatchType},
		{"url-domain-not-match", adv.urlDomainNotMatch, &filters.URLDomainNotMatchType},
		{"url-host-match", adv.urlHostMatch, &filters.URLHostMatchType},
		{"url-host-not-match", adv.urlHostNotMatch, &filters.URLHostNotMatchType},
		{"email-domain-match", adv.emailDomainMatch, &filters.EmailDomainMatchType},
		{"email-domain-not-match", adv.emailDomainNotMatch, &filters.EmailDomainNotMatchType},
		{"email-host-match", adv.emailHostMatch, &filters.EmailHostMatchType},
		{"email-host-not-match", adv.emailHostNotMatch, &filters.EmailHostNotMatchType},
	} {
		ptr, err := matchTypePtr(mf.name, mf.val)
		if err != nil {
			return filters, err
		}
		*mf.dst = ptr
	}

	filters.Username, filters.UsernameNot = adv.username, adv.usernameNot
	filters.Password, filters.PasswordNot = adv.password, adv.passwordNot
	filters.URL, filters.URLNot = adv.url, adv.urlNot
	filters.URLDomain, filters.URLDomainNot = adv.urlDomain, adv.urlDomainNot
	filters.URLHost, filters.URLHostNot = adv.urlHost, adv.urlHostNot
	filters.UsernameHash = adv.usernameHash
	filters.PasswordHash = adv.passwordHash
	filters.URLScheme, filters.URLSchemeNot = adv.urlScheme, adv.urlSchemeNot
	filters.URLPort, filters.URLPortNot = adv.urlPort, adv.urlPortNot
	filters.URLTLD, filters.URLTLDNot = adv.urlTLD, adv.urlTLDNot
	filters.EmailDomain, filters.EmailDomainNot = adv.emailDomain, adv.emailDomainNot
	filters.EmailHost, filters.EmailHostNot = adv.emailHost, adv.emailHostNot
	filters.EmailTLD, filters.EmailTLDNot = adv.emailTLD, adv.emailTLDNot
	filters.AddedFrom, filters.AddedTo = adv.addedFrom, adv.addedTo

	if adv.forceAnd {
		v := true
		filters.ForceAnd = &v
	}
	if adv.passwordStrength != "" {
		s := api.PasswordStrengthCategory(adv.passwordStrength)
		filters.PasswordStrength = &s
	}
	if isEmail, err := isEmailPtr(adv.isEmail, adv.isUsername); err != nil {
		return filters, err
	} else {
		filters.IsEmail = isEmail
	}

	return filters, nil
}
