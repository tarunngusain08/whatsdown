import { Message } from '../api/types';
import { useApp } from '../store/AppContext';
import { format } from 'date-fns';

interface MessageBubbleProps {
  message: Message;
}

export default function MessageBubble({ message }: MessageBubbleProps) {
  const { currentUser } = useApp();
  const isOwn = message.from === currentUser?.username;

  const formatTime = (timestamp: string) => {
    try {
      return format(new Date(timestamp), 'HH:mm');
    } catch {
      return '';
    }
  };

  return (
    <div className={`flex ${isOwn ? 'justify-end' : 'justify-start'}`}>
      <div
        className={`max-w-xs lg:max-w-md px-4 py-2 rounded-lg ${
          isOwn
            ? 'bg-primary-600 text-white rounded-br-none'
            : 'bg-white text-gray-900 border border-gray-200 rounded-bl-none'
        }`}
      >
        <p className="text-sm break-words">{message.content}</p>
        <div className={`flex items-center justify-end mt-1 space-x-2 ${
          isOwn ? 'text-primary-100' : 'text-gray-500'
        }`}>
          <span className="text-xs">{formatTime(message.timestamp)}</span>
          {isOwn && (
            <span className="text-xs">
              {message.status === 'delivered' ? '✓✓' : '✓'}
            </span>
          )}
        </div>
      </div>
    </div>
  );
}

