#!/usr/bin/env node

import fs from 'node:fs'
import path from 'node:path'
import process from 'node:process'
import { fileURLToPath } from 'node:url'

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const sourcePath = path.join(root, 'docs/lan-device-management-requirements.json')
const outputPath = path.join(root, 'docs/lan-device-management-coverage-matrix.md')
const layerNames = ['product', 'prototype', 'backend', 'frontend', 'device']
const statuses = new Set(['not_started', 'partial', 'blocked', 'verified', 'not_applicable'])
const requiredScenarios = ['normal', 'missing', 'error', 'recovery', 'retry']

function localRefExists(ref) {
  if (ref.startsWith('http://') || ref.startsWith('https://')) return true
  const file = ref.split('#', 1)[0]
  return file.length > 0 && fs.existsSync(path.join(root, file))
}

export function validate(matrix) {
  const errors = []
  if (matrix.schemaVersion !== 1) errors.push('schemaVersion must be 1')
  if (!Array.isArray(matrix.requirements)) return ['requirements must be an array']
  const ids = new Set()
  for (const [index, item] of matrix.requirements.entries()) {
    const at = `requirements[${index}]`
    if (!/^[A-Z0-9]+(?:-[A-Z0-9]+)+$/.test(item.id ?? '')) errors.push(`${at}: invalid Requirement ID`)
    if (ids.has(item.id)) errors.push(`${at}: duplicate Requirement ID ${item.id}`)
    ids.add(item.id)
    if (!['P0', 'P1', 'P2'].includes(item.priority)) errors.push(`${item.id}: invalid priority`)
    for (const field of ['requirement', 'acceptance', 'source', 'milestone']) {
      if (typeof item[field] !== 'string' || item[field].trim() === '') errors.push(`${item.id}: missing ${field}`)
    }
    for (const layer of layerNames) {
      const value = item[layer]
      if (!value || !statuses.has(value.status)) {
        errors.push(`${item.id}: invalid or missing ${layer} status`)
        continue
      }
      if (!Array.isArray(value.refs) || value.refs.length === 0) errors.push(`${item.id}: ${layer} needs an evidence or milestone reference`)
      for (const ref of value.refs ?? []) {
        if (ref.startsWith('planned:')) errors.push(`${item.id}: stale planned reference is not allowed: ${ref}`)
        else if (!localRefExists(ref)) errors.push(`${item.id}: ${value.status} ${layer} evidence does not exist: ${ref}`)
      }
    }
    if (!statuses.has(item.overall)) errors.push(`${item.id}: invalid overall status`)
    if (item.overall === 'verified' && layerNames.some((layer) => !['verified', 'not_applicable'].includes(item[layer]?.status))) {
      errors.push(`${item.id}: overall verified requires all five layers verified or not_applicable`)
    }
    if (item.overall !== 'verified') {
      if (typeof item.blocker !== 'string' || item.blocker.trim() === '') errors.push(`${item.id}: non-verified requirement needs blocker`)
      if (!/^M(?:5[4-9]|6[0-4])$/.test(item.closureMilestone ?? '')) errors.push(`${item.id}: non-verified requirement needs one M54-M64 closureMilestone`)
      if (/\bM(?:39|40)\b/.test(item.blocker ?? '')) errors.push(`${item.id}: blocker still points at completed M39/M40`)
    } else if (item.closureMilestone) {
      errors.push(`${item.id}: verified requirement must not retain closureMilestone`)
    }
    if (item.source === 'p0-addition') {
      for (const scenario of requiredScenarios) if (!item.scenarios?.includes(scenario)) errors.push(`${item.id}: missing P0 scenario ${scenario}`)
    }
  }
  const legacy = matrix.requirements.filter((item) => item.source === 'legacy-48').length
  const additions = matrix.requirements.filter((item) => item.source === 'p0-addition').length
  if (legacy !== 48) errors.push(`expected 48 migrated prototype requirements, got ${legacy}`)
  if (additions !== 7) errors.push(`expected 7 P0 additions, got ${additions}`)
  return errors
}

function cell(layer) {
  return `${layer.status}<br>${layer.refs.join('<br>')}`
}

function render(matrix) {
  const lines = [
    '# 局域网设备管理五层覆盖矩阵',
    '',
    '> 本文件由 `scripts/validate-lan-device-coverage.mjs --write` 从 `lan-device-management-requirements.json` 生成，请勿手工修改。',
    `> 基线：[当前唯一产品与领域基线](./lan-device-management-product-architecture.md) · 更新：${matrix.updated}`,
    '',
    `共 ${matrix.requirements.length} 项：原型历史能力 48 项，P0 新增场景 7 项。总状态只有在五层均完成后才能为 \`verified\`。`,
    '',
    '| Requirement ID | P | 用户任务与验收结果 | 产品需求 | 原型 | 后端 | 正式前端 | 实机验收 | 总状态 / 阻塞 |',
    '| --- | --- | --- | --- | --- | --- | --- | --- | --- |'
  ]
  for (const item of matrix.requirements) {
    const closeout = item.closureMilestone ? `关闭：${item.closureMilestone}<br>` : ''
    lines.push(`| ${item.id} | ${item.priority} | ${item.requirement}<br>验收：${item.acceptance} | ${cell(item.product)} | ${cell(item.prototype)} | ${cell(item.backend)} | ${cell(item.frontend)} | ${cell(item.device)} | ${item.overall}<br>${closeout}${item.blocker ?? ''} |`)
  }
  lines.push('')
  return lines.join('\n')
}

function selfTest(matrix) {
  const mutations = [
    (copy) => { copy.requirements[1].id = copy.requirements[0].id },
    (copy) => { delete copy.requirements[0].frontend },
    (copy) => { copy.requirements[0].overall = 'done' },
    (copy) => { copy.requirements[0].product.refs = ['docs/does-not-exist.md'] },
    (copy) => { copy.requirements[0].device.refs = ['planned:M99'] },
    (copy) => {
      const open = copy.requirements.find((item) => item.overall !== 'verified') ?? copy.requirements[0]
      open.overall = 'partial'
      open.device.status = 'partial'
      open.blocker = 'validator self-test fixture'
      delete open.closureMilestone
    }
  ]
  for (const mutate of mutations) {
    const copy = structuredClone(matrix)
    mutate(copy)
    if (validate(copy).length === 0) throw new Error('validator self-test accepted an invalid matrix')
  }
}

const matrix = JSON.parse(fs.readFileSync(sourcePath, 'utf8'))
const errors = validate(matrix)
if (errors.length > 0) {
  for (const error of errors) process.stderr.write(`${error}\n`)
  process.exit(1)
}
if (process.argv.includes('--self-test')) selfTest(matrix)
if (process.argv.includes('--write')) fs.writeFileSync(outputPath, render(matrix))
process.stdout.write(`coverage matrix valid: ${matrix.requirements.length} requirements, 48 migrated, 7 P0 additions\n`)
