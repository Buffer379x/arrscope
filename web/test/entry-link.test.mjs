import test from 'node:test';
import assert from 'node:assert/strict';
import {entryLink} from '../src/entry-link.ts';
test('entry links use UI slugs, preserve URL subpaths and support a read-only legacy resolver',()=>{
 const connection={url:'https://example.test/radarr/',enabled:true,hasKey:true};
 assert.equal(entryLink(connection,{service:'radarr',mediaId:10,titleSlug:'603'}),'https://example.test/radarr/movie/603');
 assert.equal(entryLink({...connection,url:'https://example.test/sonarr'},{service:'sonarr',mediaId:30,titleSlug:'breaking-bad'}),'https://example.test/sonarr/series/breaking-bad');
 assert.equal(entryLink(connection,{service:'radarr',mediaId:10}),'/api/open-entry?service=radarr&mediaId=10');
 assert.equal(entryLink({...connection,enabled:false},{service:'radarr',mediaId:10}),undefined);
 for(const url of ['javascript:alert(1)','https://user:secret@example.test','https://example.test/?apikey=secret','https://example.test/#x','']) assert.equal(entryLink({...connection,url},{service:'radarr',mediaId:10,titleSlug:'603'}),undefined);
 assert.equal(entryLink(connection,{service:'radarr',mediaId:0,titleSlug:'603'}),undefined);
 assert.equal(entryLink(connection,{service:'radarr',mediaId:10,titleSlug:'../system'}),'/api/open-entry?service=radarr&mediaId=10');
});
