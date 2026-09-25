// A small toast, bottom-right. kind is "error" (the default) or "success" - used for
// both the htmx-failure case below (originally the only caller, see FE-05: htmx 1.x
// doesn't swap a non-2xx response into its target by default, so a failed fragment
// load previously just stayed blank with nothing telling the teacher anything went
// wrong) and the question drawer's Saved/Deleted confirmations (DEC-4, T3.3). A
// success toast is shorter-lived (~3s) than an error one (6s), since it's confirming
// something that already worked rather than something the teacher needs time to read
// and act on.
//
// pointer-events-none (T3.4): the question drawer is also docked to the bottom-right
// corner, so a save's "Saved ✓" toast can still be up when the drawer reopens (e.g.
// clicking "+ Add question" right after saving) and would otherwise sit on top of the
// drawer's own footer buttons, silently eating the next click. A toast has no
// interactive content of its own, so it's safe to let clicks pass straight through it
// to whatever's underneath.
function solnShowToast(message, kind) {
	kind = kind === "success" ? "success" : "error";

	var existing = document.getElementById("soln-toast");
	if (existing) {
		existing.remove();
	}

	var toast = document.createElement("div");
	toast.id = "soln-toast";
	toast.className = "toast toast-end toast-bottom z-[1000] pointer-events-none";
	toast.innerHTML =
		'<div role="alert" class="alert ' +
		(kind === "success" ? "alert-success" : "alert-error") +
		' shadow-lg"><span></span></div>';
	toast.querySelector("span").textContent = message;
	document.body.appendChild(toast);

	setTimeout(
		function () {
			toast.remove();
		},
		kind === "success" ? 3000 : 6000
	);
}

// Kept as a thin wrapper so the existing htmx:responseError/htmx:sendError call sites
// below don't need to change.
function solnShowErrorToast(message) {
	solnShowToast(message, "error");
}

document.body.addEventListener("htmx:responseError", function (evt) {
	solnShowErrorToast("Couldn't load that. Please refresh and try again.");
});

document.body.addEventListener("htmx:sendError", function (evt) {
	solnShowErrorToast("Couldn't reach the server. Check your connection and try again.");
});

// Every per-question statistics chart (fraction, worded, quiz) used to come from the
// server as a nearly-identical inline <script> block defining numbered
// getClassStatisticsN/renderChartN globals - one pair per question, duplicated three
// times across handler/statistics.go for one Go-side difference (attempts vs choices)
// each time (see FE-34). This single renderer replaces all of that: the server now
// only emits a <canvas data-chart-type data-chart-url> per question, and this fetches
// and draws each one found in a freshly-swapped htmx fragment.
function solnInitCharts(root) {
	root.querySelectorAll("canvas[data-chart-url]").forEach(function (canvas) {
		// hx-trigger="load" fires once per element, but stay defensive: initializing
		// the same canvas twice would leak a Chart.js instance onto it.
		if (canvas.dataset.chartInitialized) {
			return;
		}
		canvas.dataset.chartInitialized = "true";

		fetch(canvas.dataset.chartUrl)
			.then(function (response) {
				return response.json();
			})
			.then(function (results) {
				// A question with no attempts/responses yet returned either an empty
				// array or a single all-zero row - Chart.js still drew axes for it, so
				// a teacher saw an empty box with no explanation of whether that meant
				// "no one has tried this" or "something's broken" (see FE-20).
				if (solnChartIsEmpty(canvas.dataset.chartType, results)) {
					solnRenderNoDataMessage(canvas);
					return;
				}
				if (canvas.dataset.chartType === "attempts") {
					solnRenderAttemptsChart(canvas, results);
				} else if (canvas.dataset.chartType === "choices") {
					solnRenderChoicesChart(canvas, results);
				}
			});
	});
}

