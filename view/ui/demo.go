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

// The template classroom: an empty classroom the static export adds after its main crawl
// (export.go's exportTemplates). public/js/demo.js shows its pages for any classroom a
// visitor creates, and its students' pages for students a visitor adds, with the name
// and section swapped for the right ones. layout.Base hands these to demo.js on <body>.
const (
	DemoTemplateClassroomID = "900"
	DemoTemplateName        = "Your new classroom"
	DemoTemplateSection     = "New section"
)

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
