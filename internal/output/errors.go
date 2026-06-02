package output

import "strings"

func RedactSecret(s string) string {
	if s == "" {
		return ""
	}
	replacers := []string{"UID=", "CID=", "SEID=", "KID=", "Authorization=", "Cookie="}
	out := s
	for _, marker := range replacers {
		idx := strings.Index(out, marker)
		if idx >= 0 {
			end := strings.Index(out[idx:], ";")
			if end < 0 {
				out = out[:idx+len(marker)] + "***"
			} else {
				out = out[:idx+len(marker)] + "***" + out[idx+end:]
			}
		}
	}
	return out
}
