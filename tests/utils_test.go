package tests

import (
	"regexp"
	"testing"
)

var (
	cases = []struct {
		mail  string
		valid bool
	}{
		{mail: "admin@mails.tsinghua.edu.cn", valid: true},
		{mail: "thu-hole@mails.tsinghua.edu.cn", valid: true},
		{mail: "thu_hole@mails.tsinghua.edu.cn", valid: true},
		{mail: "yezhisheng@pku.edu.cn,admin@mails.tsinghua.edu.cn", valid: false},
	}
)

func checkEmail(email string) bool {
	// REF: https://html.spec.whatwg.org/multipage/input.html#valid-e-mail-address
	var emailRegexp = regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+\\/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")
	return emailRegexp.MatchString(email)
}

func TestCheckMail(t *testing.T) {
	for _, c := range cases {
		if checkEmail(c.mail) != c.valid {
			t.Errorf("%s is expected to be %v", c.mail, c.valid)
		}
	}
}
