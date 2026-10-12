import test from "node:test";
import assert from "node:assert/strict";
import { matchesRuleSearch } from "../src/rule-search.ts";
test("recognition search combines terms across every displayed column", () => {
 const columns = ["105","Enabled","Audio · DTS-HD","Audio codec","Regular expression","dts[ ._-]?hd","DTS-HD","Direct recognition"];
 for (const query of ["105","ENABLED","dts","regular expression","audio 105","direct DTS-HD"]) assert.equal(matchesRuleSearch(query,columns),true,query);
 assert.equal(matchesRuleSearch("DTS 2160p",columns),false);
 assert.equal(matchesRuleSearch("  ",columns),true);
 assert.equal(matchesRuleSearch("Favorites",["Custom","Favorites"]),true);
});
