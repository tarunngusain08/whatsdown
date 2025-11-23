import { useNavigate } from 'react-router-dom';
import { useApp } from '../store/AppContext';
import { logout } from '../api/http';

export default function TopBar() {
  const { currentUser, wsClient, reconnectStatus } = useApp();
  const navigate = useNavigate();

  const handleLogout = async () => {
    try {
      if (wsClient) {
        wsClient.disconnect();
      }
      await logout();
      navigate('/');
      window.location.reload(); // Clear all state
    } catch (error) {
      console.error('Logout failed:', error);
    }
  };

  return (
    <div className="h-16 bg-white border-b border-gray-200 flex items-center justify-between px-4">
      <div className="flex items-center space-x-4">
        <h1 className="text-xl font-bold text-gray-900">WhatsDown</h1>
        {reconnectStatus && (
          <div className="text-sm text-orange-600 bg-orange-50 px-3 py-1 rounded-full">
            {reconnectStatus}
          </div>
        )}
      </div>
      <div className="flex items-center space-x-4">
        {currentUser && (
          <span className="text-sm text-gray-600">@{currentUser.username}</span>
        )}
        <button
          onClick={handleLogout}
          className="px-4 py-2 text-sm text-red-600 hover:bg-red-50 rounded-lg transition-colors"
        >
          Logout
        </button>
      </div>
    </div>
  );
}

