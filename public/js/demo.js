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
	delete d.target.dataset.demoRows;
	delete d.target.dataset.demoFill;
	if (/^\/(getfractions|getwordedquestions|getmcquestions)$/.test(path)) {
		// Replay this list's edits over the pre-rendered rows (solnDemoApplyQuestions).
		d.target.dataset.demoRows = "1";
	} else if (path === "/question/edit") {
		var editID = new URLSearchParams(d.path.split("?")[1] || "").get("questionID");
		var list = solnDemoQuestionList(solnDemoPageParam("classroomID"), solnDemoPageParam("minigameID")).list;
		if (editID && editID.indexOf("demo-") === 0) {
			// Added in this tab, so there's no pre-rendered drawer: open a new-question one
			// and fill it in (solnDemoFillDrawer).
			d.path = "/question/new?minigameID=" + solnDemoPageParam("minigameID") + "&classroomID=" + solnDemoPageParam("classroomID");
			d.target.dataset.demoFill = editID;
		} else if (editID && list.edits[editID]) {
			d.target.dataset.demoFill = editID;
		}
	}
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
	if (target && target.dataset.demoRows) {
		delete target.dataset.demoRows;
		evt.detail.serverResponse = solnDemoApplyQuestions(evt.detail.serverResponse);
		return;
	}
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

// A write never leaves the browser: the question editor's are faked below, and the rest
// say plainly that nothing was saved.
function solnDemoWrite(path, detail) {
	if (/^\/(add|update|delete)\//.test(path)) {
		solnDemoQuestionWrite(path, detail);
		return;
	}
	if (typeof solnShowToast === "function") {
		solnShowToast("This demo can't save changes yet.", "error");
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
	var key = classroomID + ":" + minigameID;
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
