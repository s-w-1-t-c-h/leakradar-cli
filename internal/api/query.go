package api

import (
	"net/url"
	"strconv"
)

func newQuery() url.Values {
	return url.Values{}
}

func setStr(q url.Values, key, val string) {
	if val != "" {
		q.Set(key, val)
	}
}

func setInt(q url.Values, key string, val int) {
	if val != 0 {
		q.Set(key, strconv.Itoa(val))
	}
}

func setBool(q url.Values, key string, val bool) {
	if val {
		q.Set(key, "true")
	}
}

func setBoolPtr(q url.Values, key string, val *bool) {
	if val != nil {
		q.Set(key, strconv.FormatBool(*val))
	}
}
