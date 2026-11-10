import test from 'node:test'
import assert from 'node:assert/strict'
import {createRequire} from 'node:module'
import {fileURLToPath} from 'node:url'
import {readFile} from 'node:fs/promises'
const {buildSync}=createRequire(import.meta.url)('esbuild')
const result=buildSync({stdin:{contents:`export {setRegisteredCards} from '../src/home/cards/catalog';export {definition,resolvePresentation,formCapability} from '../src/home/layout/widgetPresentation';export {configValue} from '../src/home/configContract'`,resolveDir:fileURLToPath(new URL('.',import.meta.url))},bundle:true,write:false,format:'esm',platform:'node'})
const api=await import('data:text/javascript;base64,'+Buffer.from(result.outputFiles[0].text).toString('base64'))
const fixture=JSON.parse(await readFile(new URL('../../docs/contracts/fastnet-card.json',import.meta.url)))
test('registered forms constrain rendering, while unsupported saved preferences remain valid layout data',()=>{
 const m={...fixture.manifest,forms:{standard:fixture.manifest.forms.standard},stacked:[]};api.setRegisteredCards([m])
 assert.deepEqual(api.definition(m.id).forms,['standard']);assert.equal(api.formCapability(m.id,'summary').stacked,false)
 const saved=structuredClone(fixture.layoutV4);saved.widgetPreferences[m.id].mode='summary';assert.equal(api.configValue('layout',saved).schemaVersion,4)
 assert.equal(api.resolvePresentation(m.id,saved.widgetPreferences[m.id],{width:600,height:400,role:'ordinary',favoriteCount:0}).issue,'form')
 api.setRegisteredCards([{...m,forms:{summary:fixture.manifest.forms.summary},stacked:['summary']}])
 assert.equal(api.resolvePresentation(m.id,{mode:'auto',iconRows:0,ruleVersion:2},{width:600,height:400,role:'ordinary',favoriteCount:0}).id,'summary')
 api.setRegisteredCards([]);assert.equal(api.configValue('layout',saved).widgetPreferences[m.id].mode,'summary')
})
