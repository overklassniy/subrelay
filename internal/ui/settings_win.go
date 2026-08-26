// Package ui (settings_win.go) implements the settings window and the
// first-run wizard. All configuration is edited through these windows;
// no manual config files are expected.
//
// The settings window groups related fields into tabs (general, ports,
// URL test, headers) and validates numeric and URL fields inline,
// disabling Save while any field is invalid.
package ui

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/validation"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"subrelay/internal/autostart"
	"subrelay/internal/config"
	"subrelay/internal/i18n"
)

// SettingsCallbacks holds actions triggered when settings are applied.
type SettingsCallbacks struct {
	OnApply func()
	// OnCheckUpdates is invoked when the user clicks the "Check for
	// updates" button. The caller receives an onResult callback that it
	// must invoke exactly once with the verdict text and, when non-empty,
	// the release page URL to display as a clickable link.
	OnCheckUpdates func(onResult func(text string, releaseURL string))
}

// SettingsWindow manages the settings dialog lifecycle.
type SettingsWindow struct {
	window    fyne.Window
	settings  *config.Settings
	callbacks SettingsCallbacks
}

// NewSettingsWindow creates a settings window bound to the given app.
//
// Args:
//   - a: the Fyne application.
//   - settings: the application settings.
//   - cb: callbacks invoked on apply.
//
// Returns:
//   - A pointer to the new SettingsWindow.
func NewSettingsWindow(a fyne.App, settings *config.Settings, cb SettingsCallbacks) *SettingsWindow {
	w := a.NewWindow(i18n.T("settings.title"))
	w.SetContent(container.NewVBox())
	sw := &SettingsWindow{window: w, settings: settings, callbacks: cb}
	w.SetCloseIntercept(func() {
		w.Hide()
	})
	return sw
}

// fieldGroup collects the entries built for a settings window instance so
// the Save button can be enabled or disabled based on their combined
// validation state.
type fieldGroup struct {
	saveBtn *widget.Button
	invalid map[*widget.Entry]bool
}

func newFieldGroup(saveBtn *widget.Button) *fieldGroup {
	return &fieldGroup{saveBtn: saveBtn, invalid: map[*widget.Entry]bool{}}
}

// watch registers a validated entry so its state contributes to the
// shared Save button's enabled state.
func (g *fieldGroup) watch(e *widget.Entry) *widget.Entry {
	g.invalid[e] = false
	e.SetOnValidationChanged(func(err error) {
		g.invalid[e] = err != nil
		g.refresh()
	})
	return e
}

// refresh disables the Save button when any watched entry is invalid.
func (g *fieldGroup) refresh() {
	for _, bad := range g.invalid {
		if bad {
			g.saveBtn.Disable()
			return
		}
	}
	g.saveBtn.Enable()
}

