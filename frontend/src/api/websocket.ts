import { WSMessage, OutboundMessage, TypingEvent, StatusEvent, AckEvent } from './types';

type MessageHandler = (msg: OutboundMessage) => void;
type TypingHandler = (event: TypingEvent) => void;
type StatusHandler = (event: StatusEvent) => void;
type AckHandler = (event: AckEvent) => void;

export class WebSocketClient {
  private ws: WebSocket | null = null;
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 10;
  private reconnectDelay = 1000;
  private messageHandlers: MessageHandler[] = [];
  private typingHandlers: TypingHandler[] = [];
  private statusHandlers: StatusHandler[] = [];
  private ackHandlers: AckHandler[] = [];
  private onConnectCallback?: () => void;
  private onDisconnectCallback?: () => void;
  private onReconnectingCallback?: () => void;

  connect(): Promise<void> {
    return new Promise((resolve, reject) => {
      const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
      const wsUrl = `${protocol}//${window.location.host}/ws`;
      
      this.ws = new WebSocket(wsUrl);

      this.ws.onopen = () => {
        console.log('WebSocket connected');
        this.reconnectAttempts = 0;
        if (this.onConnectCallback) {
          this.onConnectCallback();
        }
        resolve();
      };

      this.ws.onmessage = (event) => {
        try {
          const wsMsg: WSMessage = JSON.parse(event.data);
          console.log('WebSocket message received:', wsMsg.type, wsMsg.payload);
          this.handleMessage(wsMsg);
        } catch (error) {
          console.error('Error parsing WebSocket message:', error);
        }
      };

      this.ws.onerror = (error) => {
        console.error('WebSocket error:', error);
        reject(error);
      };

      this.ws.onclose = () => {
        console.log('WebSocket disconnected');
        this.ws = null;
        if (this.onDisconnectCallback) {
          this.onDisconnectCallback();
        }
        this.attemptReconnect();
      };
    });
  }

  private attemptReconnect() {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      console.error('Max reconnection attempts reached');
      return;
    }

    this.reconnectAttempts++;
    const delay = Math.min(this.reconnectDelay * Math.pow(2, this.reconnectAttempts - 1), 30000);
    
    if (this.onReconnectingCallback) {
      this.onReconnectingCallback();
    }

    setTimeout(() => {
      console.log(`Attempting to reconnect (${this.reconnectAttempts}/${this.maxReconnectAttempts})...`);
      this.connect().catch(() => {
        // Reconnection will be attempted again in onclose
      });
    }, delay);
  }

  private handleMessage(wsMsg: WSMessage) {
    switch (wsMsg.type) {
      case 'message':
        const msg = wsMsg.payload as OutboundMessage;
        this.messageHandlers.forEach(handler => handler(msg));
        break;
      case 'typing':
        const typingEvent = wsMsg.payload as TypingEvent;
        this.typingHandlers.forEach(handler => handler(typingEvent));
        break;
      case 'status':
        const statusEvent = wsMsg.payload as StatusEvent;
        this.statusHandlers.forEach(handler => handler(statusEvent));
        break;
      case 'ack':
        const ackEvent = wsMsg.payload as AckEvent;
        this.ackHandlers.forEach(handler => handler(ackEvent));
        break;
    }
  }

  sendMessage(to: string, content: string, tempId?: string) {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
      console.error('WebSocket is not connected');
      return;
    }

    const wsMsg: WSMessage = {
      type: 'message',
      payload: {
        to,
        content,
        tempId,
      },
    };

    this.ws.send(JSON.stringify(wsMsg));
  }

  sendTyping(to: string, isTyping: boolean) {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
      return;
    }

    const wsMsg: WSMessage = {
      type: 'typing',
      payload: {
        to,
        isTyping,
      },
    };

    this.ws.send(JSON.stringify(wsMsg));
  }

  onMessage(handler: MessageHandler) {
    this.messageHandlers.push(handler);
  }

  onTyping(handler: TypingHandler) {
    this.typingHandlers.push(handler);
  }

  onStatus(handler: StatusHandler) {
    this.statusHandlers.push(handler);
  }

  onAck(handler: AckHandler) {
    this.ackHandlers.push(handler);
  }

  onConnect(callback: () => void) {
    this.onConnectCallback = callback;
  }

  onDisconnect(callback: () => void) {
    this.onDisconnectCallback = callback;
  }

  onReconnecting(callback: () => void) {
    this.onReconnectingCallback = callback;
  }

  disconnect() {
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
    this.reconnectAttempts = this.maxReconnectAttempts; // Prevent reconnection
  }

  isConnected(): boolean {
    return this.ws !== null && this.ws.readyState === WebSocket.OPEN;
  }
}

