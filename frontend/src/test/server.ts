import { setupServer } from 'msw/node'
import { http, HttpResponse } from 'msw'

// Default handlers for the MSW server
export const handlers = [
  // Health check
  http.get('/api/v1/health', () => {
    return HttpResponse.json({ status: 'ok' })
  }),
]

// Create the MSW server
export const server = setupServer(...handlers)
