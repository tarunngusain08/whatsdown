# WhatsDown - 1:1 Real-time Chat Webapp

A minimal but polished full-stack web application for 1:1 direct messaging with real-time WebSocket communication.

## Features

- **Username-based authentication** with session management
- **Single session enforcement** - prevents multiple active sessions for the same username
- **User search** - find and start conversations with other users
- **Real-time messaging** via WebSockets
- **Typing indicators** - see when someone is typing
- **Message status** - sent and delivered acknowledgments
- **Online status** - see who's online in real-time
- **Conversation list** - view all your chats with latest messages
- **Beautiful, responsive UI** with smooth animations and transitions

## Tech Stack

- **Backend**: Go (Golang) with standard library + gorilla/websocket
- **Frontend**: React + TypeScript + Vite + TailwindCSS
- **Transport**: WebSockets for real-time chat, HTTP for API and static assets
- **State**: In-memory storage (no database)
- **Deployment**: Single multi-stage Dockerfile

## Project Structure

```
whatsdown/
├── cmd/
│   └── server/
│       └── main.go          # Application entry point
├── internal/
│   ├── models/
│   │   └── models.go        # Data models
│   └── server/
│       ├── hub.go           # WebSocket hub and message routing
│       ├── client.go        # WebSocket client handling
│       └── http.go          # HTTP handlers and session management
├── frontend/
│   ├── src/
│   │   ├── api/            # HTTP and WebSocket clients
│   │   ├── components/     # React components
│   │   ├── pages/          # Page components
│   │   ├── store/          # Context/state management
│   │   └── main.tsx        # Frontend entry point
│   ├── package.json
│   └── vite.config.ts
├── web/                     # Frontend build output (generated)
├── Dockerfile               # Multi-stage Docker build
└── README.md
```

## Local Development

### Prerequisites

- Go 1.21 or later
- Node.js 20+ and npm
- (Optional) Docker for containerized deployment

### Running Locally

#### Quick Setup

For the easiest setup, run the setup script:
```bash
./setup-dev.sh
```

This will:
- Install frontend dependencies (if needed)
- Build the frontend
- Copy web files to the correct location for Go embed

#### Backend

1. Install Go dependencies:
```bash
go mod download
```

2. **Important**: Before running the server, you need to build the frontend and set up the web directory:
```bash
# Build frontend
cd frontend
npm install
npm run build
cd ..

# Copy web directory for Go embed
mkdir -p cmd/server/web
cp -r web/* cmd/server/web/
```

3. Run the server:
```bash
go run cmd/server/main.go
```

The server will start on `http://localhost:8080`

#### Frontend Development (with Hot Reload)

1. Navigate to the frontend directory:
```bash
cd frontend
```

2. Install dependencies:
```bash
npm install
```

3. **Make sure the backend server is running** (see Backend section above)

4. Start the development server:
```bash
npm run dev
```

The frontend dev server will run on `http://localhost:5173` (Vite default).

**Note**: The Vite dev server is configured to proxy API requests (`/api`) and WebSocket connections (`/ws`) to the backend server on `http://localhost:8080`. Make sure both servers are running for full functionality.

### Building for Production

#### Frontend

```bash
cd frontend
npm run build
```

This will build the frontend and output files to `../web/` directory.

#### Backend

```bash
go build -o server ./cmd/server
```

The binary will be created in the current directory.

#### Running Production Build

1. Build the frontend first (see above)
2. Build the backend (see above)
3. Run the server:
```bash
./server
```

The server will serve both the API and the frontend static files on port 8080.

## Docker Deployment

### Building the Docker Image

```bash
docker build -t whatsdown .
```

### Running the Container

```bash
docker run -p 8080:8080 whatsdown
```

The application will be available at `http://localhost:8080`

### Docker Image Details

The Dockerfile uses a multi-stage build:

1. **Frontend builder**: Builds the React application using Node.js
2. **Backend builder**: Compiles the Go binary and embeds the frontend build
3. **Runtime**: Minimal Alpine-based image containing only the Go binary and static assets

The final image is optimized for size and contains:
- Statically linked Go binary
- Built frontend assets embedded in the binary
- No runtime dependencies (except ca-certificates for HTTPS if needed)

## API Endpoints

### Authentication

- `POST /api/login` - Login with username
  - Body: `{ "username": "string" }`
  - Returns: `{ "username": "string", "online": boolean }`
  - Error 409: User already logged in from another device

- `POST /api/logout` - Logout current session

- `GET /api/me` - Get current user info
  - Returns: `{ "username": "string", "online": boolean }`

### Users

- `GET /api/users?search=<query>` - Search users by username
  - Returns: Array of `{ "username": "string", "online": boolean }`

### Conversations

- `GET /api/conversations` - Get all conversations for current user
  - Returns: Array of conversation objects

- `GET /api/conversations/{peerUsername}` - Get messages for a conversation
  - Returns: Array of message objects

### WebSocket

- `GET /ws` - WebSocket endpoint for real-time communication
  - Requires authentication via session cookie
  - Message format: `{ "type": "message"|"typing"|"status"|"ack", "payload": {...} }`

## WebSocket Message Types

### Client → Server

**Send Message**:
```json
{
  "type": "message",
  "payload": {
    "to": "username",
    "content": "message text",
    "tempId": "optional-temp-id"
  }
}
```

**Typing Indicator**:
```json
{
  "type": "typing",
  "payload": {
    "to": "username",
    "isTyping": true
  }
}
```

### Server → Client

**Message**:
```json
{
  "type": "message",
  "payload": {
    "id": "message-id",
    "from": "sender-username",
    "to": "recipient-username",
    "content": "message text",
    "timestamp": "2024-01-01T12:00:00Z",
    "status": "sent" | "delivered"
  }
}
```

**Typing Indicator**:
```json
{
  "type": "typing",
  "payload": {
    "from": "username",
    "isTyping": true
  }
}
```

**Status Update**:
```json
{
  "type": "status",
  "payload": {
    "username": "username",
    "online": true
  }
}
```

**Acknowledgment**:
```json
{
  "type": "ack",
  "payload": {
    "messageId": "message-id",
    "status": "delivered"
  }
}
```

## Architecture Notes

### Backend

- **Hub Pattern**: Central hub manages all WebSocket connections and routes messages
- **Goroutines**: Each client has read/write pump goroutines for efficient I/O
- **Mutex Protection**: All shared state (users, conversations, clients) is protected with RWMutex
- **In-Memory Storage**: All data stored in memory (no persistence)

### Frontend

- **React Context**: Global state management via Context API
- **WebSocket Client**: Auto-reconnecting WebSocket client with exponential backoff
- **Optimistic Updates**: Messages appear immediately, status updates via acknowledgments
- **Responsive Design**: Mobile-friendly layout with TailwindCSS

## Security Considerations

- Session cookies are HttpOnly and use SameSite protection
- Username validation (alphanumeric + underscores only)
- Single session enforcement per username
- WebSocket origin checking (currently allows all for demo)

**Note**: For production deployment, consider:
- HTTPS/WSS for secure connections
- Rate limiting on API endpoints
- Input sanitization and validation
- Session expiration and cleanup
- CORS configuration

## Limitations

- **No persistence**: All data is lost on server restart
- **Single server**: No horizontal scaling support
- **No message history**: Messages only available while server is running
- **No file attachments**: Text messages only
- **No group chats**: 1:1 conversations only

## License

See LICENSE file for details.
