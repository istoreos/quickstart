#!/usr/bin/env node
import { readFile } from 'node:fs/promises'
import { pathToFileURL } from 'node:url'

const number = value => Number.parseFloat(value || '0')
const unique = (rows, key) => [...new Set(rows.map(row => row[key]))]
const percentileMedian = values => {
  const sorted = [...values].sort((a, b) => a - b)
  const middle = Math.floor(sorted.length / 2)
  return sorted.length % 2 ? sorted[middle] : (sorted[middle - 1] + sorted[middle]) / 2
}

const rssTrend = (rows, key, durationSeconds) => {
  const values = rows.map(row => number(row[key]))
  const quarter = Math.max(1, Math.floor(values.length / 4))
  const firstMedianKiB = percentileMedian(values.slice(0, quarter))
  const lastMedianKiB = percentileMedian(values.slice(-quarter))
  const growthKiB = lastMedianKiB - firstMedianKiB
  const nonDecreasingRatio = values.slice(1).filter((value, index) => value >= values[index]).length / Math.max(1, values.length - 1)
  const growthKiBPerHour = durationSeconds > 0 ? growthKiB / (durationSeconds / 3600) : 0
  const sustainedGrowth = growthKiB > 8192 && growthKiBPerHour > 256 && nonDecreasingRatio >= 0.9
  return { minKiB: Math.min(...values), maxKiB: Math.max(...values), firstMedianKiB, lastMedianKiB, growthKiB, growthKiBPerHour, nonDecreasingRatio, sustainedGrowth }
}

export const analyzeSoakRows = (rows, { minimumDurationSeconds = 86400 } = {}) => {
  if (rows.length < 2) return { passed: false, sampleCount: rows.length, failures: ['at least two samples are required'] }
  const durationSeconds = number(rows.at(-1).epoch) - number(rows[0].epoch)
  const failures = []
  const checkUnique = (key, label) => { const values = unique(rows, key); if (values.length !== 1 || ['missing', 'unreachable', ''].includes(values[0])) failures.push(`${label} changed or was unavailable: ${values.join(', ')}`) }
  checkUnique('a_pid', 'A process identity')
  checkUnique('a_start_ticks', 'A process start ticks')
  checkUnique('b_pid', 'B process identity')
  checkUnique('b_start_ticks', 'B process start ticks')
  if (durationSeconds < minimumDurationSeconds) failures.push(`duration ${durationSeconds}s is below ${minimumDurationSeconds}s`)

  const apiErrors = rows.filter(row => row.a_api_http !== '200' || row.b_api_http !== '200').length
  const doubleMasterSamples = rows.filter(row => row.a_vip === 'present' && row.b_vip === 'present').length
  const noMasterSamples = rows.filter(row => row.a_vip !== 'present' && row.b_vip !== 'present').length
  const cRouteErrors = rows.filter(row => row.c_gateway !== '192.168.30.244' || row.c_ping !== 'ok').length
  const dRouteErrors = rows.filter(row => row.d_gateway !== '192.168.30.1' || row.d_ping !== 'ok').length
  if (apiErrors) failures.push(`API failed in ${apiErrors} samples`)
  if (doubleMasterSamples) failures.push(`double master observed in ${doubleMasterSamples} samples`)
  if (noMasterSamples) failures.push(`no VIP master observed in ${noMasterSamples} samples`)
  if (cRouteErrors) failures.push(`C gateway or Internet check failed in ${cRouteErrors} samples`)
  if (dRouteErrors) failures.push(`D gateway or Internet check failed in ${dRouteErrors} samples`)

  const aRss = rssTrend(rows, 'a_rss_kb', durationSeconds)
  const bRss = rssTrend(rows, 'b_rss_kb', durationSeconds)
  if (aRss.sustainedGrowth) failures.push(`A RSS has sustained growth: ${aRss.growthKiB} KiB`)
  if (bRss.sustainedGrowth) failures.push(`B RSS has sustained growth: ${bRss.growthKiB} KiB`)
  return {
    passed: failures.length === 0, failures, sampleCount: rows.length, durationSeconds,
    apiErrors, doubleMasterSamples, noMasterSamples, cRouteErrors, dRouteErrors,
    a: { pid: rows[0].a_pid, startTicks: rows[0].a_start_ticks, maxThreads: Math.max(...rows.map(row => number(row.a_threads))), maxFD: Math.max(...rows.map(row => number(row.a_fd))), rss: aRss },
    b: { pid: rows[0].b_pid, startTicks: rows[0].b_start_ticks, maxThreads: Math.max(...rows.map(row => number(row.b_threads))), maxFD: Math.max(...rows.map(row => number(row.b_fd))), rss: bRss },
  }
}

export const parseTSV = source => {
  const [header, ...lines] = source.trim().split(/\r?\n/)
  const keys = header.split('\t')
  return lines.filter(Boolean).map(line => Object.fromEntries(line.split('\t').map((value, index) => [keys[index], value])))
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  const file = process.argv[2]
  if (!file) throw new Error('usage: analyze-lan-device-soak.mjs <samples.tsv> [minimum-duration-seconds]')
  const result = analyzeSoakRows(parseTSV(await readFile(file, 'utf8')), { minimumDurationSeconds: Number(process.argv[3] || 86400) })
  console.log(JSON.stringify(result, null, 2))
  if (!result.passed) process.exitCode = 1
}