// Show opens the settings window with the current settings loaded into
// editable fields, grouped into tabs.
func (s *SettingsWindow) Show() {
	s.settings.Lock()
	current := s.settings.Snapshot()
	s.settings.Unlock()

	saveBtn := widget.NewButton(i18n.T("settings.save"), nil)
	saveBtn.Importance = widget.HighImportance
	cancelBtn := widget.NewButton(i18n.T("settings.cancel"), func() {
		s.window.Hide()
	})
	group := newFieldGroup(saveBtn)

	urlEntry := group.watch(urlEntryWithValidator(current.SubscriptionURL))
	intervalEntry := group.watch(intEntry(current.UpdateIntervalMin, 1, 1440))
	autostartCheck := widget.NewCheck(i18n.T("settings.autostart"), nil)
	autostartCheck.SetChecked(current.Autostart)
	langSelect := widget.NewSelect([]string{"ru", "en"}, nil)
	langSelect.SetSelected(current.Language)

	generalForm := widget.NewForm(
		widget.NewFormItem(i18n.T("settings.subscription.url"), urlEntry),
		widget.NewFormItem(i18n.T("settings.update.interval"), intervalEntry),
		widget.NewFormItem(i18n.T("settings.autostart"), autostartCheck),
		widget.NewFormItem(i18n.T("settings.language"), langSelect),
	)

	// Update check section: a centered non-full-width button, a centered
	// verdict label, and an optional centered hyperlink to the release
	// page. Double padding above the button increases the visual
	// separation from the settings form.
	verdictLabel := widget.NewLabel("")
	verdictLabel.Wrapping = fyne.TextWrapWord
	verdictLabel.Alignment = fyne.TextAlignCenter
	verdictLabel.Hide()

	verdictLink := widget.NewHyperlink(i18n.T("update.open_page"), nil)
	verdictLink.Wrapping = fyne.TextWrapWord
	verdictLink.Alignment = fyne.TextAlignCenter
	verdictLink.Hide()

	checkUpdatesBtn := widget.NewButton(i18n.T("settings.check_updates"), func() {
		if s.callbacks.OnCheckUpdates == nil {
			return
		}
		verdictLabel.SetText(i18n.T("update.checking"))
		verdictLabel.Show()
		verdictLink.Hide()
		s.callbacks.OnCheckUpdates(func(text string, releaseURL string) {
			fyne.Do(func() {
				verdictLabel.SetText(text)
				verdictLabel.Show()
				if releaseURL != "" {
					if u, err := url.Parse(releaseURL); err == nil {
						verdictLink.URL = u
						verdictLink.Show()
					}
				} else {
					verdictLink.Hide()
				}
			})
		})
	})

	updateSection := container.NewVBox(
		container.NewPadded(container.NewPadded(container.NewCenter(checkUpdatesBtn))),
		container.NewCenter(verdictLabel),
		container.NewCenter(verdictLink),
	)
	// A padded empty label acts as a fixed-height spacer between the
	// settings form and the update check section.
	spacer := container.NewPadded(widget.NewLabel(""))
	generalTab := container.NewVBox(generalForm, spacer, updateSection)

	ruSocks := group.watch(intEntry(current.BalancerPorts.RUSocks, 1, 65535))
	ruHTTP := group.watch(intEntry(current.BalancerPorts.RUHTTP, 1, 65535))
	nonruSocks := group.watch(intEntry(current.BalancerPorts.NonRUSocks, 1, 65535))
	nonruHTTP := group.watch(intEntry(current.BalancerPorts.NonRUHTTP, 1, 65535))
	socksStart := group.watch(intEntry(current.SOCKSPortStart, 1, 65535))
	httpStart := group.watch(intEntry(current.HTTPPortStart, 1, 65535))

	ruCard := widget.NewCard(i18n.T("settings.section.balancer.ru"), "", widget.NewForm(
		widget.NewFormItem(i18n.T("settings.ru.socks"), ruSocks),
		widget.NewFormItem(i18n.T("settings.ru.http"), ruHTTP),
	))
	nonruCard := widget.NewCard(i18n.T("settings.section.balancer.nonru"), "", widget.NewForm(
		widget.NewFormItem(i18n.T("settings.nonru.socks"), nonruSocks),
		widget.NewFormItem(i18n.T("settings.nonru.http"), nonruHTTP),
	))
	rangeCard := widget.NewCard(i18n.T("settings.section.perNode"), "", widget.NewForm(
		widget.NewFormItem(i18n.T("settings.socks.start"), socksStart),
		widget.NewFormItem(i18n.T("settings.http.start"), httpStart),
	))
	portsTab := container.NewVBox(ruCard, nonruCard, rangeCard)

	urlInterval := group.watch(intEntry(current.URLTest.IntervalSec, 1, 3600))
	urlTolerance := group.watch(intEntry(current.URLTest.ToleranceMs, 0, 10000))
	urlURL := group.watch(urlEntryWithValidator(current.URLTest.URL))
	urltestForm := widget.NewForm(
		widget.NewFormItem(i18n.T("settings.urltest.interval"), urlInterval),
		widget.NewFormItem(i18n.T("settings.urltest.tolerance"), urlTolerance),
		widget.NewFormItem(i18n.T("settings.urltest.url"), urlURL),
	)

	uaEntry := widget.NewEntry()
	uaEntry.SetText(current.Headers.UserAgent)
	hwidEntry := widget.NewEntry()
	hwidEntry.SetText(current.Headers.XHWID)
	osEntry := widget.NewEntry()
	osEntry.SetText(current.Headers.XDeviceOS)
	verOS := widget.NewEntry()
	verOS.SetText(current.Headers.XVerOS)
	modelEntry := widget.NewEntry()
	modelEntry.SetText(current.Headers.XDeviceModel)
	appVer := widget.NewEntry()
	appVer.SetText(current.Headers.XAppVersion)
	headersForm := widget.NewForm(
		widget.NewFormItem("User-Agent", uaEntry),
		widget.NewFormItem("X-HWID", hwidEntry),
		widget.NewFormItem("X-Device-OS", osEntry),
		widget.NewFormItem("X-Ver-OS", verOS),
		widget.NewFormItem("X-Device-Model", modelEntry),
		widget.NewFormItem("X-App-Version", appVer),
	)

	tabs := container.NewAppTabs(
		container.NewTabItem(i18n.T("settings.tab.general"), container.NewVScroll(container.NewPadded(generalTab))),
		container.NewTabItem(i18n.T("settings.tab.ports"), container.NewVScroll(container.NewPadded(portsTab))),
		container.NewTabItem(i18n.T("settings.tab.urltest"), container.NewVScroll(container.NewPadded(urltestForm))),
		container.NewTabItem(i18n.T("settings.tab.headers"), container.NewVScroll(container.NewPadded(headersForm))),
	)

	saveBtn.OnTapped = func() {
		s.apply(
			urlEntry.Text,
			intervalEntry.Text,
			autostartCheck.Checked,
			langSelect.Selected,
			ruSocks.Text, ruHTTP.Text, nonruSocks.Text, nonruHTTP.Text,
			socksStart.Text, httpStart.Text,
			urlInterval.Text, urlTolerance.Text, urlURL.Text,
			uaEntry.Text, hwidEntry.Text, osEntry.Text, verOS.Text,
			modelEntry.Text, appVer.Text,
		)
		s.window.Hide()
	}

	buttons := container.NewBorder(nil, nil, nil, container.NewHBox(cancelBtn, saveBtn))
	s.window.SetContent(container.NewBorder(nil, buttons, nil, nil, tabs))
	s.window.Resize(fyne.NewSize(560, 620))
	s.window.Show()
}

