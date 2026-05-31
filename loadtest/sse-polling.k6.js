import http from 'k6/http'
import { check, sleep } from 'k6'

export const options = {
  scenarios: {
    sse_and_polling: {
      executor: 'ramping-vus',
      stages: [
        { duration: '1m', target: Number(__ENV.TARGET_VUS || 100) },
        { duration: '3m', target: Number(__ENV.TARGET_VUS || 100) },
        { duration: '1m', target: 0 },
      ],
    },
  },
  thresholds: {
    http_req_failed: ['rate<0.02'],
    http_req_duration: ['p(95)<750'],
  },
}

const baseUrl = __ENV.BASE_URL || 'http://localhost:8080'
const loginType = __ENV.LOGIN_TYPE || 'fte'
const email = __ENV.LOGIN_EMAIL || 'loadtest@example.com'
const opsId = __ENV.LOGIN_OPS_ID || ''
const password = __ENV.LOGIN_PASSWORD || ''

export default function () {
  const login = http.post(`${baseUrl}/api/login`, JSON.stringify({
    login_type: loginType,
    email,
    ops_id: opsId,
    password,
  }), { headers: { 'Content-Type': 'application/json' } })

  check(login, { 'login succeeded': (r) => r.status === 200 })
  if (login.status !== 200) {
    sleep(1)
    return
  }

  const jar = http.cookieJar()
  const cookies = jar.cookiesForURL(baseUrl)
  const cookieHeader = Object.entries(cookies).map(([k, values]) => `${k}=${values[0]}`).join('; ')

  const responses = http.batch([
    ['GET', `${baseUrl}/api/stats`, null, { headers: { Cookie: cookieHeader } }],
    ['GET', `${baseUrl}/api/requests?page=1&page_size=50`, null, { headers: { Cookie: cookieHeader } }],
    ['GET', `${baseUrl}/api/events`, null, { headers: { Cookie: cookieHeader }, timeout: '10s' }],
  ])

  check(responses[0], { 'stats ok': (r) => r.status === 200 })
  check(responses[1], { 'requests ok': (r) => r.status === 200 })
  check(responses[2], { 'events reachable': (r) => r.status === 200 || r.error_code === 1050 })
  sleep(5)
}
