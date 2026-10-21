// The static showcase's client side (only loaded when DEMO_MODE is on - see
// layout.Base). Edits a visitor makes are kept in sessionStorage under this prefix, so
// they last while the tab is open and "Reset demo" can wipe them in one go.
var SOLN_DEMO_PREFIX = "soln.demo.";

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
