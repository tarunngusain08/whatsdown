import { createContext, useContext, useState, useEffect, useCallback, ReactNode } from 'react';
import { User, Message, Conversation } from '../api/types';
import { getMe, getConversations, getConversation } from '../api/http';
import { WebSocketClient } from '../api/websocket';
import { OutboundMessage, TypingEvent, StatusEvent, AckEvent } from '../api/types';

interface AppContextType {
  currentUser: User | null;
  setCurrentUser: (user: User | null) => void;
  conversations: Conversation[];
  setConversations: (conversations: Conversation[]) => void;
  messages: Record<string, Message[]>;
  setMessages: (peer: string, messages: Message[]) => void;
  addMessage: (message: Message) => void;
  updateMessageStatus: (messageId: string, status: 'sent' | 'delivered') => void;
  typingUsers: Record<string, boolean>;
  setTyping: (username: string, isTyping: boolean) => void;
  onlineUsers: Record<string, boolean>;
  setOnlineStatus: (username: string, online: boolean) => void;
  wsClient: WebSocketClient | null;
  selectedPeer: string | null;
  setSelectedPeer: (peer: string | null) => void;
  loading: boolean;
  setLoading: (loading: boolean) => void;
  reconnectStatus: string | null;
}

const AppContext = createContext<AppContextType | undefined>(undefined);

export function AppProvider({ children }: { children: ReactNode }) {
  const [currentUser, setCurrentUser] = useState<User | null>(null);
  const [conversations, setConversations] = useState<Conversation[]>([]);
  const [messages, setMessagesState] = useState<Record<string, Message[]>>({});
  const [typingUsers, setTypingUsers] = useState<Record<string, boolean>>({});
  const [onlineUsers, setOnlineUsers] = useState<Record<string, boolean>>({});
  const [wsClient, setWsClient] = useState<WebSocketClient | null>(null);
  const [selectedPeer, setSelectedPeer] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [reconnectStatus, setReconnectStatus] = useState<string | null>(null);

  const setMessages = (peer: string, msgs: Message[]) => {
    setMessagesState(prev => ({ ...prev, [peer]: msgs }));
  };

  const addMessage = (message: Message) => {
    const peer = message.from === currentUser?.username ? message.to : message.from;
    console.log('Adding message:', message, 'peer:', peer, 'currentUser:', currentUser?.username);
    setMessagesState(prev => {
      const existing = prev[peer] || [];
      // Check if message already exists by ID
      if (existing.some(m => m.id === message.id)) {
        console.log('Message already exists, skipping:', message.id);
        return prev;
      }
      // If this is a message we sent, check if there's a temp message with same content to replace
      if (message.from === currentUser?.username) {
        const tempIndex = existing.findIndex(m => 
          m.id.startsWith('temp-') && 
          m.content === message.content &&
          Math.abs(new Date(m.timestamp).getTime() - new Date(message.timestamp).getTime()) < 5000
        );
        if (tempIndex >= 0) {
          // Replace temp message with real one
          const updated = [...existing];
          updated[tempIndex] = message;
          return { ...prev, [peer]: updated };
        }
      }
      return { ...prev, [peer]: [...existing, message] };
    });
  };

  const updateMessageStatus = (messageId: string, status: 'sent' | 'delivered') => {
    setMessagesState(prev => {
      const updated = { ...prev };
      for (const peer in updated) {
        updated[peer] = updated[peer].map(msg =>
          msg.id === messageId ? { ...msg, status } : msg
        );
      }
      return updated;
    });
  };

  const setTyping = (username: string, isTyping: boolean) => {
    setTypingUsers(prev => ({ ...prev, [username]: isTyping }));
    if (!isTyping) {
      // Clear typing indicator after 3 seconds
      setTimeout(() => {
        setTypingUsers(prev => {
          const updated = { ...prev };
          delete updated[username];
          return updated;
        });
      }, 3000);
    }
  };

  const setOnlineStatus = (username: string, online: boolean) => {
    setOnlineUsers(prev => ({ ...prev, [username]: online }));
  };

  const refreshConversations = useCallback(async () => {
    if (!currentUser) return;
    try {
      const convs = await getConversations();
      setConversations(convs);
    } catch (error) {
      console.error('Failed to refresh conversations:', error);
    }
  }, [currentUser]);

  const refreshData = useCallback(async () => {
    if (!currentUser) return;
    try {
      await refreshConversations();
      if (selectedPeer) {
        const msgs = await getConversation(selectedPeer);
        setMessages(selectedPeer, msgs);
      }
    } catch (error) {
      console.error('Failed to refresh data:', error);
    }
  }, [currentUser, selectedPeer, refreshConversations, setMessages]);

  // Initialize WebSocket when user is logged in
  useEffect(() => {
    if (!currentUser || wsClient) {
      return;
    }

    const client = new WebSocketClient();
    
    client.onConnect(() => {
      setReconnectStatus(null);
      // Refresh conversations and messages
      refreshData();
    });

    client.onDisconnect(() => {
      setReconnectStatus('Disconnected');
    });

    client.onReconnecting(() => {
      setReconnectStatus('Reconnecting...');
    });

    client.onMessage((msg: OutboundMessage) => {
      console.log('Received message in AppContext:', msg);
      addMessage(msg as Message);
      // Refresh conversations to update last message and show new conversations
      refreshConversations();
    });

    client.onTyping((event: TypingEvent) => {
      if (event.from) {
        setTyping(event.from, event.isTyping);
      }
    });

    client.onStatus((event: StatusEvent) => {
      setOnlineStatus(event.username, event.online);
      // Update conversations to reflect online status
      getConversations().then(convs => {
        setConversations(convs);
      }).catch(err => console.error('Failed to refresh conversations:', err));
    });

    client.onAck((event: AckEvent) => {
      updateMessageStatus(event.messageId, event.status as 'sent' | 'delivered');
    });

    setWsClient(client);
    client.connect().catch(err => {
      console.error('Failed to connect WebSocket:', err);
    });

    return () => {
      client.disconnect();
      setWsClient(null);
    };
  }, [currentUser]); // Only depend on currentUser, not wsClient

  // Load user on mount
  useEffect(() => {
    getMe()
      .then(user => {
        setCurrentUser(user);
      })
      .catch(() => {
        // Not logged in, ignore
      });
  }, []);

  return (
    <AppContext.Provider
      value={{
        currentUser,
        setCurrentUser,
        conversations,
        setConversations,
        messages,
        setMessages,
        addMessage,
        updateMessageStatus,
        typingUsers,
        setTyping,
        onlineUsers,
        setOnlineStatus,
        wsClient,
        selectedPeer,
        setSelectedPeer,
        loading,
        setLoading,
        reconnectStatus,
      }}
    >
      {children}
    </AppContext.Provider>
  );
}

export function useApp() {
  const context = useContext(AppContext);
  if (context === undefined) {
    throw new Error('useApp must be used within an AppProvider');
  }
  return context;
}