function solnChartIsEmpty(chartType, results) {
	if (!results || results.length === 0) {
		return true;
	}
	if (chartType === "attempts") {
		return results.every(function (item) {
			return !item.num_right_attempts && !item.num_wrong_attempts;
		});
	}
	if (chartType === "choices") {
		return results.every(function (item) {
			return !item.count;
		});
	}
	return false;
}

function solnRenderNoDataMessage(canvas) {
	var message = document.createElement("p");
	message.className = "text-white text-opacity-60";
	message.textContent = "No responses yet.";
	canvas.replaceWith(message);
}

// "attempts" charts: the fraction/worded per-question breakdown - always exactly one
// result row, with a right/wrong attempt count each.
function solnRenderAttemptsChart(canvas, results) {
	var right = results.map(function (item) {
		return item.num_right_attempts;
	});
	var wrong = results.map(function (item) {
		return item.num_wrong_attempts;
	});
	Chart.defaults.font.size = 30;
	new Chart(canvas.getContext("2d"), {
		type: "bar",
		data: {
			labels: ["Correct Attempts", "Wrong Attempts"],
			datasets: [
				{
					data: right.concat(wrong),
					borderWidth: 1,
					categoryPercentage: 0.3,
					backgroundColor: ["rgba(75, 192, 192, 0.5)", "rgba(255, 99, 132, 0.5)"],
				},
			],
		},
		options: {
			indexAxis: "x",
			scales: {
				y: {
					beginAtZero: true,
					ticks: { stepSize: 1 },
				},
			},
			plugins: { legend: { display: false } },
		},
	});
}

// "choices" charts: the quiz per-question answer distribution - one result row per
// choice actually chosen at least once, colored via data-colors (computed
// server-side by setColors, in the same order as the question's own choices).
function solnRenderChoicesChart(canvas, results) {
	var labels = results.map(function (item) {
		return item.choice;
	});
	var count = results.map(function (item) {
		return item.count;
	});
	var colors = JSON.parse(canvas.dataset.colors);
	Chart.defaults.font.size = 30;
	new Chart(canvas.getContext("2d"), {
		type: "bar",
		data: {
			labels: labels,
			datasets: [
				{
					label: "number of responses",
					data: count,
					backgroundColor: colors,
					borderWidth: 1,
				},
			],
		},
		options: {
			indexAxis: "y",
			scales: {
				x: {
					beginAtZero: true,
					ticks: { stepSize: 1 },
				},
			},
			plugins: { legend: { display: false } },
		},
	});
}

// Native <dialog> modals replace the old checkbox-hack pattern (an
// `<input type="checkbox" class="modal-toggle">` whose :checked state showed a sibling
// `.modal` div via CSS) - that had no keyboard support at all: Esc did nothing, there
// was no focus trapping, and clicking outside the box didn't close it (see FE-43). A
// dialog shown via showModal() gets all three from the browser for free; only the
// open/close triggers need wiring up here. Delegated on document.body (rather than one
// listener per trigger) so it keeps working for triggers that arrive later via an htmx
// swap without needing to be re-bound.
document.body.addEventListener("click", function (evt) {
	var opener = evt.target.closest("[data-modal-open]");
	if (opener) {
		var dialog = document.getElementById(opener.dataset.modalOpen);
		if (dialog) {
			dialog.showModal();
		}
		return;
	}

	var closer = evt.target.closest("[data-modal-close]");
	if (closer) {
		var dialog = closer.closest("dialog");
		if (dialog) {
			dialog.close();
		}
	}
});

// The Classroom Overview's "Add students" action (01 §1b) links to
// /classroom/students?classroom_id=...#add instead of opening the modal directly,
// since #modal_add_students only exists on the Students page (T2.4b) - this opens it
// once that page has loaded, if the hash says to (T4.7).
if (location.hash === "#add") {
	var addStudentsDialog = document.getElementById("modal_add_students");
	if (addStudentsDialog) {
		addStudentsDialog.showModal();
	}
}

