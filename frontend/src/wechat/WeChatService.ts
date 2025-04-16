import { http } from "@/utils/Http";
import { ChatlogItem, ChatRoomData, ChatSessionData, ContactData, GetDataParams } from "./typing";

export async function getContact(params: GetDataParams): Promise<ContactData> {
  try {
    const response = await http.get<ContactData>('/api/v1/contact?format=json', { params });
    return response.data;
  } catch (error) {
    console.error('Error fetching contact:', error);
    throw error;
  }
}

export async function getChatRoom(params: GetDataParams): Promise<ChatRoomData> {
  try {
    const response = await http.get<ChatRoomData>('/api/v1/chatroom?format=json', { params });
    return response.data;
  } catch (error) {
    console.error('Error fetching chat room:', error);
    throw error;
  }
}

export async function getChatSessions(params: GetDataParams): Promise<ChatSessionData> {
  try {
    const response = await http.get<ChatSessionData>('/api/v1/session?format=json', { params });
    return response.data;
  } catch (error) {
    console.error('Error fetching chat sessions:', error);
    throw error;
  }
}

export async function getChatlog(params: GetDataParams): Promise<ChatlogItem[]> {
  try {
    const response = await http.get<ChatlogItem[]>('/api/v1/chatlog?format=json', { params });
    return response.data;
  } catch (error) {
    console.error('Error fetching chat logs:', error);
    throw error;
  }
}
