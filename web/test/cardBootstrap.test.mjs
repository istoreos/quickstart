import test from 'node:test'
import assert from 'node:assert/strict'
import {readFile,readdir} from 'node:fs/promises'
import path from 'node:path'
const root=process.env.CARD_HOST_BUILD
// Exercise the actual emitted graph, including the query part of module identity.
test('FN09 emitted async modules never import the LuCI bootstrap under a second URL',{skip:!root},async()=>{
 const assets=path.join(root,'luci-static/quickstart')
 for(const name of await readdir(assets))if(name.endsWith('.js')&&name!=='index.js'){
  const source=await readFile(path.join(assets,name),'utf8')
  assert.doesNotMatch(source,/(?:from|import\()\s*["']\.\/index\.js(?:[?"'])/,name+' imports the bootstrap')
 }
})
