package ui

// Demo is the static showcase's render settings (see handler.InitDemo). It lives in ui,
// not layout, so Sidebar can take it without ui importing layout back (an import cycle).
type Demo struct {
	Enabled bool
	// BookingURL is the owner's scheduling page. "" hides every "Book a demo" button.
	BookingURL string
	// DownloadURL is where the landing page's game download points. "" means the
	// embedded /public/downloads/soln.zip.
	DownloadURL string
}

// ShowBooking reports whether the "Book a demo" buttons and dialog should render.
func (d Demo) ShowBooking() bool {
	return d.Enabled && d.BookingURL != ""
}

// Download is the landing page's game download link.
func (d Demo) Download() string {
	if d.DownloadURL != "" {
		return d.DownloadURL
	}
	return "/public/downloads/soln.zip"
}
