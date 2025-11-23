import { useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { useApp } from '../store/AppContext';
import { getConversation, getConversations } from '../api/http';
import TopBar from '../components/TopBar';
import ChatList from '../components/ChatList';
import ChatWindow from '../components/ChatWindow';
import UserSearch from '../components/UserSearch';

export default function ChatPage() {
  const {
    currentUser,
    conversations,
    setConversations,
    messages,
    setMessages,
    addMessage,
    wsClient,
    selectedPeer,
    setSelectedPeer,
    typingUsers,
    onlineUsers,
  } = useApp();
  const navigate = useNavigate();

  useEffect(() => {
    if (!currentUser) {
      navigate('/');
      return;
    }
    // Load conversations on mount
    getConversations()
      .then(convs => {
        setConversations(convs);
      })
      .catch(error => {
        console.error('Failed to load conversations:', error);
      });
  }, [currentUser, navigate, setConversations]);

  useEffect(() => {
    if (selectedPeer && !messages[selectedPeer]) {
      // Load conversation messages
      getConversation(selectedPeer)
        .then(msgs => {
          setMessages(selectedPeer, msgs);
        })
        .catch(error => {
          console.error('Failed to load conversation:', error);
        });
    }
  }, [selectedPeer, messages, setMessages]);

  const handleSendMessage = (content: string) => {
    if (!selectedPeer || !wsClient) return;

    // Optimistically add message
    const tempMessage = {
      id: `temp-${Date.now()}`,
      from: currentUser!.username,
      to: selectedPeer,
      content,
      timestamp: new Date().toISOString(),
      status: 'sent' as const,
    };
    addMessage(tempMessage);

    // Send via WebSocket
    wsClient.sendMessage(selectedPeer, content, tempMessage.id);
  };

  const handleTyping = (isTyping: boolean) => {
    if (!selectedPeer || !wsClient) return;
    wsClient.sendTyping(selectedPeer, isTyping);
  };

  const handleSelectPeer = (peer: string) => {
    setSelectedPeer(peer);
    if (!messages[peer]) {
      getConversation(peer)
        .then(msgs => {
          setMessages(peer, msgs);
        })
        .catch(error => {
          console.error('Failed to load conversation:', error);
        });
    }
  };

  if (!currentUser) {
    return null;
  }

  return (
    <div className="h-screen flex flex-col bg-gray-100">
      <TopBar />
      
      <div className="flex-1 flex overflow-hidden">
        {/* Left Panel - Chat List */}
        <div className="w-full md:w-80 lg:w-96 flex flex-col border-r border-gray-200 bg-white">
          <div className="p-4 border-b border-gray-200">
            <UserSearch onSelectUser={handleSelectPeer} />
          </div>
          <div className="flex-1 overflow-hidden">
            <ChatList
              conversations={conversations}
              selectedPeer={selectedPeer}
              onSelectPeer={handleSelectPeer}
              onlineUsers={onlineUsers}
            />
          </div>
        </div>

        {/* Right Panel - Chat Window */}
        <div className="flex-1 flex flex-col min-w-0">
          {selectedPeer ? (
            <ChatWindow
              peerUsername={selectedPeer}
              messages={messages[selectedPeer] || []}
              onSendMessage={handleSendMessage}
              onTyping={handleTyping}
              typing={typingUsers[selectedPeer] || false}
              online={onlineUsers[selectedPeer] || false}
            />
          ) : (
            <div className="flex-1 flex items-center justify-center bg-gray-50">
              <div className="text-center text-gray-500">
                <p className="text-lg font-medium mb-2">Select a conversation</p>
                <p className="text-sm">Choose a chat from the list or search for a user</p>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

