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
				if (canvas.dataset.chartType === "attempts") {
					solnRenderAttemptsChart(canvas, results);
				} else if (canvas.dataset.chartType === "choices") {
					solnRenderChoicesChart(canvas, results);
				}
			});
	});
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

// htmx:afterSettle (not the plain "load" event, which won't fire for content that
// arrives via an htmx swap) covers both the initial hx-trigger="load" fragment and
// any later swap into the same target.
document.body.addEventListener("htmx:afterSettle", function (evt) {
	solnInitCharts(evt.detail.elt);
});
