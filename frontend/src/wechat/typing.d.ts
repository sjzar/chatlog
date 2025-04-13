export interface GetDataParams {
  limit: number;
  offset: number;
}

export interface ContactData {
  items: ContactItem[];
}

export interface ContactItem {
  userName: string;
  alias: string;
  remark: string;
  nickName: string;
  isFriend: boolean;
}

export interface ChatRoomData {
  items: ChatRoomItem[];
}

export interface ChatRoomItem {
  name: string;
  nickName: string;
  owner: string;
  remark: string;
  users: ChatRoomUser[];
}

export interface ChatRoomUser {
  userName: string;
  displayName: string;
}

export interface ChatSessionData {
  items: ChatSessionItem[];
}

export interface ChatSessionItem {
  userName: string;
  nickName: string;
  nOrder: string;
  content: string;
  nTime: string;
}
