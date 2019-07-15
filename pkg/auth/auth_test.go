package auth

import (
	"testing"
)

func TestGetServicePrincipalToken(t *testing.T) {
	_, err := GetServicePrincipalToken("tid", "cid", "", "")
	if err == nil {
		t.Fatal("should be error with empty secret")
	}
}
