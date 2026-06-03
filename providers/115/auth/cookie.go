package auth

import (
	"errors"
	"net/http"
	"strings"

	model115 "github.com/onetreegrove/panctl/providers/115/model"
)

func ParseCookie(raw string) (model115.Credential, error) {
	header := http.Header{}
	header.Add("Cookie", raw)
	req := http.Request{Header: header}
	values := map[string]string{}
	for _, c := range req.Cookies() {
		values[strings.ToUpper(c.Name)] = c.Value
	}
	cred := model115.Credential{
		UID:  values["UID"],
		CID:  values["CID"],
		SEID: values["SEID"],
		KID:  values["KID"],
	}
	if cred.UID == "" || cred.CID == "" || cred.SEID == "" || cred.KID == "" {
		return model115.Credential{}, errors.New("cookie must contain UID, CID, SEID, and KID")
	}
	return cred, nil
}
