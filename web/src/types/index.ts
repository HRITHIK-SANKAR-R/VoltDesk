export interface MessagePayload {
  id: string;
  conversation_id: string;
  sender_id: string;
  content: string;
  is_ai_draft: boolean;
  created_at: string;
}

export interface WsEvent {
  type: 'chat_message' | 'ai_smart_draft' | 'accept_ai_draft' | 'SYS_RESOLVE';
  payload: any; // Ideally we use a union here
}

export interface Conversation {
  id: string;
  customer_id: string;
  status: string;
  last_activity_at: string;
  created_at: string;
}