// apply parses the form fields, updates settings, persists them, and
// triggers the OnApply callback.
func (s *SettingsWindow) apply(
	urlStr, intervalStr string, autostartVal bool, lang string,
	ruSocks, ruHTTP, nonruSocks, nonruHTTP, socksStart, httpStart,
	urlInterval, urlTolerance, urlURL,
	ua, hwid, osVal, verOS, model, appVer string,
) {
	s.settings.Lock()
	s.settings.SubscriptionURL = urlStr
	s.settings.UpdateIntervalMin = atoiOr(intervalStr, config.DefaultUpdateIntervalMin)
	s.settings.Autostart = autostartVal
	s.settings.Language = lang
	s.settings.BalancerPorts.RUSocks = atoiOr(ruSocks, config.DefaultRUSocksPort)
	s.settings.BalancerPorts.RUHTTP = atoiOr(ruHTTP, config.DefaultRUHTTPPort)
	s.settings.BalancerPorts.NonRUSocks = atoiOr(nonruSocks, config.DefaultNonRUSocksPort)
	s.settings.BalancerPorts.NonRUHTTP = atoiOr(nonruHTTP, config.DefaultNonRUHTTPPort)
	s.settings.SOCKSPortStart = atoiOr(socksStart, config.DefaultSOCKSPortStart)
	s.settings.HTTPPortStart = atoiOr(httpStart, config.DefaultHTTPPortStart)
	s.settings.URLTest.IntervalSec = atoiOr(urlInterval, config.DefaultURLTestIntervalSec)
	s.settings.URLTest.ToleranceMs = atoiOr(urlTolerance, config.DefaultURLTestToleranceMs)
	s.settings.URLTest.URL = strOr(urlURL, config.DefaultURLTestURL)
	s.settings.Headers.UserAgent = strOr(ua, config.DefaultUserAgent)
	if hwid != "" {
		s.settings.Headers.XHWID = hwid
	}
	s.settings.Headers.XDeviceOS = strOr(osVal, config.DefaultDeviceOS)
	s.settings.Headers.XVerOS = strOr(verOS, config.DefaultVerOS)
	s.settings.Headers.XDeviceModel = strOr(model, config.DefaultDeviceModel)
	s.settings.Headers.XAppVersion = strOr(appVer, config.DefaultAppVersion)
	s.settings.Unlock()

	if autostartVal {
		_ = autostart.Enable()
	} else {
		_ = autostart.Disable()
	}

	i18n.SetLanguage(lang)
	_ = s.settings.Save()
	s.callbacks.OnApply()
}