// A form that re-renders itself with a validation error (e.g. Create Classroom) needs
// its modal to come back open after the htmx swap replaces it, the same "stay open on
// error" behaviour the checkbox-hack version had via a server-rendered `checked`
// attribute (see FE-02). A server-rendered data-autoopen="true" on the dialog signals
// that here instead, since a plain `open` attribute on a <dialog> shows it without the
// backdrop or Esc/focus handling that only showModal() turns on.
function solnOpenPendingModals(root) {
	// For an outerHTML swap targeted directly at the dialog itself (hx-target pointing
	// at the dialog's own id, as CreateClassForm does so the whole dialog - not just
	// the form inside it - gets replaced), root *is* the dialog, and
	// querySelectorAll only matches descendants, never the root element itself -
	// missing this left the modal closed after a real error swap even though the
	// error message rendered correctly.
	var matches = root.matches && root.matches('dialog[data-autoopen="true"]') ? [root] : [];
	matches
		.concat(Array.from(root.querySelectorAll('dialog[data-autoopen="true"]')))
		.forEach(function (dialog) {
			dialog.showModal();
		});
}

// Makes a table[data-sortable]'s columns clickable to re-sort its rows client-side, no
// server round trip - the class-wide question tables (see FE-30, D6) arrive
// pre-sorted weakest-question-first, but a teacher may want to re-sort by a different
// column (e.g. most attempts, or alphabetically by question).
function solnInitSortableTables(root) {
	var tables = root.matches && root.matches("table[data-sortable]") ? [root] : [];
	tables = tables.concat(Array.from(root.querySelectorAll("table[data-sortable]")));
	tables.forEach(function (table) {
		if (table.dataset.sortableInitialized) {
			return;
		}
		table.dataset.sortableInitialized = "true";

		var headers = Array.from(table.querySelectorAll("thead th[data-sort]"));
		headers.forEach(function (th, index) {
			th.style.cursor = "pointer";
			th.addEventListener("click", function () {
				var ascending = th.dataset.sortDir !== "asc";
				headers.forEach(function (h) {
					delete h.dataset.sortDir;
				});
				th.dataset.sortDir = ascending ? "asc" : "desc";

				var tbody = table.querySelector("tbody");
				var rows = Array.from(tbody.querySelectorAll("tr"));
				var type = th.dataset.sort;
				rows.sort(function (a, b) {
					var aText = a.children[index].textContent.trim();
					var bText = b.children[index].textContent.trim();
					if (type === "number") {
						var aNum = parseFloat(aText) || 0;
						var bNum = parseFloat(bText) || 0;
						return ascending ? aNum - bNum : bNum - aNum;
					}
					return ascending ? aText.localeCompare(bText) : bText.localeCompare(aText);
				});
				rows.forEach(function (row) {
					tbody.appendChild(row);
				});
			});
		});
	});
}

// The add-students modal's select-all (with an indeterminate state when some but not
// all rows are checked), the "n available"/"n selected" counts, the "Add n students"
// button label, and bg-primary/10 on checked rows (see addstudents.templ, T2.4b).
// Recomputed from scratch on every change rather than incrementally tracked, since
// it's cheap at classroom-roster size and stays correct no matter what triggered it -
// a checkbox toggle, or the row set itself changing after a search.
//
// A disabled checkbox (a student already enrolled elsewhere, DEC-25) is excluded from
// every count and from what "select all" touches - selecting them would just be
// silently skipped by AddStudents, so counting them as available/selectable here would
// be misleading.
function solnUpdateAddStudentsModal() {
	var modal = document.getElementById("modal_add_students");
	if (!modal) {
		return;
	}
	var selectAll = modal.querySelector("#select-all");
	var selectable = Array.from(modal.querySelectorAll('input[name="userID"]:not([disabled])'));
	var checked = selectable.filter(function (cb) {
		return cb.checked;
	});

	if (selectAll) {
		selectAll.checked = selectable.length > 0 && checked.length === selectable.length;
		selectAll.indeterminate = checked.length > 0 && checked.length < selectable.length;
	}

	modal.querySelectorAll("[data-student-row]").forEach(function (row) {
		var cb = row.querySelector('input[name="userID"]');
		row.classList.toggle("bg-primary/10", !!(cb && cb.checked));
	});

	var availableCount = modal.querySelector("#available-count");
	if (availableCount) {
		availableCount.textContent = selectable.length + " available";
	}
	var selectedCount = modal.querySelector("#selected-count");
	if (selectedCount) {
		selectedCount.textContent = checked.length + " selected";
	}
	var submitBtn = modal.querySelector("#add-students-submit");
	if (submitBtn) {
		submitBtn.textContent = checked.length > 0 ? "Add " + checked.length + " students" : "Add students";
	}
}

