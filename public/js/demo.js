// The static showcase's client side (only loaded when DEMO_MODE is on - see
// layout.Base). There's no server behind the static site, so every htmx request is
// pointed at its pre-rendered copy (export.go wrote one per URL, at solnStaticPath -
// public/js/demopath.js), and searches run over a pre-rendered list in the browser.
//
// Edits a visitor makes are kept in sessionStorage under this prefix, so they last while
// the tab is open and "Reset demo" can wipe them in one go.
var SOLN_DEMO_PREFIX = "soln.demo.";

// Routes that write. The static site has nothing to send them to, so they never leave
// the browser.
var SOLN_DEMO_WRITE = /^\/(add|update|delete)\/|^\/(createclassroom|addstudents|login|register|logout)$/;

document.body.addEventListener("click", function (evt) {
	if (!evt.target.closest("[data-demo-reset]")) {
		return;
	}
	try {
		Object.keys(sessionStorage).forEach(function (key) {
			if (key.indexOf(SOLN_DEMO_PREFIX) === 0) {
				sessionStorage.removeItem(key);
			}
		});
	} catch (e) {}
	location.reload();
});

// htmx 1.9 reads path and verb back from this event's detail after it fires, so a
// request can be redirected to a static file here - or cancelled outright.
document.body.addEventListener("htmx:configRequest", function (evt) {
	var d = evt.detail;
	var path = d.path.split("?")[0];
	if (SOLN_DEMO_WRITE.test(path)) {
		evt.preventDefault();
		solnDemoWrite(path, d);
		return;
	}

	var params = Object.assign({}, d.parameters);
	var q = String(params.q || "").trim();
	delete d.target.dataset.demoSearch;
	if (path === "/students" && q) {
		// The export can't pre-render a search, but it did write every row of each sort
		// (page=all, no filter): fetch that, and let solnDemoSearchStudents cut it down.
		d.target.dataset.demoSearch = JSON.stringify({ q: q.toLowerCase(), filter: params.filter || "all" });
		params = { classroomID: params.classroomID, sort: params.sort, filter: "all", page: "all" };
	} else if (path === "/unenrolledstudents" && q) {
		d.target.dataset.demoSearch = JSON.stringify({ q: q.toLowerCase() });
	}
	delete params.q;

	d.path = solnStaticPath(d.path, params);
	d.verb = "get";
	d.parameters = {};
});

document.body.addEventListener("htmx:beforeSwap", function (evt) {
	var target = evt.detail.target;
	var search = target && target.dataset.demoSearch;
	if (!search) {
		return;
	}
	delete target.dataset.demoSearch;
	search = JSON.parse(search);
	evt.detail.serverResponse =
		target.id === "getstudents"
			? solnDemoSearchStudents(evt.detail.serverResponse, search)
			: solnDemoSearchUnenrolled(evt.detail.serverResponse, search);
});

function solnDemoMatches(el, needle) {
	return (el.dataset.searchName || "").indexOf(needle) >= 0;
}

// Mirrors handler.applyStudentQuery on a page=all copy of StudentRows: search first,
// count the pills from what matched, then apply the filter. Results aren't paged.
function solnDemoSearchStudents(response, search) {
	var tpl = document.createElement("template");
	tpl.innerHTML = response;
	var root = tpl.content;
	var rows = Array.from(root.querySelectorAll("[data-search-name]"));
	var matched = rows.filter(function (row) {
		return solnDemoMatches(row, search.q);
	});

	var counts = { all: matched.length, attention: 0, w1: 0, w2: 0, w3: 0 };
	matched.forEach(function (row) {
		if (row.dataset.flagged === "true") {
			counts.attention++;
		}
		if (counts["w" + row.dataset.world] !== undefined) {
			counts["w" + row.dataset.world]++;
		}
	});
	Object.keys(counts).forEach(function (key) {
		var el = root.getElementById("count-" + key);
		if (el) {
			el.textContent = counts[key];
		}
	});

	var shown = matched.filter(function (row) {
		if (search.filter === "attention") {
			return row.dataset.flagged === "true";
		}
		return search.filter === "all" || "w" + row.dataset.world === search.filter;
	});
	rows.forEach(function (row) {
		if (shown.indexOf(row) < 0) {
			row.remove();
		}
	});

	var table = root.firstElementChild;
	var footer = table && table.lastElementChild;
	if (!shown.length && table) {
		var empty = document.createElement("div");
		empty.className = "px-4 py-10 text-center text-neutral";
		empty.textContent = "No students match your search.";
		table.insertBefore(empty, footer);
	}
	if (footer) {
		var pager = footer.querySelector(".join");
		if (pager) {
			pager.remove();
		}
		footer.firstElementChild.textContent = shown.length
			? "Showing 1–" + shown.length + " of " + shown.length
			: "Showing 0 of 0";
	}
	return tpl.innerHTML;
}

function solnDemoSearchUnenrolled(response, search) {
	var tpl = document.createElement("template");
	tpl.innerHTML = response;
	var rows = Array.from(tpl.content.querySelectorAll("[data-search-name]"));
	rows.forEach(function (row) {
		if (!solnDemoMatches(row, search.q)) {
			row.remove();
		}
	});
	if (rows.length && !tpl.content.querySelector("[data-search-name]")) {
		// What the server's UnenrolledRows says for an empty result, search or not.
		tpl.innerHTML = '<p class="text-center text-neutral py-8">All students are already enrolled in this classroom.</p>';
	}
	return tpl.innerHTML;
}

// Placeholder until the write fakes land: say plainly that nothing was saved.
function solnDemoWrite(path, detail) {
	if (typeof solnShowToast === "function") {
		solnShowToast("This demo can't save changes yet.", "error");
	}
}
