package model

// CREATE TABLE WCContact(
// m_nsUsrName TEXT PRIMARY KEY ASC,
// m_uiConType INTEGER,
// nickname TEXT,
// m_nsFullPY TEXT,
// m_nsShortPY TEXT,
// m_nsRemark TEXT,
// m_nsRemarkPYFull TEXT,
// m_nsRemarkPYShort TEXT,
// m_uiCertificationFlag INTEGER,
// m_uiSex INTEGER,
// m_uiType INTEGER,
// m_nsImgStatus TEXT,
// m_uiImgKey INTEGER,
// m_nsHeadImgUrl TEXT,
// m_nsHeadHDImgUrl TEXT,
// m_nsHeadHDMd5 TEXT,
// m_nsChatRoomMemList TEXT,
// m_nsChatRoomAdminList TEXT,
// m_uiChatRoomStatus INTEGER,
// m_nsChatRoomDesc TEXT,
// m_nsDraft TEXT,
// m_nsBrandIconUrl TEXT,
// m_nsGoogleContactName TEXT,
// m_nsAliasName TEXT,
// m_nsEncodeUserName TEXT,
// m_uiChatRoomVersion INTEGER,
// m_uiChatRoomMaxCount INTEGER,
// m_uiChatRoomType INTEGER,
// m_patSuffix TEXT,
// richChatRoomDesc TEXT,
// _packed_WCContactData BLOB,
// openIMInfo BLOB
// )
type ContactDarwinV3 struct {
	M_nsUsrName   string `json:"m_nsUsrName"`
	Nickname      string `json:"nickname"`
	M_nsRemark    string `json:"m_nsRemark"`
	M_uiSex       int    `json:"m_uiSex"`
	M_nsAliasName string `json:"m_nsAliasName"`
}

func (c *ContactDarwinV3) Wrap() *Contact {
	return &Contact{
		UserName: c.M_nsUsrName,
		Alias:    c.M_nsAliasName,
		Remark:   c.M_nsRemark,
		NickName: c.Nickname,
		IsFriend: true,
	}
}
