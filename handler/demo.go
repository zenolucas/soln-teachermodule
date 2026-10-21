package handler

import (
	"os"

	"soln-teachermodule/view/ui"
)

// demo is the static-showcase configuration, set once by InitDemo. It only changes how
// pages render (a sample-data banner, "Book a demo" buttons, landing-page calls to
// action, public/js/demo.js) - it never changes what a handler reads or writes. It's
// meant for the static export of the portal, not for a live deployment.
var demo ui.Demo

// InitDemo reads DEMO_MODE, DEMO_BOOKING_URL and DEMO_DOWNLOAD_URL from the environment.
// Like InitSessionStore, it must run after database.InitializeDatabase has loaded .env.
func InitDemo() {
	demo = ui.Demo{
		Enabled:     os.Getenv("DEMO_MODE") == "true",
		BookingURL:  os.Getenv("DEMO_BOOKING_URL"),
		DownloadURL: os.Getenv("DEMO_DOWNLOAD_URL"),
	}
}
