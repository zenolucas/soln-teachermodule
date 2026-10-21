// The static showcase's client side (only loaded when DEMO_MODE is on - see
// layout.Base). There's no server behind the static site, so every htmx request is
// pointed at its pre-rendered copy (export.go wrote one per URL, at solnStaticPath -
// public/js/demopath.js), searches run over pre-rendered lists in the browser, and
// writes are faked here instead of sent.
//
// Edits a visitor makes are kept in sessionStorage under this prefix, so they last while
// the tab is open and "Reset demo" can wipe them in one go.
var SOLN_DEMO_PREFIX = "soln.demo.";

// Routes that write. The static site has nothing to send them to, so they never leave
// the browser.
var SOLN_DEMO_WRITE = /^\/(add|update|delete)\/|^\/(createclassroom|addstudents|login|register|logout)$/;

// handler.HandleGetStudents' page size (types.StudentsPageSize).
var SOLN_DEMO_PAGE_SIZE = 10;

// What layout.Base tells this script on <body>: the page's classroom (id, name,
// section; empty off classroom pages) and the export's template classroom (see
// ui.DemoTemplateClassroomID): an empty classroom whose pages stand in for any classroom
// a visitor creates.
var SOLN_DEMO_PAGE = document.body.dataset;
var SOLN_DEMO_TEMPLATE = {
	id: SOLN_DEMO_PAGE.demoTemplateId,
	name: SOLN_DEMO_PAGE.demoTemplateName,
	section: SOLN_DEMO_PAGE.demoTemplateSection,
};

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
		// The request was cancelled, so htmx won't fire htmx:afterRequest - which index.js's
		// double-submit guard waits on to re-enable the button it disables (on a 0ms timer
		// queued after this). Fire it once that's happened, as a real response would.
		setTimeout(function () {
			htmx.trigger(d.elt, "htmx:afterRequest", { elt: d.elt, successful: true });
		}, 50);
		return;
	}

	var params = Object.assign({}, d.parameters);
	var q = String(params.q || "").trim();
	delete d.target.dataset.demoUnenrolled;
	delete d.target.dataset.demoRows;
	delete d.target.dataset.demoFill;
	delete d.target.dataset.demoClassroomsAdded;
	if (path === "/students" && (q || solnDemoRosterChanged(solnDemoClassroomKey(params.classroomID)))) {
		// A search, or a roster this tab has changed: neither was pre-rendered, so the list
		// is built in the browser instead (solnDemoRenderStudents).
		evt.preventDefault();
		solnDemoRenderStudents(d.target, params, q);
		return;
	}
	if (path === "/unenrolledstudents") {
		// Searched, and adjusted for this tab's roster changes, in solnDemoUnenrolled.
		d.target.dataset.demoUnenrolled = JSON.stringify({ q: q.toLowerCase(), classroomID: params.classroomID });
	} else if (/^\/(getfractions|getwordedquestions|getmcquestions)$/.test(path)) {
		// Replay this list's edits over the pre-rendered rows (solnDemoApplyQuestions).
		d.target.dataset.demoRows = "1";
	} else if (path === "/question/edit") {
		var editID = new URLSearchParams(d.path.split("?")[1] || "").get("questionID");
		var list = solnDemoQuestionList(solnDemoPageParam("classroomID"), solnDemoPageParam("minigameID")).list;
		if (editID && editID.indexOf("demo-") === 0) {
			// Added in this tab, so there's no pre-rendered drawer: open a new-question one
			// and fill it in (see the htmx:afterSwap listener below).
			d.path = "/question/new?minigameID=" + solnDemoPageParam("minigameID") + "&classroomID=" + solnDemoPageParam("classroomID");
			d.target.dataset.demoFill = editID;
		} else if (editID && list.edits[editID]) {
			d.target.dataset.demoFill = editID;
		}
	}
	delete params.q;

	d.path = solnStaticPath(d.path, params);
	d.verb = "get";
	d.parameters = {};
});

document.body.addEventListener("htmx:beforeSwap", function (evt) {
	var target = evt.detail.target;
	if (target && target.dataset.demoRows) {
		delete target.dataset.demoRows;
		evt.detail.serverResponse = solnDemoApplyQuestions(evt.detail.serverResponse);
	} else if (target && target.dataset.demoUnenrolled) {
		var opts = JSON.parse(target.dataset.demoUnenrolled);
		delete target.dataset.demoUnenrolled;
		evt.detail.serverResponse = solnDemoUnenrolled(evt.detail.serverResponse, opts);
	}
});

function solnDemoFragment(html) {
	var tpl = document.createElement("template");
	tpl.innerHTML = html;
	return tpl;
}

function solnDemoFetch(path, params) {
	return fetch(solnStaticPath(path, params)).then(function (resp) {
		if (!resp.ok) {
			throw new Error(resp.status + " " + resp.url);
		}
		return resp.text();
	});
}

