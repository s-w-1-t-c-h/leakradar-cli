package commands

import (
	"fmt"

	"github.com/spf13/pflag"

	"leakradar-cli/internal/api"
)

// cadv holds the combolist advanced filter surface — a strict subset of
// the main dataset's `adv` (advfilters.go): the combolist corpus has no
// URL data, so there's no url/url-domain/url-host/url-scheme/url-port/
// url-tld/email-host/email-tld here. Shared by `combolist advanced`,
// `combolist unlock advanced` and `combolist export advanced` via the same
// underlying pflag.FlagSet (registerComboAdvancedFlags).
var cadv struct {
	username, usernameNot                 []string
	usernameMatch, usernameNotMatch       string
	password, passwordNot                 []string
	passwordMatch, passwordNotMatch       string
	emailDomain, emailDomainNot           []string
	emailDomainMatch, emailDomainNotMatch string
	usernameHash, passwordHash            []string
	passwordStrength                      string
	addedFrom, addedTo                    string
	forceAnd                              bool
	isEmail, isUsername                   bool
}

// registerComboAdvancedFlags attaches the combolist filter flag set to cmd.
func registerComboAdvancedFlags(f *pflag.FlagSet) {
	f.StringArrayVar(&cadv.username, "username", nil, "match username (repeatable)")
	f.StringArrayVar(&cadv.usernameNot, "username-not", nil, "exclude username (repeatable)")
	f.StringVar(&cadv.usernameMatch, "username-match", "", "contains|starts_with|ends_with (default contains)")
	f.StringVar(&cadv.usernameNotMatch, "username-not-match", "", "contains|starts_with|ends_with (default contains)")

	f.StringArrayVar(&cadv.password, "password", nil, "match password (repeatable)")
	f.StringArrayVar(&cadv.passwordNot, "password-not", nil, "exclude password (repeatable)")
	f.StringVar(&cadv.passwordMatch, "password-match", "", "contains|starts_with|ends_with (default contains)")
	f.StringVar(&cadv.passwordNotMatch, "password-not-match", "", "contains|starts_with|ends_with (default contains)")

	f.StringArrayVar(&cadv.emailDomain, "email-domain", nil, "match email domain (repeatable)")
	f.StringArrayVar(&cadv.emailDomainNot, "email-domain-not", nil, "exclude email domain (repeatable)")
	f.StringVar(&cadv.emailDomainMatch, "email-domain-match", "", "contains|starts_with|ends_with (default contains)")
	f.StringVar(&cadv.emailDomainNotMatch, "email-domain-not-match", "", "contains|starts_with|ends_with (default contains)")

	f.StringArrayVar(&cadv.usernameHash, "username-hash", nil, "username SHA-1 hash prefix (repeatable)")
	f.StringArrayVar(&cadv.passwordHash, "password-hash", nil, "password SHA-1 hash prefix (repeatable)")

	f.StringVar(&cadv.passwordStrength, "password-strength", "", "too_weak|weak|medium|strong")
	f.StringVar(&cadv.addedFrom, "added-from", "", "only records indexed on/after this RFC3339 datetime")
	f.StringVar(&cadv.addedTo, "added-to", "", "only records indexed on/before this RFC3339 datetime")
	f.BoolVar(&cadv.forceAnd, "force-and", false, "require all values within each field (AND within field, default OR)")
	f.BoolVar(&cadv.isEmail, "is-email", false, "restrict identifier matches to emails only")
	f.BoolVar(&cadv.isUsername, "is-username", false, "restrict identifier matches to usernames only")
}

// buildComboAdvancedFilters validates the shared `cadv` flag values and
// builds the CombolistAdvancedSearchRequest body. Reuses matchTypePtr and
// validStrengths from advfilters.go — the validation rules are identical,
// only the available fields differ.
func buildComboAdvancedFilters() (api.CombolistAdvancedSearchRequest, error) {
	var filters api.CombolistAdvancedSearchRequest

	if cadv.isEmail && cadv.isUsername {
		return filters, fmt.Errorf("--is-email and --is-username are mutually exclusive")
	}
	if cadv.passwordStrength != "" && !validStrengths[cadv.passwordStrength] {
		return filters, fmt.Errorf("invalid --password-strength %q (must be too_weak, weak, medium, or strong)", cadv.passwordStrength)
	}

	type matchField struct {
		name string
		val  string
		dst  **api.TextMatchType
	}
	for _, mf := range []matchField{
		{"username-match", cadv.usernameMatch, &filters.UsernameMatchType},
		{"username-not-match", cadv.usernameNotMatch, &filters.UsernameNotMatchType},
		{"password-match", cadv.passwordMatch, &filters.PasswordMatchType},
		{"password-not-match", cadv.passwordNotMatch, &filters.PasswordNotMatchType},
		{"email-domain-match", cadv.emailDomainMatch, &filters.EmailDomainMatchType},
		{"email-domain-not-match", cadv.emailDomainNotMatch, &filters.EmailDomainNotMatchType},
	} {
		ptr, err := matchTypePtr(mf.name, mf.val)
		if err != nil {
			return filters, err
		}
		*mf.dst = ptr
	}

	filters.Username, filters.UsernameNot = cadv.username, cadv.usernameNot
	filters.Password, filters.PasswordNot = cadv.password, cadv.passwordNot
	filters.EmailDomain, filters.EmailDomainNot = cadv.emailDomain, cadv.emailDomainNot
	filters.UsernameHash = cadv.usernameHash
	filters.PasswordHash = cadv.passwordHash
	filters.AddedFrom, filters.AddedTo = cadv.addedFrom, cadv.addedTo
	filters.ForceAnd = cadv.forceAnd

	if cadv.passwordStrength != "" {
		s := api.PasswordStrengthCategory(cadv.passwordStrength)
		filters.PasswordStrength = &s
	}
	isEmail, err := isEmailPtr(cadv.isEmail, cadv.isUsername)
	if err != nil {
		return filters, err
	}
	filters.IsEmail = isEmail

	return filters, nil
}
