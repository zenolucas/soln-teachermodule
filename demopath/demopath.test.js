// Run with `node --test demopath/demopath.test.js` - checks public/js/demopath.js against the same
// vectors as demopath_test.go, so the browser and the exporter agree on every path.
const test = require("node:test");
const assert = require("node:assert");
const { solnStaticPath } = require("../public/js/demopath.js");
const vectors = require("./testdata/vectors.json");

test("solnStaticPath matches the Go vectors", () => {
	for (const v of vectors) {
		// htmx hands over a parameters object; build it the way htmx would, repeating
		// keys as arrays, so both input shapes (path query + params) get exercised.
		const params = {};
		new URLSearchParams(v.query).forEach((value, key) => {
			params[key] = key in params ? [].concat(params[key], value) : value;
		});
		assert.strictEqual(solnStaticPath(v.path, params), v.want, `${v.path}?${v.query}`);
		assert.strictEqual(solnStaticPath(v.query ? `${v.path}?${v.query}` : v.path, {}), v.want, `${v.path}?${v.query} (in path)`);
	}
});