document.body.addEventListener("change", function (evt) {
	var modal = evt.target.closest("#modal_add_students");
	if (!modal) {
		return;
	}
	if (evt.target.id === "select-all") {
		modal.querySelectorAll('input[name="userID"]:not([disabled])').forEach(function (cb) {
			cb.checked = evt.target.checked;
		});
	}
	solnUpdateAddStudentsModal();
});

// htmx:afterSettle (not the plain "load" event, which won't fire for content that
// arrives via an htmx swap) covers both the initial hx-trigger="load" fragment and
// any later swap into the same target.
document.body.addEventListener("htmx:afterSettle", function (evt) {
	solnInitCharts(evt.detail.elt);
	solnOpenPendingModals(evt.detail.elt);
	solnInitSortableTables(evt.detail.elt);
	solnUpdateAddStudentsModal();
});

// Disables a form's submit button just after it's submitted, so a double-click (or an
// impatient second click while a request is still in flight) can't create a duplicate
// classroom, question, or account (see FE-25). This listens for the native `submit`
// event itself - which still fires for an htmx-managed form, since htmx only calls
// preventDefault on it, not stopPropagation - so one listener covers every form in the
// app, htmx-managed or a plain POST, without changing each one individually.
document.body.addEventListener("submit", function (evt) {
	var form = evt.target;
	var submitter = evt.submitter || form.querySelector('button[type="submit"]');
	if (!submitter || submitter.disabled) {
		return;
	}

	// Deferred to the next tick rather than disabled synchronously in this handler -
	// htmx handles this same event, in this same dispatch, to read the form and start
	// its request; mutating the submitter's disabled state before that finishes is a
	// known cross-browser footgun that can interfere with the in-flight submission
	// itself (reproduced here: it crashed the tab under headless Chromium during
	// verification). A same-tick disable isn't actually needed anyway - nothing
	// user-triggered can land between this handler returning and the next tick.
	setTimeout(function () {
		submitter.disabled = true;
	}, 0);

	// A plain form submission navigates away, so there's nothing left to re-enable.
	// An htmx-managed form almost always replaces itself entirely on response
	// (hx-swap="outerHTML"), which discards this disabled button along with it - a
	// fresh, enabled one comes back in the swapped-in HTML. Re-enable defensively
	// anyway, in case a future form targets somewhere other than itself and so
	// survives the request with this same button still in the DOM.
	if (form.hasAttribute("hx-post") || form.hasAttribute("hx-get")) {
		form.addEventListener(
			"htmx:afterRequest",
			function () {
				submitter.disabled = false;
			},
			{ once: true }
		);
	}
});

// Toggles the sidebar between expanded and collapsed (see sidebar.templ, T1.6). The
// actual expanded/collapsed markup is pure CSS (group-data-[sidebar=.../shell:
// variants keyed off this same attribute) - this just flips document.body.dataset
// and remembers the choice, the same pre-paint script in layout/app.templ reads on
// every later page load so there's no flash of the wrong state. try/catch: a full or
// blocked localStorage (private browsing) must not break the toggle itself, just the
// "remember it next time" part.
document.body.addEventListener("click", function (evt) {
	var toggle = evt.target.closest("[data-sidebar-toggle]");
	if (!toggle) {
		return;
	}
	var next = document.body.dataset.sidebar === "collapsed" ? "expanded" : "collapsed";
	document.body.dataset.sidebar = next;
	try {
		localStorage.setItem("soln.sidebar", next);
	} catch (e) {}
});

