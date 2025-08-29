export interface GetDataParams {
  limit: number;
  offset: number;
  talker?: string;
  time?: string;
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

export interface ChatlogItem {
  seq: number;
  time: string;
  talker: string;
  talkerName: string;
  isChatRoom: boolean;
  sender: string;
  senderName: string;
  isSelf: boolean;
  type: number;
  subType: number;
  content: string;
}