import test from 'node:test';
import assert from 'node:assert/strict';
import { confirmDeletion,currentConfirmation,answerConfirmation,subscribeConfirmation } from '../src/confirmation.ts';
test('deletion waits for explicit confirmation, supports cancellation and rejects overlapping prompts',async()=>{
 let updates=0;const unsubscribe=subscribeConfirmation(()=>updates++);
 const cancelled=confirmDeletion('Delete first?');assert.equal(currentConfirmation().message,'Delete first?');
 assert.equal(await confirmDeletion('Delete second?'),false);
 answerConfirmation(false);assert.equal(await cancelled,false);assert.equal(currentConfirmation(),null);
 const accepted=confirmDeletion('Delete third?');answerConfirmation(true);assert.equal(await accepted,true);
 answerConfirmation(true);assert.equal(updates,4);unsubscribe();
});
