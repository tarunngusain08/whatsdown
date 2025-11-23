import { Conversation } from '../api/types';
import { formatDistanceToNow } from 'date-fns';

interface ChatListProps {
  conversations: Conversation[];
  selectedPeer: string | null;
  onSelectPeer: (peer: string) => void;
  onlineUsers: Record<string, boolean>;
}

export default function ChatList({ conversations, selectedPeer, onSelectPeer, onlineUsers }: ChatListProps) {
  const getInitials = (username: string) => {
    return username.charAt(0).toUpperCase();
  };

  const formatTime = (timestamp: string) => {
    try {
      return formatDistanceToNow(new Date(timestamp), { addSuffix: true });
    } catch {
      return 'Just now';
    }
  };

  return (
    <div className="h-full flex flex-col bg-gray-50">
      <div className="p-4 border-b border-gray-200 bg-white">
        <h2 className="text-xl font-semibold text-gray-900">Chats</h2>
      </div>
      
      <div className="flex-1 overflow-y-auto">
        {conversations.length === 0 ? (
          <div className="p-8 text-center text-gray-500">
            <p>No conversations yet</p>
            <p className="text-sm mt-2">Start a new chat to get started!</p>
          </div>
        ) : (
          <div className="divide-y divide-gray-200">
            {conversations.map((conv) => (
              <button
                key={conv.peerUsername}
                onClick={() => onSelectPeer(conv.peerUsername)}
                className={`w-full p-4 hover:bg-gray-100 transition-colors text-left ${
                  selectedPeer === conv.peerUsername ? 'bg-primary-50 border-l-4 border-primary-500' : ''
                }`}
              >
                <div className="flex items-start space-x-3">
                  <div className="relative">
                    <div className="w-12 h-12 rounded-full bg-primary-500 text-white flex items-center justify-center font-semibold text-lg">
                      {getInitials(conv.peerUsername)}
                    </div>
                    {onlineUsers[conv.peerUsername] && (
                      <div className="absolute bottom-0 right-0 w-3 h-3 bg-green-500 border-2 border-white rounded-full"></div>
                    )}
                  </div>
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center justify-between mb-1">
                      <p className="text-sm font-semibold text-gray-900 truncate">
                        {conv.peerUsername}
                      </p>
                      <span className="text-xs text-gray-500 ml-2">
                        {formatTime(conv.lastMessageTime)}
                      </span>
                    </div>
                    <p className="text-sm text-gray-600 truncate">
                      {conv.lastMessagePreview || 'No messages yet'}
                    </p>
                  </div>
                </div>
              </button>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}

