import { User, Conversation, Message } from './types';

const API_BASE = '/api';

export async function login(username: string): Promise<User> {
  const response = await fetch(`${API_BASE}/login`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ username }),
  });

  if (!response.ok) {
    const error = await response.text();
    throw new Error(error || 'Login failed');
  }

  return response.json();
}

export async function logout(): Promise<void> {
  await fetch(`${API_BASE}/logout`, {
    method: 'POST',
  });
}

export async function getMe(): Promise<User> {
  const response = await fetch(`${API_BASE}/me`);
  if (!response.ok) {
    throw new Error('Not authenticated');
  }
  return response.json();
}

export async function searchUsers(query: string): Promise<User[]> {
  const response = await fetch(`${API_BASE}/users?search=${encodeURIComponent(query)}`);
  if (!response.ok) {
    throw new Error('Failed to search users');
  }
  return response.json();
}

export async function getConversations(): Promise<Conversation[]> {
  const response = await fetch(`${API_BASE}/conversations`);
  if (!response.ok) {
    throw new Error('Failed to fetch conversations');
  }
  return response.json();
}

export async function getConversation(peerUsername: string): Promise<Message[]> {
  const response = await fetch(`${API_BASE}/conversations/${encodeURIComponent(peerUsername)}`);
  if (!response.ok) {
    throw new Error('Failed to fetch conversation');
  }
  return response.json();
}