// ShowWizard opens a minimal first-run dialog asking only for the
// subscription URL. On confirm it applies the URL and triggers OnApply.
//
// Args:
//   - parent: the parent window for the dialog.
func (s *SettingsWindow) ShowWizard(parent fyne.Window) {
	urlEntry := urlEntryWithValidator("")
	urlEntry.SetPlaceHolder("https://example.com/sub")

	items := []*widget.FormItem{
		{Text: i18n.T("settings.subscription.url"), Widget: urlEntry},
	}

	dialog.ShowForm(
		i18n.T("wizard.title"),
		i18n.T("settings.connect"),
		i18n.T("settings.cancel"),
		items,
		func(ok bool) {
			// Hide the temporary parent window regardless of the
			// dialog outcome; the app lives in the tray.
			parent.Hide()
			if !ok || urlEntry.Text == "" {
				return
			}
			s.settings.Lock()
			s.settings.SubscriptionURL = urlEntry.Text
			s.settings.Unlock()
			_ = s.settings.Save()
			s.callbacks.OnApply()
		},
		parent,
	)
}

// intEntry creates a single-line entry pre-filled with an integer and
// validated to lie within [min, max].
func intEntry(v, min, max int) *widget.Entry {
	e := widget.NewEntry()
	e.SetText(strconv.Itoa(v))
	e.Validator = intRangeValidator(min, max)
	return e
}

// urlEntryWithValidator creates a single-line entry pre-filled with a URL
// and validated to start with http:// or https://. Wrapping is set to
// TextWrapOff and Scroll to ScrollVerticalOnly so the entry does not show
// a horizontal scrollbar while keeping a compact MinSize (the entry's
// internal scroll container only scrolls vertically, so no horizontal
// scrollbar is rendered).
func urlEntryWithValidator(v string) *widget.Entry {
	e := widget.NewEntry()
	e.Wrapping = fyne.TextWrapOff
	e.Scroll = fyne.ScrollVerticalOnly
	e.SetText(v)
	e.Validator = validation.NewRegexp(`^https?://\S+$`, i18n.T("validation.url"))
	return e
}

// intRangeValidator returns a validator that requires the input to parse
// as an integer within [min, max].
func intRangeValidator(min, max int) fyne.StringValidator {
	return func(s string) error {
		s = strings.TrimSpace(s)
		if s == "" {
			return errors.New(i18n.T("validation.required"))
		}
		n, err := strconv.Atoi(s)
		if err != nil {
			return errors.New(i18n.T("validation.int"))
		}
		if n < min || n > max {
			return fmt.Errorf(i18n.T("validation.range"), min, max)
		}
		return nil
	}
}

// atoiOr parses s as an int, returning def on failure.
func atoiOr(s string, def int) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}

// strOr returns s when non-empty, otherwise def.
func strOr(s, def string) string {
	if s == "" {
		return def
	}
	return s
}