// The Minigames page's "All worlds / World 1 / World 2 / World 3" filter (see
// classroom.templ's Minigames, T2.1) is client-side only - every world's scenes are
// already on the page, so there's nothing to fetch, just [data-world] blocks to
// show/hide. Sets style.display directly rather than the `hidden` attribute/property -
// each block also carries Tailwind's `flex` utility, and `[hidden]` loses to a `.flex`
// class rule of equal specificity that happens to come later in the stylesheet, so
// `block.hidden = true` silently doesn't hide anything. An inline style always wins on
// specificity regardless of class order.
document.body.addEventListener("click", function (evt) {
	var filterBtn = evt.target.closest("[data-world-filter]");
	if (!filterBtn) {
		return;
	}
	var value = filterBtn.dataset.worldFilter;
	var join = filterBtn.closest(".join");
	if (join) {
		join.querySelectorAll("[data-world-filter]").forEach(function (btn) {
			btn.classList.toggle("btn-active", btn === filterBtn);
		});
	}
	document.querySelectorAll("[data-world]").forEach(function (block) {
		var hide = value !== "all" && block.dataset.world !== value;
		block.style.display = hide ? "none" : "";
	});
});

// A live "n / 200" counter next to any [data-count-for] textarea/input (currently just
// the create-classroom modal's Description field, T2.3) - delegated on document.body,
// like the rest of this file's listeners, so it keeps working after an htmx swap
// re-renders the field with a validation error, without needing to be re-bound.
document.body.addEventListener("input", function (evt) {
	var field = evt.target.closest("[data-count-for]");
	if (!field) {
		return;
	}
	var label = field.closest("label");
	var counter = label && label.querySelector("[data-count]");
	if (counter) {
		counter.textContent = field.value.length;
	}
});

// The question editor drawer (DEC-3). Inert until T3.4/T3.7 render the markup it
// targets - #question-drawer, [data-drawer-opener] rows/buttons, [data-drawer-close].
// It's a plain <aside>, not a daisyUI drawer (a second drawer-toggle checkbox inside
// the shell complicates focus handling): closed means `hidden` and empty, opened by an
// hx-get response swapping into it (hx-swap="innerHTML" targeting the drawer itself).
document.body.addEventListener("htmx:afterSwap", function (evt) {
	if (evt.detail.target && evt.detail.target.id === "question-drawer") {
		var drawer = evt.detail.target;
		drawer.hidden = false;
		var firstField = drawer.querySelector("input, textarea, select");
		if (firstField) {
			firstField.focus();
		}
		solnSetQuizDrafting(solnQuizNewDrawerActive(drawer));
	}
});

function solnCloseQuestionDrawer() {
	var drawer = document.getElementById("question-drawer");
	if (!drawer || drawer.hidden) {
		return;
	}
	drawer.hidden = true;
	drawer.innerHTML = "";
	solnSetQuizDrafting(false);
	var opener = document.querySelector("[data-drawer-opener].is-selected");
	if (opener) {
		opener.classList.remove("is-selected");
		opener.focus();
	}
}

document.body.addEventListener("click", function (evt) {
	if (evt.target.closest("[data-drawer-close]")) {
		solnCloseQuestionDrawer();
		return;
	}
	var opener = evt.target.closest("[data-drawer-opener]");
	if (opener) {
		document.querySelectorAll("[data-drawer-opener].is-selected").forEach(function (el) {
			el.classList.remove("is-selected");
		});
		opener.classList.add("is-selected");
	}
});

document.addEventListener("keydown", function (evt) {
	if (evt.key === "Escape") {
		solnCloseQuestionDrawer();
	}
});

