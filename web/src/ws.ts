import { ServerMessage } from './types'

export type WSClient = {
  send: (data: any) => void
  close: () => void
  readyState: () => number
}

export type WSStatusHandler = (status: { readyState: number; error?: string }) => void

export function connect(
  onMessage: (msg: ServerMessage) => void,
  onStatus?: WSStatusHandler
): WSClient {
  const proto = location.protocol === 'https:' ? 'wss' : 'ws'
  const url = `${proto}://${location.host}/ws`
  const ws = new WebSocket(url)

  const log = (direction: 'in' | 'out', payload: any) => {
    if (import.meta.env.DEV) {
      console.debug(`[ws:${direction}]`, payload)
    }
  }

  const notify = (error?: string) => {
    onStatus?.({ readyState: ws.readyState, error })
  }

  ws.addEventListener('open', () => notify())
  ws.addEventListener('close', () => notify())
  ws.addEventListener('error', () => notify('WebSocket error'))
  ws.onmessage = (ev) => {
    const msg = JSON.parse(ev.data) as ServerMessage
    log('in', msg)
    onMessage(msg)
  }
  return {
    send: (data) => {
      log('out', data)
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
    close: () => ws.close(),
    readyState: () => ws.readyState
  }
}
