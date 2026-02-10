import { ServerMessage } from './types'

export type WSClient = {
  send: (data: any) => void
  close: () => void
}

export function connect(onMessage: (msg: ServerMessage) => void): WSClient {
  const ws = new WebSocket(`${location.protocol === 'https:' ? 'wss' : 'ws'}://${location.host}/ws`)
  ws.onmessage = (ev) => {
    const msg = JSON.parse(ev.data) as ServerMessage
    onMessage(msg)
  }
  return {
    send: (data) => {
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify(data))
      } else {
        ws.addEventListener(
          'open',
          () => {
            ws.send(JSON.stringify(data))
          },
          { once: true }
        )
      }
    },
    close: () => ws.close()
  }
}
