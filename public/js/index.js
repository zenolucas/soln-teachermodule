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
