export interface User {
  username: string;
  online: boolean;
}

export interface Message {
  id: string;
  from: string;
  to: string;
  content: string;
  timestamp: string;
  status: 'sent' | 'delivered';
}

export interface Conversation {
  peerUsername: string;
  lastMessagePreview: string;
  lastMessageTime: string;
  peerOnline: boolean;
  unreadCount?: number;
}

export interface WSMessage {
  type: 'message' | 'typing' | 'status' | 'ack';
  payload: any;
}

export interface OutboundMessage {
  id: string;
  from: string;
  to: string;
  content: string;
  timestamp: string;
  status: 'sent' | 'delivered';
}

export interface TypingEvent {
  from?: string;
  to?: string;
  isTyping: boolean;
}

export interface StatusEvent {
  username: string;
  online: boolean;
}

export interface AckEvent {
  messageId: string;
  status: string;
}

