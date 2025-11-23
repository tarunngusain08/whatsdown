import { useState, useEffect } from 'react';
import { searchUsers } from '../api/http';
import { User } from '../api/types';

interface UserSearchProps {
  onSelectUser: (username: string) => void;
}

export default function UserSearch({ onSelectUser }: UserSearchProps) {
  const [query, setQuery] = useState('');
  const [results, setResults] = useState<User[]>([]);
  const [loading, setLoading] = useState(false);
  const [showResults, setShowResults] = useState(false);

  useEffect(() => {
    if (query.trim().length > 0) {
      const timeoutId = setTimeout(() => {
        searchUsers(query).then(users => {
          setResults(users);
          setLoading(false);
          setShowResults(true);
        }).catch(() => {
          setLoading(false);
        });
      }, 300);

      setLoading(true);
      return () => clearTimeout(timeoutId);
    } else {
      setResults([]);
      setShowResults(false);
    }
  }, [query]);

  const getInitials = (username: string) => {
    return username.charAt(0).toUpperCase();
  };

  return (
    <div className="relative">
      <input
        type="text"
        value={query}
        onChange={(e) => setQuery(e.target.value)}
        onFocus={() => query && setShowResults(true)}
        placeholder="Search users..."
        className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-transparent outline-none"
      />
      
      {showResults && (loading || results.length > 0) && (
        <div className="absolute z-10 w-full mt-1 bg-white border border-gray-200 rounded-lg shadow-lg max-h-60 overflow-y-auto">
          {loading ? (
            <div className="p-4 text-center text-gray-500">Searching...</div>
          ) : (
            results.map((user) => (
              <button
                key={user.username}
                onClick={() => {
                  onSelectUser(user.username);
                  setQuery('');
                  setShowResults(false);
                }}
                className="w-full p-3 hover:bg-gray-50 flex items-center space-x-3 text-left transition-colors"
              >
                <div className="relative">
                  <div className="w-8 h-8 rounded-full bg-primary-500 text-white flex items-center justify-center text-sm font-semibold">
                    {getInitials(user.username)}
                  </div>
                  {user.online && (
                    <div className="absolute bottom-0 right-0 w-2 h-2 bg-green-500 border border-white rounded-full"></div>
                  )}
                </div>
                <div className="flex-1">
                  <p className="text-sm font-medium text-gray-900">{user.username}</p>
                  <p className="text-xs text-gray-500">{user.online ? 'Online' : 'Offline'}</p>
                </div>
              </button>
            ))
          )}
        </div>
      )}
    </div>
  );
}

