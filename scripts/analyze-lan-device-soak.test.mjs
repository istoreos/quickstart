import assert from 'node:assert/strict'
import test from 'node:test'
import { analyzeSoakRows } from './analyze-lan-device-soak.mjs'

const row = (epoch, overrides = {}) => ({
  utc: new Date(epoch * 1000).toISOString(), epoch: String(epoch),
  a_pid: '10', a_start_ticks: '100', a_rss_kb: '20000', a_threads: '6', a_fd: '8', a_api_http: '200', a_api_seconds: '0.01',
  b_pid: '20', b_start_ticks: '200', b_rss_kb: '15000', b_threads: '5', b_fd: '8', b_api_http: '200', b_api_seconds: '0.01',
  a_vip: 'absent', b_vip: 'present', c_gateway: '192.168.30.244', c_dns: '', c_ping: 'ok', d_gateway: '192.168.30.1', d_dns: '192.168.30.1', d_ping: 'ok',
  ...overrides,
})

test('healthy 24 hour topology soak passes all release gates', () => {
  const rows = Array.from({ length: 289 }, (_, index) => row(index * 300, {
    a_rss_kb: String(20000 + (index % 4) * 128),
    b_rss_kb: String(15000 + (index % 3) * 128),
  }))
  const result = analyzeSoakRows(rows, { minimumDurationSeconds: 86400 })
  assert.equal(result.passed, true)
  assert.equal(result.sampleCount, 289)
  assert.equal(result.doubleMasterSamples, 0)
})

test('restart API error route drift and sustained RSS growth fail closed', () => {
  const rows = Array.from({ length: 25 }, (_, index) => row(index * 3600, {
    a_pid: index === 20 ? '11' : '10',
    a_api_http: index === 10 ? '500' : '200',
    c_gateway: index === 12 ? '192.168.30.1' : '192.168.30.244',
    a_vip: index === 14 ? 'present' : 'absent',
    a_rss_kb: String(20000 + index * 1024),
  }))
  const result = analyzeSoakRows(rows, { minimumDurationSeconds: 86400 })
  assert.equal(result.passed, false)
  assert.ok(result.failures.some(value => value.includes('A process identity')))
  assert.ok(result.failures.some(value => value.includes('API')))
  assert.ok(result.failures.some(value => value.includes('C gateway')))
  assert.ok(result.failures.some(value => value.includes('double master')))
  assert.ok(result.failures.some(value => value.includes('RSS')))
})
