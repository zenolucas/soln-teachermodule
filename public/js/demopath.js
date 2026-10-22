// JS port of demopath.StaticPath (demopath/demopath.go): maps a portal URL to the path
// its pre-rendered copy is served at in the static showcase. params is htmx's
// parameters object (string or array values). demopath/testdata/vectors.json pins this
// and the Go version to the same answers (demopath/demopath.test.js, run with node --test).
function solnStaticPath(path, params) {
	var query = {};
	var add = function (key, value) {
		if (value === undefined || value === null || value === "") {
			return;
		}
		(query[key] = query[key] || []).push(String(value));
	};
	var qi = path.indexOf("?");
	if (qi >= 0) {
		new URLSearchParams(path.slice(qi + 1)).forEach(function (value, key) {
			add(key, value);
		});
		path = path.slice(0, qi);
	}
	Object.keys(params || {}).forEach(function (key) {
		[].concat(params[key]).forEach(function (value) {
			add(key, value);
		});
	});

	var out = path.replace(/\/+$/, "");
	Object.keys(query)
		.sort()
		.forEach(function (key) {
			query[key].forEach(function (value) {
				out += "/~" + solnDemoEncode(key) + "=" + solnDemoEncode(value);
			});
		});
	return out + "/";
}

// Keeps [A-Za-z0-9_-] and writes every other UTF-8 byte as ".xx", like demopath.encode.
function solnDemoEncode(s) {
	var out = "";
	new TextEncoder().encode(s).forEach(function (byte) {
		var c = String.fromCharCode(byte);
		out += /[A-Za-z0-9_-]/.test(c) ? c : "." + (byte < 16 ? "0" : "") + byte.toString(16);
	});
	return out;
}

if (typeof module !== "undefined") {
	module.exports = { solnStaticPath: solnStaticPath };
}
