import {shallowRef} from 'vue'
import {cardRegistry} from './registryClient'
import type {CardManifest} from './contract'
const cards=shallowRef<CardManifest[]>([]),error=shallowRef(''),state=shallowRef<'unknown'|'ready'|'unavailable'>('unknown')
async function discover(force=false){try{cards.value=await cardRegistry.read(force);state.value='ready';error.value=''}catch{state.value='unavailable';error.value='无法确认插件注册信息'}return cards.value}
export function useCardRegistry(){return {cards,error,state,discover}}
