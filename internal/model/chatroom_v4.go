package model

// CREATE TABLE chat_room(
// id INTEGER PRIMARY KEY,
// username TEXT,
// owner TEXT,
// ext_buffer BLOB
// )
type ChatRoomV4 struct {
	ID        int    `json:"id"`
	UserName  string `json:"username"`
	Owner     string `json:"owner"`
	ExtBuffer []byte `json:"ext_buffer"`
}

func (c *ChatRoomV4) Wrap() *ChatRoom {

	var users []ChatRoomUser
	if len(c.ExtBuffer) != 0 {
		users = ParseRoomData(c.ExtBuffer)
	}

	user2DisplayName := make(map[string]string, len(users))
	for _, user := range users {
		if user.DisplayName != "" {
			user2DisplayName[user.UserName] = user.DisplayName
		}
	}

	return &ChatRoom{
		Name:             c.UserName,
		Owner:            c.Owner,
		Users:            users,
		User2DisplayName: user2DisplayName,
	}
}
