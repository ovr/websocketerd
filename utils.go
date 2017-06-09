package main

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"regexp"
	"strconv"
	"strings"
)

func RawUrlDecode(str string) string {
	re := regexp.MustCompile(`(?Ui)%[0-9A-F]{2}`)
	str = re.ReplaceAllStringFunc(str, func(s string) string {
		b, err := hex.DecodeString(s[1:])
		if err == nil {
			return string(b)
		}
		return s
	})
	return str
}

func parseAutoLoginToken(token string) (*AutoLoginToken, error) {
	var err error

	tokenValue := RawUrlDecode(token)

	parts := strings.Split(tokenValue, ",")
	if len(parts) != 3 {
		return nil, errors.New("Wrong login token")
	}

	_, err = strconv.ParseUint(parts[0], 10, 64)
	if err != nil {
		return nil, err
	}

	loginToken := &AutoLoginToken{
		UserId:      json.Number(parts[0]),
		Token:       parts[1],
		BrowserHash: parts[2],
	}

	return loginToken, nil
}