// DEC-4: every question add/update/delete responds with the whole #question-rows body
// plus an HX-Trigger header naming which happened - questionSaved (add/update) or
// questionDeleted (delete), each carrying {message}. The toast + drawer-close reaction
// is the same regardless of which form or question kind triggered it.
document.body.addEventListener("questionSaved", function (evt) {
	solnShowToast((evt.detail && evt.detail.message) || "Saved", "success");
	solnCloseQuestionDrawer();
});

document.body.addEventListener("questionDeleted", function (evt) {
	solnShowToast((evt.detail && evt.detail.message) || "Deleted", "success");
	solnCloseQuestionDrawer();
});

// DEC-4: a 422 (validation failure) re-renders the drawer in place - via
// HX-Retarget/HX-Reswap on the response, which htmx itself honors - instead of being
// treated as a failed request. Without this, htmx 1.x only swaps a 2xx response by
// default, so the inline field errors would never reach the DOM, and the
// htmx:responseError listener above would also fire its own generic error toast on
// top of them.
document.body.addEventListener("htmx:beforeSwap", function (evt) {
	if (evt.detail.xhr.status === 422) {
		evt.detail.shouldSwap = true;
		evt.detail.isError = false;
	}
});

// Live answer + in-game preview (T3.6, DEC-18). Purely client-side - mirrors
// util.Combine/Frac.String() in JS just enough for the drawer's own display, since
// nothing here is ever posted (the server computes its own answer independently on
// save). solnStackedFrac reuses fraction.templ's stackedFraction classes verbatim, so
// no new Tailwind coverage is needed for this JS-generated markup.
function solnGcd(a, b) {
	a = Math.abs(a);
	b = Math.abs(b);
	while (b) {
		var t = b;
		b = a % b;
		a = t;
	}
	return a || 1;
}

function solnStackedFrac(num, den) {
	return (
		'<span class="inline-flex flex-col items-center leading-none font-semibold min-w-4">' +
		"<span>" + num + "</span>" +
		'<span class="self-stretch h-[1.5px] bg-current my-[3px]"></span>' +
		"<span>" + den + "</span></span>"
	);
}

function solnUpdateFractionPreview(form) {
	var inputs = form.querySelectorAll("input[type=number]");
	if (inputs.length < 4) {
		return;
	}
	var num1 = parseInt(inputs[0].value, 10);
	var den1 = parseInt(inputs[1].value, 10);
	var num2 = parseInt(inputs[2].value, 10);
	var den2 = parseInt(inputs[3].value, 10);
	var op = form.dataset.op;

	var answerSpan = form.querySelector("[data-answer] span");
	if (answerSpan) {
		if (isNaN(num1) || isNaN(den1) || isNaN(num2) || isNaN(den2) || den1 < 1 || den2 < 1) {
			answerSpan.textContent = "—";
		} else {
			var resultNum =
				op === "-" || op === "−" ? num1 * den2 - num2 * den1 : num1 * den2 + num2 * den1;
			var resultDen = den1 * den2;
			var g = solnGcd(resultNum, resultDen);
			resultNum = resultNum / g;
			resultDen = resultDen / g;
			answerSpan.textContent = resultNum === 0 ? "0" : resultDen === 1 ? String(resultNum) : resultNum + "/" + resultDen;
		}
	}

	var drawer = document.getElementById("question-drawer");
	if (!drawer) {
		return;
	}
	var eqEl = drawer.querySelector("[data-preview-eq]");
	if (eqEl) {
		eqEl.innerHTML = solnStackedFrac(num1, den1) + "<span>" + op + "</span>" + solnStackedFrac(num2, den2);
	}
	var textEl = drawer.querySelector("[data-preview-text]");
	if (textEl) {
		var textarea = drawer.querySelector("textarea[name=question_text]");
		textEl.textContent = textarea
			? textarea.value
			: "What is " + num1 + "/" + den1 + " " + op + " " + num2 + "/" + den2 + "?";
	}
}

