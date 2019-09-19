package utils

import "regexp"

// RedactClientID redacts client id
func RedactClientID(clientID string) string {
	return redact(clientID, "$1##### REDACTED #####$3")
}

func redact(src, repl string) string {
	r, _ := regexp.Compile("^(\\S{4})(\\S|\\s)*(\\S{4})$")
	return r.ReplaceAllString(src, repl)
}
