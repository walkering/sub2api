export interface AccountTestStreamEvent {
  type: string
  text?: string
  model?: string
  success?: boolean
  error?: string
  image_url?: string
  mime_type?: string
}

export interface StreamAccountTestParams {
  accountId: number
  authToken?: string | null
  modelId?: string
  prompt?: string
  mode?: string
  signal?: AbortSignal
  onEvent?: (event: AccountTestStreamEvent) => void
}

export interface StreamAccountTestResult {
  success: boolean
  error?: string
}

export async function streamAccountTest({
  accountId,
  authToken,
  modelId,
  prompt,
  mode,
  signal,
  onEvent
}: StreamAccountTestParams): Promise<StreamAccountTestResult> {
  const url = `/api/v1/admin/accounts/${accountId}/test`
  const payload: Record<string, unknown> = {}

  if (modelId) payload.model_id = modelId
  if (prompt) payload.prompt = prompt
  if (mode) payload.mode = mode

  const response = await fetch(url, {
    method: 'POST',
    headers: {
      Authorization: `Bearer ${authToken || ''}`,
      'Content-Type': 'application/json'
    },
    body: JSON.stringify(payload),
    signal
  })

  if (!response.ok) {
    throw new Error(`HTTP error! status: ${response.status}`)
  }

  const reader = response.body?.getReader()
  if (!reader) {
    throw new Error('No response body')
  }

  const decoder = new TextDecoder()
  let buffer = ''
  let terminalResult: StreamAccountTestResult | null = null

  while (true) {
    const { done, value } = await reader.read()
    if (done) break

    buffer += decoder.decode(value, { stream: true })
    const lines = buffer.split('\n')
    buffer = lines.pop() || ''

    for (const rawLine of lines) {
      const line = rawLine.trim()
      if (!line.startsWith('data:')) continue

      const jsonStr = line.slice(5).trim()
      if (!jsonStr) continue

      try {
        const event = JSON.parse(jsonStr) as AccountTestStreamEvent
        onEvent?.(event)

        if (event.type === 'test_complete') {
          terminalResult = event.success
            ? { success: true }
            : { success: false, error: event.error || 'Test failed' }
        } else if (event.type === 'error') {
          terminalResult = { success: false, error: event.error || 'Unknown error' }
        }
      } catch (error) {
        console.error('Failed to parse SSE event:', error)
      }
    }
  }

  if (terminalResult) {
    return terminalResult
  }

  return { success: false, error: 'Stream ended before test completion' }
}