document.body.addEventListener("input", function (evt) {
	// Checked first and unconditionally: the quiz drawer's own question textarea is
	// also named question_text in new mode (matching AddMCQuestions), which would
	// otherwise also match the fraction/worded "textarea outside its form" selector
	// below and get routed to the wrong preview updater (found via T3.8 verification -
	// the quiz preview's speech text never updated because of this).
	var quizForm = evt.target.closest("#quiz-question-form");
	if (quizForm) {
		solnUpdateQuizPreview(quizForm);
		return;
	}

	var form = evt.target.closest("#fraction-question-form");
	var textarea = evt.target.closest("#question-drawer textarea[name=question_text]");
	if (form) {
		solnUpdateFractionPreview(form);
	} else if (textarea) {
		var ownForm = document.getElementById("fraction-question-form");
		if (ownForm) {
			solnUpdateFractionPreview(ownForm);
		}
	}
});

// Quiz drawer preview + radio tint (T3.8) - mirrors solnUpdateFractionPreview above,
// but keyed by data-quiz-option/data-preview-choice index pairs instead of a fixed
// field order, since a quiz question has 4 independent choices rather than 2 fraction
// pairs.
function solnUpdateQuizPreview(form) {
	var drawer = document.getElementById("question-drawer");
	if (!drawer) {
		return;
	}

	for (var i = 0; i < 4; i++) {
		var row = form.querySelector('[data-quiz-option="' + i + '"]');
		if (!row) {
			continue;
		}
		var radio = row.querySelector("input[type=radio]");
		var textInput = row.querySelector("input[type=text]");
		var checked = !!(radio && radio.checked);
		if (textInput) {
			textInput.classList.toggle("bg-success/20", checked);
		}
		var previewChoice = drawer.querySelector('[data-preview-choice="' + i + '"]');
		if (previewChoice) {
			previewChoice.classList.toggle("border-info", checked);
			previewChoice.classList.toggle("border-base-100", !checked);
			previewChoice.textContent =
				String.fromCharCode(65 + i) + ". " + (textInput ? textInput.value : "");
		}
	}

	var questionField = form.querySelector('textarea[name="question_text"], textarea[name="question"]');
	var previewText = drawer.querySelector("[data-preview-text]");
	if (questionField && previewText) {
		previewText.textContent = questionField.value;
	}
}

// "Add & next" (T3.8): the response's HX-Trigger-After-Settle names this event once
// the swap has settled - by then questionSaved (above) has already closed the drawer,
// so this reopens a fresh one via the URL the server sent.
document.body.addEventListener("questionReopenNew", function (evt) {
	var url = evt.detail && evt.detail.url;
	if (url && window.htmx) {
		htmx.ajax("GET", url, "#question-drawer");
	}
});

// While a new quiz question's drawer is open, the list gets an extra "drafting" row
// and the header's "+ Add question" button dims (01 §1f) - both purely visual, so
// plain DOM manipulation rather than a server round-trip.
function solnQuizNewDrawerActive(drawer) {
	return !!(
		drawer.querySelector("#quiz-question-form") &&
		drawer.querySelector('#quiz-question-form input[name="minigameID"]')
	);
}

function solnSetQuizDrafting(active) {
	var addBtn = document.querySelector('[hx-get*="/question/new"]');
	if (addBtn) {
		addBtn.classList.toggle("opacity-[.55]", active);
	}
	var existing = document.getElementById("quiz-drafting-row");
	if (active && !existing) {
		var container = document.getElementById("question-rows");
		if (container) {
			var row = document.createElement("div");
			row.id = "quiz-drafting-row";
			row.className =
				"grid grid-cols-[40px_1fr_90px_130px] items-center px-4 h-[52px] text-sm font-semibold bg-info/10 ring-1 ring-inset ring-info text-info";
			row.innerHTML = "<span></span><span>New question — drafting…</span>";
			container.appendChild(row);
		}
	} else if (!active && existing) {
		existing.remove();
	}
}
