#!/usr/bin/env node
import { readFile } from 'node:fs/promises'
import { pathToFileURL } from 'node:url'

const isObject = value => value !== null && typeof value === 'object' && !Array.isArray(value)
const devicesAt = node => Array.isArray(node?.devices) ? node.devices : []
const targetsAt = node => Array.isArray(node?.targets) ? node.targets : []

export const analyzeBusinessSnapshot = snapshot => {
  const failures = []
  const require = (condition, message) => { if (!condition) failures.push(message) }
  const hasCurrentIPv4 = (node, address) => devicesAt(node).some(device =>
    device?.online === true && device?.addresses?.current?.some(item => item?.family === 4 && item?.address === address))
  const hasTarget = (node, kind, gateway) => targetsAt(node).some(target =>
    target?.kind === kind && target?.gateway === gateway && target?.supported === true)
  const checkUnique = (values, label) => require(new Set(values).size === values.length, `${label} contain duplicate device IDs`)
  const checkRules = (node, label) => {
    const rules = Array.isArray(node?.rules) ? node.rules : []
    require(Array.isArray(node?.rules), `${label} rules are unavailable`)
    require(new Set(rules.map(rule => rule?.id)).size === rules.length, `${label} rules contain duplicate IDs`)
    require(rules.every(rule => rule?.status !== 'orphaned' || rule?.orphaned === true), `${label} orphaned rule state is inconsistent`)
  }
  const checkGroups = (node, label) => {
    require(Array.isArray(node?.groups), `${label} groups are unavailable`)
    require(isObject(node?.devicePolicies), `${label} device policy map is unavailable`)
    require(Array.isArray(node?.effective), `${label} effective policy list is unavailable`)
    require(typeof node?.timezone === 'string' && node.timezone.length > 0, `${label} timezone is unavailable`)
    require(typeof node?.version === 'string' && node.version.length > 0, `${label} group version is unavailable`)
  }
  const checkTraffic = (node, label) => {
    require(!node?.error, `${label} traffic error: ${node?.error?.code || 'unknown'}`)
    require(node?.range === 'today', `${label} traffic range is not today`)
    require(Array.isArray(node?.buckets) || node?.buckets === null, `${label} traffic buckets are invalid`)
    require(node?.local?.state === 'available', `${label} local traffic adapter is unavailable`)
    require(Number.isFinite(node?.uploadBytes) && node.uploadBytes >= 0, `${label} upload counter is invalid`)
    require(Number.isFinite(node?.downloadBytes) && node.downloadBytes >= 0, `${label} download counter is invalid`)
    require(Number.isFinite(node?.storageBytes) && Number.isFinite(node?.storageBudget) && node.storageBytes <= node.storageBudget, `${label} traffic storage exceeds its budget`)
  }

  require(snapshot?.a?.routerContext?.topologyPosition === 'lan_gateway_candidate', 'A topology role is not LAN gateway')
  require(snapshot?.a?.routerContext?.dhcpAuthority === 'local', 'A DHCP authority is not local')
  require(snapshot?.a?.routerContext?.routeEditability?.editable === true, 'A route editing is not available')
  require(snapshot?.b?.routerContext?.topologyPosition === 'downstream_router', 'B topology role is not downstream router')
  require(snapshot?.b?.routerContext?.dhcpAuthority !== 'local', 'B must not claim local DHCP authority')
  require(snapshot?.b?.routerContext?.routeEditability?.editable === false, 'B route editing must fail closed')

  checkUnique(devicesAt(snapshot?.a).map(device => device?.deviceId), 'A inventory')
  checkUnique(devicesAt(snapshot?.b).map(device => device?.deviceId), 'B inventory')
  require(hasCurrentIPv4(snapshot?.a, '192.168.30.93'), 'A inventory does not observe C online at 192.168.30.93')
  require(hasCurrentIPv4(snapshot?.a, '192.168.30.7'), 'A inventory does not observe D online at 192.168.30.7')
  require(hasCurrentIPv4(snapshot?.b, '192.168.30.93'), 'B inventory does not observe C online at 192.168.30.93')
  require(hasCurrentIPv4(snapshot?.b, '192.168.30.7'), 'B inventory does not observe D online at 192.168.30.7')

  require(hasTarget(snapshot?.a, 'self', '192.168.30.1'), 'A self route target is missing')
  require(hasTarget(snapshot?.a, 'bypass', '192.168.30.244'), 'A bypass route target is missing')
  require(hasTarget(snapshot?.a, 'floating', '192.168.30.3'), 'A floating route target is missing')
  require(hasTarget(snapshot?.b, 'self', '192.168.30.244'), 'B self route target is missing')
  require(hasTarget(snapshot?.b, 'upstream', '192.168.30.1'), 'B upstream route target is missing')
  require(hasTarget(snapshot?.b, 'floating', '192.168.30.3'), 'B floating route target is missing')

  require(snapshot?.a?.dhcp?.settings?.enabled === true && snapshot?.a?.dhcp?.editable === true, 'A DHCP settings must be enabled and editable')
  require(snapshot?.b?.dhcp?.editable === false, 'B DHCP settings must be read-only')
  require(snapshot?.a?.floating?.config?.enabled === true && snapshot?.a?.floating?.status?.serviceRunning === true, 'A floating gateway service is unavailable')
  require(snapshot?.b?.floating?.config?.enabled === true && snapshot?.b?.floating?.status?.serviceRunning === true, 'B floating gateway service is unavailable')
  require(snapshot?.b?.floating?.status?.holder === 'local' && snapshot?.b?.floating?.status?.state === 'healthy', 'B floating holder is not healthy/local')
  require(snapshot?.a?.floating?.status?.holder === 'peer', 'A floating holder must report the peer while B owns the VIP')

  checkRules(snapshot?.a?.rules, 'A')
  checkRules(snapshot?.b?.rules, 'B')
  checkGroups(snapshot?.a?.groups, 'A')
  checkGroups(snapshot?.b?.groups, 'B')
  checkTraffic(snapshot?.a?.traffic, 'A')
  checkTraffic(snapshot?.b?.traffic, 'B')

  const aCEffect = snapshot?.a?.cPolicy?.policy?.path?.effect
  const aDEffect = snapshot?.a?.dPolicy?.policy?.path?.effect
  const bCEffect = snapshot?.b?.cPolicy?.policy?.path?.effect
  require(isObject(aCEffect?.desired) && isObject(aCEffect?.applied) && isObject(aCEffect?.observed), 'A C policy does not separate desired, applied and observed state')
  require(aCEffect?.observed?.state !== 'active', 'A C policy must not claim active without terminal route evidence')
  require(aDEffect?.desired?.targetId === 'default' && aDEffect?.needsAttention === false, 'A D default route policy is inconsistent')
  require(bCEffect?.needsAttention === true && bCEffect?.attentionReason === 'dhcp_authority_unavailable', 'B C policy must explain unavailable DHCP authority')

  require(snapshot?.network?.aVip !== snapshot?.network?.bVip && ['present', 'absent'].includes(snapshot?.network?.aVip) && ['present', 'absent'].includes(snapshot?.network?.bVip), 'network must have exactly one VIP owner')
  require(snapshot?.network?.cGateway === '192.168.30.244', 'C default gateway changed from B')
  require(snapshot?.network?.dGateway === '192.168.30.1', 'D default gateway changed from A')
  require(snapshot?.network?.cPing === 'ok' && snapshot?.network?.dPing === 'ok', 'C or D Internet reachability failed')

  return { passed: failures.length === 0, failures, checks: 43 }
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  const file = process.argv[2]
  if (!file) throw new Error('usage: analyze-lan-device-business-smoke.mjs <snapshot.json>')
  const result = analyzeBusinessSnapshot(JSON.parse(await readFile(file, 'utf8')))
  console.log(JSON.stringify(result, null, 2))
  if (!result.passed) process.exitCode = 1
}