// Mirrors handler.HandleGetStudents + applyStudentQuery in the browser, for a search or a
// roster this tab has changed. It starts from the export's page=all copy of the list for
// this sort (every row, no filter), applies the roster changes, then searches, counts the
// pills, filters and pages exactly as the server does, and swaps the result in itself
// (the fetches are asynchronous, which htmx's own swap can't wait for).
function solnDemoRenderStudents(target, params, q) {
	var classroomID = String(params.classroomID);
	var roster = solnDemoRoster(solnDemoClassroomKey(classroomID));
	var sort = params.sort || "attention";
	var filter = params.filter || "all";
	var fetches = [solnDemoFetch("/students", { classroomID: classroomID, filter: "all", sort: sort, page: "all" })];
	if (roster.added.length) {
		fetches.push(solnDemoEnrolledTemplate());
	}
	Promise.all(fetches).then(function (texts) {
		var tpl = solnDemoFragment(texts[0]);
		var root = tpl.content;
		var table = root.firstElementChild;
		var footer = table.lastElementChild;
		var rows = Array.from(root.querySelectorAll("[data-search-name]")).filter(function (row) {
			return roster.removed.indexOf(row.id.replace("student-", "")) < 0;
		});
		if (texts[1]) {
			var enrolled = solnDemoFragment(texts[1]).content;
			roster.added.forEach(function (id) {
				var row = enrolled.getElementById("student-" + id);
				if (row && !rows.some(function (r) { return r.id === row.id; })) {
					// Its Remove form belongs to this classroom, not the template.
					row.querySelectorAll('input[name="classroomID"]').forEach(function (input) {
						input.value = classroomID;
					});
					solnDemoInsertRow(rows, row, sort);
				}
			});
		}

		var needle = q.toLowerCase();
		var matched = rows.filter(function (row) {
			return !needle || row.dataset.searchName.indexOf(needle) >= 0;
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
		var shown = matched.filter(function (row) {
			if (filter === "attention") {
				return row.dataset.flagged === "true";
			}
			return filter === "all" || "w" + row.dataset.world === filter;
		});
		var pageCount = Math.max(1, Math.ceil(shown.length / SOLN_DEMO_PAGE_SIZE));
		var page = Math.min(Math.max(parseInt(params.page, 10) || 1, 1), pageCount);
		var pageRows = shown.slice((page - 1) * SOLN_DEMO_PAGE_SIZE, page * SOLN_DEMO_PAGE_SIZE);

		Array.from(root.querySelectorAll("[data-search-name]")).forEach(function (row) {
			row.remove();
		});
		var empty = table.querySelector(":scope > .py-10");
		if (empty) {
			empty.remove();
		}
		pageRows.forEach(function (row) {
			table.insertBefore(row, footer);
		});
		if (!pageRows.length) {
			// StudentRows' empty states.
			empty = document.createElement("div");
			empty.className = "px-4 py-10 text-center text-neutral";
			empty.textContent = shown.length === 0 && (needle || filter !== "all") ? "No students match your search." : "No students enrolled yet.";
			table.insertBefore(empty, footer);
		}
		footer.firstElementChild.textContent = shown.length
			? "Showing " + ((page - 1) * SOLN_DEMO_PAGE_SIZE + 1) + "–" + Math.min(page * SOLN_DEMO_PAGE_SIZE, shown.length) + " of " + shown.length
			: "Showing 0 of 0";
		var pager = footer.querySelector(".join");
		if (pager) {
			pager.remove();
		}
		if (pageCount > 1) {
			// StudentRows' pager.
			pager = document.createElement("div");
			pager.className = "join";
			for (var i = 1; i <= pageCount; i++) {
				var button = document.createElement("button");
				button.type = "submit";
				button.name = "page";
				button.value = i;
				button.setAttribute("form", "students-query");
				button.className = "join-item btn btn-sm" + (i === page ? " btn-primary" : "");
				button.textContent = i;
				pager.appendChild(button);
			}
			footer.appendChild(pager);
		}

		// StudentRows' out-of-band counts: the header's "{n} enrolled" and each pill.
		var oob = { "enrolled-count": rows.length };
		Object.keys(counts).forEach(function (key) {
			oob["count-" + key] = counts[key];
		});
		Object.keys(oob).forEach(function (id) {
			var el = document.getElementById(id);
			if (el) {
				el.textContent = oob[id];
			}
		});
		root.querySelectorAll("[hx-swap-oob]").forEach(function (el) {
			el.remove();
		});
		target.innerHTML = "";
		target.appendChild(root);
		htmx.process(target);
	});
}

// Slots a newly added student's row where applyStudentQuery's sort would put it. They
// have no flags and no data yet, so: by name among the unflagged for "attention", by name
// for "name", and last for the quiz/accuracy sorts (no data sorts last).
function solnDemoInsertRow(rows, row, sort) {
	var at = rows.length;
	if (sort === "name" || sort === "attention") {
		for (var i = 0; i < rows.length; i++) {
			if (sort === "attention" && rows[i].dataset.flagged === "true") {
				continue;
			}
			if (rows[i].dataset.sortName > row.dataset.sortName) {
				at = i;
				break;
			}
		}
	}
	rows.splice(at, 0, row);
}

// The real rows of the students Add students can pick, enrolled and "not started yet":
// the export's demo=enrolled copy of the template classroom's list (exportTemplates).
function solnDemoEnrolledTemplate() {
	return solnDemoFetch("/students", { classroomID: SOLN_DEMO_TEMPLATE.id, filter: "all", sort: "name", page: "all", demo: "enrolled" });
}

// The Add students list: students this tab added to this classroom drop off it, ones
// added to another classroom show as in that one (DEC-25: they can't be picked), ones
// removed from this classroom come back as not in a class, and a search filters it.
function solnDemoUnenrolled(response, opts) {
	var tpl = solnDemoFragment(response);
	var root = tpl.content;
	var key = solnDemoClassroomKey(opts.classroomID);
	var rosters = solnDemoLoad("rosters");
	var rows = Array.from(root.querySelectorAll("[data-search-name]"));
	var sample = rows[0];
	rows.forEach(function (row) {
		var id = row.querySelector('input[name="userID"]').value;
		Object.keys(rosters).forEach(function (other) {
			if (rosters[other].added.indexOf(id) < 0) {
				return;
			}
			if (other === key) {
				row.remove();
			} else {
				row.classList.add("opacity-60");
				row.querySelector('input[name="userID"]').disabled = true;
				row.lastElementChild.textContent = "In " + rosters[other].name;
			}
		});
	});
	var roster = solnDemoRoster(key);
	if (sample) {
		roster.removed.forEach(function (id) {
			var info = roster.info[id];
			var row = sample.cloneNode(true);
			var checkbox = row.querySelector('input[name="userID"]');
			checkbox.value = id;
			checkbox.disabled = false;
			checkbox.checked = false;
			row.classList.remove("opacity-60");
			row.dataset.searchName = info.searchName;
			row.children[1].textContent = info.initials;
			row.children[2].textContent = info.name;
			row.lastElementChild.textContent = "Not in a class";
			root.appendChild(row);
		});
	}
	Array.from(root.querySelectorAll("[data-search-name]")).forEach(function (row) {
		if (opts.q && row.dataset.searchName.indexOf(opts.q) < 0) {
			row.remove();
		}
	});
	if (!root.querySelector("[data-search-name]")) {
		// What UnenrolledRows says for an empty result, search or not.
		tpl.innerHTML = '<p class="text-center text-neutral py-8">All students are already enrolled in this classroom.</p>';
	}
	return tpl.innerHTML;
}

// A write never leaves the browser: the question editor's are faked below, and the rest
// say plainly that nothing was saved.
function solnDemoWrite(path, detail) {
	if (path === "/createclassroom") {
		solnDemoCreateClassroom(detail.parameters);
	} else if (path === "/addstudents") {
		solnDemoAddStudents(detail.parameters);
	} else if (path === "/delete/student") {
		solnDemoRemoveStudent(detail.parameters);
	} else if (/^\/(add|update|delete)\//.test(path)) {
		solnDemoQuestionWrite(path, detail);
	} else if (typeof solnShowToast === "function") {
		solnShowToast("This demo can't do that.", "error");
	}
}

function solnDemoLoad(name) {
	try {
		return JSON.parse(sessionStorage.getItem(SOLN_DEMO_PREFIX + name)) || {};
	} catch (e) {
		return {};
	}
}

function solnDemoStore(name, value) {
	try {
		sessionStorage.setItem(SOLN_DEMO_PREFIX + name, JSON.stringify(value));
	} catch (e) {}
}

// A value from the current page's static path, e.g. classroomID from
// /minigame/~classroomID=1/~minigameID=5/ (demopath only ever escapes non-[A-Za-z0-9_-]
// characters, which ids never contain).
function solnDemoPageParam(name) {
	var match = location.pathname.match(new RegExp("/~" + name + "=([^/]+)"));
	return match ? match[1] : "";
}

// ---- Question editor ----
// Each list's changes live in sessionStorage under "questions", keyed
// "<classroomID>:<minigameID>": edits to pre-rendered questions (by question id), the ids
// deleted, and questions added in this tab (ids "demo-..."). A question's fields are
// {text, n1, d1, n2, d2, answer} (fraction/worded) or {text, options, correct, answer}
// (quiz), where answer is what the list's Answer column shows.
function solnDemoQuestionList(classroomID, minigameID) {
	var all = solnDemoLoad("questions");
	var key = solnDemoClassroomKey(classroomID) + ":" + minigameID;
	var list = all[key] || { edits: {}, deleted: [], added: [] };
	return {
		list: list,
		save: function () {
			all[key] = list;
			solnDemoStore("questions", all);
		},
	};
}

function solnDemoFindAdded(list, id) {
	return list.added.filter(function (q) {
		return q.id === id;
	})[0];
}

// The question id a list row opens, from its hx-get.
function solnDemoRowID(row) {
	return new URLSearchParams((row.getAttribute("hx-get") || "").split("?")[1] || "").get("questionID");
}

// The id of the row the server would highlight after an update (see
// respondFractionRowsSaved's sel); null after an add or a delete.
var solnDemoSelected = null;

// Replays the current list's edits over a pre-rendered FractionRows/QuizRows response:
// drops deleted rows, rewrites edited ones (keeping their class accuracy, as the real
// update does), appends added ones from #demo-question-row, and renumbers.
function solnDemoApplyQuestions(response) {
	var list = solnDemoQuestionList(solnDemoPageParam("classroomID"), solnDemoPageParam("minigameID")).list;
	var selected = solnDemoSelected;
	solnDemoSelected = null;
	var rowTemplate = document.getElementById("demo-question-row");
	if (!rowTemplate) {
		return response;
	}

	var tpl = document.createElement("template");
	tpl.innerHTML = response;
	var root = tpl.content;
	Array.from(root.querySelectorAll("[data-drawer-opener]")).forEach(function (row) {
		var id = solnDemoRowID(row);
		if (list.deleted.indexOf(id) >= 0) {
			row.remove();
		} else if (list.edits[id]) {
			solnDemoFillRow(row, list.edits[id]);
		}
	});
	list.added.forEach(function (q) {
		var row = rowTemplate.content.querySelector("[data-drawer-opener]").cloneNode(true);
		row.setAttribute(
			"hx-get",
			"/question/edit?minigameID=" + solnDemoPageParam("minigameID") + "&classroomID=" + solnDemoPageParam("classroomID") + "&questionID=" + q.id
		);
		row.children[3].outerHTML = '<span class="text-neutral">—</span>';
		solnDemoFillRow(row, q.fields);
		root.appendChild(row);
	});

	var rows = Array.from(root.querySelectorAll("[data-drawer-opener]"));
	var header = rowTemplate.content.firstElementChild;
	var empty = root.querySelector("p");
	if (rows.length && empty) {
		// The list was empty before this tab's additions: it needs its header row back.
		empty.remove();
		root.insertBefore(header.cloneNode(true), root.firstChild);
	} else if (!rows.length && !empty) {
		// What FractionRows/QuizRows render for an empty list.
		tpl.innerHTML = '<p class="text-center text-neutral py-8">No questions yet. Add one with Add question.</p>';
		return tpl.innerHTML;
	}
	rows.forEach(function (row, i) {
		row.firstElementChild.textContent = i + 1;
		row.classList.toggle("is-selected", solnDemoRowID(row) === selected);
	});
	return tpl.innerHTML;
}

// Writes a question's fields into a list row: [#, question, answer, accuracy, (chevron)].
function solnDemoFillRow(row, f) {
	var cell = row.children[1];
	var fractions = cell.querySelectorAll(":scope > span.inline-flex");
	if (fractions.length === 2) {
		[
			[f.n1, f.d1],
			[f.n2, f.d2],
		].forEach(function (pair, i) {
			fractions[i].firstElementChild.textContent = pair[0];
			fractions[i].lastElementChild.textContent = pair[1];
		});
	} else {
		cell.textContent = f.text;
	}
	row.children[2].textContent = f.answer;
}

// The kind of list on this page, from #question-rows' own hx-get.
function solnDemoListKind() {
	var rows = document.getElementById("question-rows");
	var get = rows ? rows.getAttribute("hx-get") : "";
	return /getmcquestions/.test(get) ? "quiz" : /getwordedquestions/.test(get) ? "worded" : "fraction";
}

// Mirrors HandleAdd*/HandleUpdate*/HandleDelete* for fraction, worded and quiz questions:
// the same validation and messages, then the list redrawn and the same
// questionSaved/questionDeleted events the server's HX-Trigger header fires.
function solnDemoQuestionWrite(path, d) {
	var p = d.parameters;
	var form = d.elt.closest("form") || d.elt;
	var kind = path.split("/")[1];
	var classroomID = solnDemoPageParam("classroomID");
	var minigameID = solnDemoPageParam("minigameID");
	var ctx = solnDemoQuestionList(classroomID, minigameID);
	var list = ctx.list;
	// A drawer opened for a question added in this tab carries its id (solnDemoFillDrawer).
	var id = form.dataset.demoQuestion || p.questionID || p.question_id || "";
	var added = id.indexOf("demo-") === 0;

	if (kind === "delete") {
		if (added) {
			list.added = list.added.filter(function (q) {
				return q.id !== id;
			});
		} else if (list.deleted.indexOf(id) < 0) {
			list.deleted.push(id);
		}
		delete list.edits[id];
		ctx.save();
		solnDemoRedrawQuestions(null, "questionDeleted", "Question deleted", false);
		return;
	}

	var result = solnDemoListKind() === "quiz" ? solnDemoQuizFields(p) : solnDemoFractionFields(p, form);
	solnDemoShowErrors(result.errors);
	if (Object.keys(result.errors).length) {
		return;
	}
	var selected = id;
	if (added) {
		solnDemoFindAdded(list, id).fields = result.fields;
	} else if (kind === "add") {
		list.added.push({ id: "demo-" + Date.now().toString(36), fields: result.fields });
		selected = null;
	} else {
		list.edits[id] = result.fields;
	}
	ctx.save();
	solnDemoRedrawQuestions(selected, "questionSaved", "Saved ✓", kind === "add" && !added && p.next === "1");
}

// handler.validateFractionField, for one numeric field.
function solnDemoWholeNumber(p, field, min, errors) {
	var raw = String(p[field] === undefined ? "" : p[field]).trim();
	var n = /^[+-]?\d+$/.test(raw) ? parseInt(raw, 10) : NaN;
	if (isNaN(n) || n < min) {
		errors[field] = "Must be a whole number, " + min + " or more.";
	}
	return n;
}

// handler.validateFractionForm (worded questions also need question_text).
function solnDemoFractionFields(p, form) {
	var errors = {};
	var f = {
		n1: solnDemoWholeNumber(p, "fraction1_numerator", 0, errors),
		d1: solnDemoWholeNumber(p, "fraction1_denominator", 1, errors),
		n2: solnDemoWholeNumber(p, "fraction2_numerator", 0, errors),
		d2: solnDemoWholeNumber(p, "fraction2_denominator", 1, errors),
		text: p.question_text || "",
	};
	if (solnDemoListKind() === "worded" && !f.text.trim()) {
		errors.question_text = "Question text can't be blank.";
	}
	// index.js's live preview already works the answer out like util.Combine does.
	solnUpdateFractionPreview(form);
	var answer = form.querySelector("[data-answer] span");
	f.answer = answer ? answer.textContent : "";
	return { fields: f, errors: errors };
}

// handler.validateQuizForm. A new question names its fields question_text/option_N with
// correct_answer=option_N; an existing one question/optionN with
// correct_answer=<choice id> (see quizOptionField/quizRadioValue).
function solnDemoQuizFields(p) {
	var errors = {};
	var isNew = p.question_text !== undefined;
	var textField = isNew ? "question_text" : "question";
	var f = { text: p[textField] || "", options: [], correct: -1 };
	if (!f.text.trim()) {
		errors[textField] = "Question text can't be blank.";
	} else if (f.text.length > 500) {
		errors[textField] = "Question text must be 500 characters or fewer.";
	}
	for (var i = 0; i < 4; i++) {
		var field = isNew ? "option_" + (i + 1) : "option" + (i + 1);
		var text = p[field] || "";
		f.options.push(text);
		if (!text.trim()) {
			errors[field] = "Option text can't be blank.";
		} else if (text.length > 255) {
			errors[field] = "Option text must be 255 characters or fewer.";
		}
		var value = isNew ? "option_" + (i + 1) : String(p["option" + (i + 1) + "_choiceID"]);
		if (p.correct_answer && p.correct_answer === value) {
			f.correct = i;
		}
	}
	if (f.correct < 0) {
		errors.correct_answer = "Choose the correct answer.";
	}
	f.answer = f.options[f.correct] || "";
	return { fields: f, errors: errors };
}

// The server answers a validation failure by re-rendering the drawer with an error under
// each field (DEC-4); this marks the same fields in place.
function solnDemoShowErrors(errors) {
	var drawer = document.getElementById("question-drawer");
	if (!drawer) {
		return;
	}
	drawer.querySelectorAll("[data-demo-error]").forEach(function (el) {
		el.remove();
	});
	drawer.querySelectorAll(".input-error, .textarea-error").forEach(function (el) {
		el.classList.remove("input-error", "textarea-error");
	});
	Object.keys(errors).forEach(function (field) {
		var message = document.createElement("span");
		message.dataset.demoError = "";
		message.className = "text-[11px] text-error";
		message.textContent = errors[field];
		if (field === "correct_answer") {
			var options = drawer.querySelectorAll("[data-quiz-option]");
			var last = options[options.length - 1];
			last.parentNode.insertBefore(message, last.nextSibling);
			return;
		}
		var el = drawer.querySelector('[name="' + field + '"]');
		if (!el) {
			return;
		}
		el.classList.add(el.tagName === "TEXTAREA" ? "textarea-error" : "input-error");
		if (el.type === "number") {
			el.parentNode.appendChild(message);
		} else {
			el.parentNode.insertBefore(message, el.nextSibling);
		}
	});
}

// Re-fetches the list (solnDemoApplyQuestions replays the edits over it), then fires the
// event the server's HX-Trigger would have - and, for the quiz's "Add & next", the
// questionReopenNew its HX-Trigger-After-Settle would have.
function solnDemoRedrawQuestions(selected, event, message, reopen) {
	var rows = document.getElementById("question-rows");
	solnDemoSelected = selected;
	var done = htmx.ajax("GET", rows.getAttribute("hx-get"), "#question-rows");
	(done && done.then ? done : Promise.resolve()).then(function () {
		htmx.trigger(document.body, event, { message: message });
		if (reopen) {
			htmx.trigger(document.body, "questionReopenNew", {
				url: "/question/new?minigameID=" + solnDemoPageParam("minigameID") + "&classroomID=" + solnDemoPageParam("classroomID"),
			});
		}
	});
}

// Fills a pre-rendered drawer with a question's edited fields. A question added in this
// tab has no drawer of its own, so it opens in a new-question drawer (see
// htmx:configRequest), which is turned into an edit one here: its title, a Save button,
// and a Delete form.
document.body.addEventListener("htmx:afterSwap", function (evt) {
	var drawer = evt.detail.target;
	if (!drawer || drawer.id !== "question-drawer" || !drawer.dataset.demoFill) {
		return;
	}
	var id = drawer.dataset.demoFill;
	delete drawer.dataset.demoFill;
	var list = solnDemoQuestionList(solnDemoPageParam("classroomID"), solnDemoPageParam("minigameID")).list;
	var added = solnDemoFindAdded(list, id);
	var f = added ? added.fields : list.edits[id];
	if (!f) {
		return;
	}

	var quizForm = drawer.querySelector("#quiz-question-form");
	if (quizForm) {
		quizForm.querySelector("textarea").value = f.text;
		for (var i = 0; i < 4; i++) {
			var option = quizForm.querySelector('[data-quiz-option="' + i + '"]');
			option.querySelector("input[type=text]").value = f.options[i];
			option.querySelector("input[type=radio]").checked = i === f.correct;
		}
		solnUpdateQuizPreview(quizForm);
	} else {
		var form = drawer.querySelector("#fraction-question-form");
		var text = drawer.querySelector("textarea[name=question_text]");
		if (text) {
			text.value = f.text;
		}
		[
			["fraction1_numerator", f.n1],
			["fraction1_denominator", f.d1],
			["fraction2_numerator", f.n2],
			["fraction2_denominator", f.d2],
		].forEach(function (pair) {
			form.querySelector('[name="' + pair[0] + '"]').value = pair[1];
		});
		solnUpdateFractionPreview(form);
	}

	var row = Array.from(document.querySelectorAll("#question-rows [data-drawer-opener]")).filter(function (r) {
		return solnDemoRowID(r) === id;
	})[0];
	var title = drawer.querySelector(".text-base.font-bold");
	if (row && title) {
		title.textContent = "Edit question " + row.firstElementChild.textContent;
	}
	if (!added) {
		return;
	}

	(quizForm || drawer.querySelector("#fraction-question-form")).dataset.demoQuestion = id;
	var footer = drawer.firstElementChild.lastElementChild;
	var next = footer.querySelector('button[name="next"]');
	if (next) {
		next.remove();
	}
	footer.lastElementChild.textContent = "Save changes";
	var kind = solnDemoListKind();
	var del = document.createElement("form");
	del.setAttribute("hx-post", kind === "quiz" ? "/delete/mcquestions" : kind === "worded" ? "/delete/worded" : "/delete/fractions");
	del.setAttribute("hx-confirm", "Delete this question? Students' recorded answers to it will also be deleted.");
	del.dataset.demoQuestion = id;
	del.innerHTML =
		'<button type="submit" class="btn btn-ghost text-error btn-sm"><i class="fa-solid fa-trash" aria-hidden="true"></i> Delete</button>';
	footer.insertBefore(del, footer.firstChild);
	htmx.process(del);
	if (typeof solnSetQuizDrafting === "function") {
		solnSetQuizDrafting(false);
	}
});

// The classroom's Minigames page: each scene's question count, adjusted for this tab's
// additions and deletions.
document.querySelectorAll("[data-question-count]").forEach(function (el) {
	var list = solnDemoQuestionList(solnDemoPageParam("classroom_id"), el.dataset.questionCount).list;
	var n = parseInt(el.textContent, 10) + list.added.length - list.deleted.length;
	if (list.added.length || list.deleted.length) {
		el.textContent = n + " questions";
	}
});

// ---- Classrooms and rosters ----
// sessionStorage holds the classrooms this tab created ("classrooms": [{id: "demo-...",
// name, section}]), each classroom's roster changes ("rosters", by classroom id: students
// added, students removed, plus what's needed to list a removed student again), and
// "context": which classroom the template classroom's pages are standing in for.

function solnDemoClassrooms() {
	return solnDemoLoad("classrooms").list || [];
}

function solnDemoFindClassroom(id) {
	return solnDemoClassrooms().filter(function (c) {
		return c.id === id;
	})[0];
}

function solnDemoContext() {
	var context = solnDemoLoad("context");
	return context.id ? context : null;
}

function solnDemoOnTemplate() {
	return SOLN_DEMO_PAGE.demoClassroomId === SOLN_DEMO_TEMPLATE.id;
}

// The classroom a template classroom id really means here: whichever one its pages are
// standing in for. Every other id means itself.
function solnDemoClassroomKey(id) {
	var context = String(id) === SOLN_DEMO_TEMPLATE.id && solnDemoContext();
	return context ? context.id : String(id);
}

function solnDemoRoster(key) {
	var roster = solnDemoLoad("rosters")[key] || {};
	return { name: roster.name || "", added: roster.added || [], removed: roster.removed || [], info: roster.info || {} };
}

function solnDemoSaveRoster(key, roster) {
	var rosters = solnDemoLoad("rosters");
	rosters[key] = roster;
	solnDemoStore("rosters", rosters);
}

function solnDemoRosterChanged(key) {
	var roster = solnDemoRoster(key);
	return roster.added.length > 0 || roster.removed.length > 0;
}

// The current page's classroom as a context, for the template classroom's pages.
function solnDemoPageClassroom() {
	if (solnDemoOnTemplate()) {
		return solnDemoContext() || SOLN_DEMO_TEMPLATE;
	}
	return { id: SOLN_DEMO_PAGE.demoClassroomId, name: SOLN_DEMO_PAGE.demoClassroomName, section: SOLN_DEMO_PAGE.demoClassroomSection };
}

// Mirrors HandleClassroomCreate: validateCreateClassroom's checks and messages, under
// each field as CreateClassForm shows them, then the redirect to /home.
function solnDemoCreateClassroom(p) {
	var values = {
		classname: String(p.classname || "").trim(),
		section: String(p.section || "").trim(),
		description: String(p.description || "").trim(),
	};
	var errors = {};
	if (!values.classname) {
		errors.classname = "Class name is required.";
	} else if (values.classname.length > 100) {
		errors.classname = "Class name must be 100 characters or fewer.";
	}
	if (!values.section) {
		errors.section = "Section is required.";
	} else if (values.section.length > 100) {
		errors.section = "Section must be 100 characters or fewer.";
	}
	if (values.description.length > 200) {
		errors.description = "Description must be 200 characters or fewer.";
	}

	var dialog = document.getElementById("create-classroom-modal");
	dialog.querySelectorAll("[data-demo-error]").forEach(function (el) {
		el.remove();
	});
	dialog.querySelectorAll(".input-error, .textarea-error").forEach(function (el) {
		el.classList.remove("input-error", "textarea-error");
	});
	Object.keys(errors).forEach(function (field) {
		var el = dialog.querySelector('[name="' + field + '"]');
		el.classList.add(el.tagName === "TEXTAREA" ? "textarea-error" : "input-error");
		var message = document.createElement("div");
		message.dataset.demoError = "";
		message.className = "label py-0 pt-1";
		message.innerHTML = '<span class="label-text-alt text-error flex items-center gap-1.5"><i class="fa-solid fa-circle-exclamation" aria-hidden="true"></i></span>';
		message.firstElementChild.appendChild(document.createTextNode(errors[field]));
		el.closest("label").appendChild(message);
	});
	if (Object.keys(errors).length) {
		return;
	}

	var list = solnDemoClassrooms();
	list.push({ id: "demo-" + Date.now().toString(36), name: values.classname, section: values.section });
	solnDemoStore("classrooms", { list: list });
	location.href = "/home/";
}

// Mirrors HandleAddStudents: enrol the checked students, then go to the Students page.
// A student this tab removed from the classroom just comes back with their own row.
function solnDemoAddStudents(p) {
	var classroomID = String(p.classroomID);
	var key = solnDemoClassroomKey(classroomID);
	var roster = solnDemoRoster(key);
	roster.name = solnDemoPageClassroom().name;
	[].concat(p.userID || []).forEach(function (id) {
		id = String(id);
		if (roster.removed.indexOf(id) >= 0) {
			roster.removed.splice(roster.removed.indexOf(id), 1);
		} else if (roster.added.indexOf(id) < 0) {
			roster.added.push(id);
		}
	});
	solnDemoSaveRoster(key, roster);
	location.href = solnStaticPath("/classroom/students", { classroom_id: classroomID });
}

// Mirrors HandleUnenrollStudent, whose empty response removes the row; the roster keeps
// it removed on later visits, and remembers the student so Add students can list them.
function solnDemoRemoveStudent(p) {
	var id = String(p.studentID);
	var key = solnDemoClassroomKey(p.classroomID);
	var roster = solnDemoRoster(key);
	var row = document.getElementById("student-" + id);
	if (roster.added.indexOf(id) >= 0) {
		roster.added.splice(roster.added.indexOf(id), 1);
	} else if (roster.removed.indexOf(id) < 0) {
		roster.removed.push(id);
		if (row) {
			roster.info[id] = {
				name: row.querySelector(".truncate").textContent.trim(),
				initials: row.querySelector(".rounded-full").textContent.trim(),
				searchName: row.dataset.searchName,
			};
		}
	}
	roster.name = solnDemoPageClassroom().name;
	solnDemoSaveRoster(key, roster);
	if (row) {
		row.remove();
	}
}

// ui.tileInitials: the first letters of the first and last words.
function solnDemoTileInitials(name) {
	var words = name.trim().split(/\s+/).filter(Boolean);
	var first = function (w) {
		return w ? w.charAt(0).toUpperCase() : "";
	};
	if (!words.length) {
		return "?";
	}
	return words.length === 1 ? first(words[0]) + first(words[0]) : first(words[0]) + first(words[words.length - 1]);
}

// Puts a classroom's name and section in place of the template classroom's, in root's
// text, tooltips and labels, and in the sidebar's initials tiles.
function solnDemoRename(root, room) {
	var swap = function (text) {
		return text.split(SOLN_DEMO_TEMPLATE.name).join(room.name).split(SOLN_DEMO_TEMPLATE.section).join(room.section);
	};
	var walker = document.createTreeWalker(root, NodeFilter.SHOW_TEXT);
	var tile = solnDemoTileInitials(SOLN_DEMO_TEMPLATE.name);
	for (var node = walker.nextNode(); node; node = walker.nextNode()) {
		if (node.nodeValue.trim() === tile && node.parentNode.closest && node.parentNode.closest("a.tooltip")) {
			node.nodeValue = node.nodeValue.replace(tile, solnDemoTileInitials(room.name));
		} else if (node.nodeValue.indexOf(SOLN_DEMO_TEMPLATE.name) >= 0 || node.nodeValue.indexOf(SOLN_DEMO_TEMPLATE.section) >= 0) {
			node.nodeValue = swap(node.nodeValue);
		}
	}
	(root.querySelectorAll ? Array.from(root.querySelectorAll("[aria-label], [data-tip], [title]")) : []).forEach(function (el) {
		["aria-label", "data-tip", "title"].forEach(function (attr) {
			if (el.hasAttribute(attr)) {
				el.setAttribute(attr, swap(el.getAttribute(attr)));
			}
		});
	});
}

var SOLN_DEMO_TEMPLATE_LINK = new RegExp("(~classroom_?(?:id|ID)=)" + SOLN_DEMO_TEMPLATE.id + "/");

// On the template classroom's pages: show the classroom they're standing in for. That's
// one this tab created, or - for the page of a student added to a real classroom - that
// real classroom, whose own pages its links then go back to (and whose own sidebar entry
// the template's renamed one replaces).
function solnDemoPatchTemplatePage() {
	var context = solnDemoContext();
	if (!solnDemoOnTemplate() || !context) {
		return;
	}
	var real = context.id.indexOf("demo-") !== 0;
	if (real) {
		// (This runs after every swap, so links it already pointed back are marked and left be.)
		var entries = function (selector) {
			return Array.from(document.querySelectorAll(selector))
				.filter(function (a) {
					return !a.closest("[data-demo-classroom]") && !a.dataset.demoStandin;
				})
				.map(function (a) {
					return a.parentNode;
				});
		};
		// Expanded row, then collapsed tile, in both lists: the renamed template entry
		// takes the real one's place in the sidebar.
		var standins = entries('aside a[href*="~classroom_id=' + SOLN_DEMO_TEMPLATE.id + '/"]');
		entries('aside a[href*="~classroom_id=' + context.id + '/"]').forEach(function (entry, i) {
			if (standins[i]) {
				entry.replaceWith(standins[i]);
			} else {
				entry.remove();
			}
		});
		document.querySelectorAll("a[href]").forEach(function (a) {
			var href = a.getAttribute("href");
			if (!a.closest("[data-demo-classroom]") && SOLN_DEMO_TEMPLATE_LINK.test(href)) {
				a.setAttribute("href", href.replace(new RegExp(SOLN_DEMO_TEMPLATE_LINK.source, "g"), "$1" + context.id + "/"));
				a.dataset.demoStandin = "1";
			}
		});
	}
	solnDemoRename(document.body, context);
	document.title = document.title.split(SOLN_DEMO_TEMPLATE.name).join(context.name);
	// A student page's Previous/Next would step through the template's other students.
	if (/^\/student\/score\//.test(location.pathname)) {
		document.querySelectorAll("a, span").forEach(function (el) {
			var text = el.textContent.trim();
			if ((text === "Previous" || text === "Next") && el.classList.contains("h-[34px]")) {
				el.remove();
			}
		});
	}
}

solnDemoPatchTemplatePage();
document.body.addEventListener("htmx:afterSettle", solnDemoPatchTemplatePage);

// A link into the template classroom says which classroom it's standing in for: one this
// tab created (its card or sidebar entry), or the classroom the visitor is in (a student
// they added: their "View scores" leads to the template's copy of that student).
document.body.addEventListener(
	"click",
	function (evt) {
		var a = evt.target.closest("a[href]");
		if (!a) {
			return;
		}
		var created = a.closest("[data-demo-classroom]");
		if (created) {
			var room = solnDemoFindClassroom(created.dataset.demoClassroom);
			if (room) {
				solnDemoStore("context", room);
			}
		} else if (SOLN_DEMO_TEMPLATE_LINK.test(a.getAttribute("href")) && SOLN_DEMO_PAGE.demoClassroomId && !solnDemoOnTemplate()) {
			solnDemoStore("context", solnDemoPageClassroom());
		}
	},
	true
);

// Classrooms this tab created get a card on /home and an entry in the sidebar: clones of
// the template classroom's, from the export's demo=template copies of both lists.
document.body.addEventListener("htmx:afterSwap", function (evt) {
	var target = evt.detail.target;
	var get = (target && target.getAttribute("hx-get")) || "";
	var created = solnDemoClassrooms().filter(function (room) {
		var context = solnDemoContext();
		return !(solnDemoOnTemplate() && context && context.id === room.id);
	});
	// htmx fires afterSwap again for each out-of-band swap in the same response (the
	// /home extras), so add them once per load.
	if (!created.length || !/^\/getclassrooms(_menu)?$/.test(get.split("?")[0]) || target.dataset.demoClassroomsAdded) {
		return;
	}
	target.dataset.demoClassroomsAdded = "1";
	if (get === "/getclassrooms") {
		solnDemoFetch("/getclassrooms", { demo: "template" }).then(function (html) {
			var card = solnDemoFragment(html).content.querySelector('a[href*="~classroom_id=' + SOLN_DEMO_TEMPLATE.id + '/"]').closest(".card");
			var grid = target.querySelector(".grid");
			if (!grid) {
				return;
			}
			created.forEach(function (room) {
				var el = card.cloneNode(true);
				el.dataset.demoClassroom = room.id;
				solnDemoRename(el, room);
				el.querySelector(".text-xl.font-bold").textContent = solnDemoRoster(room.id).added.length;
				grid.appendChild(el);
			});
			var counts = document.getElementById("home-counts");
			if (counts) {
				counts.textContent = counts.textContent.replace(/(\d+) classrooms/, function (m, n) {
					return parseInt(n, 10) + created.length + " classrooms";
				});
			}
		});
	} else if (get.indexOf("/getclassrooms_menu") === 0) {
		solnDemoFetch("/getclassrooms_menu", { active: "home", demo: "template" }).then(function (html) {
			var entries = Array.from(solnDemoFragment(html).content.querySelectorAll('a[href*="~classroom_id=' + SOLN_DEMO_TEMPLATE.id + '/"]')).map(function (a) {
				return a.parentNode;
			});
			created.forEach(function (room) {
				entries.forEach(function (entry) {
					var el = entry.cloneNode(true);
					el.dataset.demoClassroom = room.id;
					solnDemoRename(el, room);
					target.appendChild(el);
				});
			});
		});
	}
});
