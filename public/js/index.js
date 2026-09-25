// htmx 1.x doesn't swap a non-2xx response into its target by default, so a failed
// fragment load (e.g. the students list, a chart, a question card) previously just
// stayed blank with nothing telling the teacher anything went wrong (see FE-05). This
// shows a small toast instead, for both a response the server answered with an error
// status and a request that never got a response at all (network drop, server down).
function solnShowErrorToast(message) {
	var existing = document.getElementById("soln-error-toast");
	if (existing) {
		existing.remove();
	}

	var toast = document.createElement("div");
	toast.id = "soln-error-toast";
	toast.className = "toast toast-end toast-bottom z-[1000]";
	toast.innerHTML =
		'<div role="alert" class="alert alert-error shadow-lg"><span></span></div>';
	toast.querySelector("span").textContent = message;
	document.body.appendChild(toast);

	setTimeout(function () {
		toast.remove();
	}, 6000);
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

// htmx:afterSettle (not the plain "load" event, which won't fire for content that
// arrives via an htmx swap) covers both the initial hx-trigger="load" fragment and
// any later swap into the same target.
document.body.addEventListener("htmx:afterSettle", function (evt) {
	solnInitCharts(evt.detail.elt);
	solnOpenPendingModals(evt.detail.elt);
	solnInitSortableTables(evt.detail.elt);
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
