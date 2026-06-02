package model

type Credential struct {
	UID  string `json:"uid"`
	CID  string `json:"cid"`
	SEID string `json:"seid"`
	KID  string `json:"kid"`
}

func (c Credential) Cookie() string {
	return "UID=" + c.UID + ";CID=" + c.CID + ";SEID=" + c.SEID + ";KID=" + c.KID
}

func (c Credential) RedactedUID() string {
	if len(c.UID) < 6 {
		return "***"
	}
	return c.UID[:3] + "***" + c.UID[len(c.UID)-3:]
}
