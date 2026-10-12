import test from 'node:test';
import assert from 'node:assert/strict';
import { tagTransition } from '../src/tag-policy.ts';
const base={enabled:true,url:'http://synthetic',tagSync:{resolution:false,codec:false},hasKey:true,interval:60};
test('tag policy confirmation covers activation, retirement, mode and endpoint changes',()=>{
 const active={...base,tagSync:{resolution:true,codec:true},tagSyncMode:'add'};
 let c=tagTransition('radarr',base,active);assert.deepEqual(c.enabled,['resolution','codec']);assert.equal(c.needsMode,true);
 c=tagTransition('radarr',active,{...active,tagSync:{resolution:false,codec:true}});assert.deepEqual(c.disabled,['resolution']);assert.equal(c.needsMode,false);
 assert.equal(tagTransition('radarr',active,{...active,tagSyncMode:'replace'}).needsMode,true);
 assert.equal(tagTransition('radarr',active,{...active,url:'http://other'}).needsMode,true);
 assert.equal(tagTransition('radarr',active,{...active,interval:30}).needsMode,false);
 assert.equal(tagTransition('radarr',base,{...base,url:'http://other'}).needsMode,false);
});
