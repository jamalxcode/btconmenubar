// BTCMenuBar shows the Bitcoin price in the macOS menu bar.
// Price: heat.sala.company/pulse.json (CoinGecko data stored by heat), falling back to CoinGecko directly.
// Updates: "Check for Updates…" installs the newest GitHub release of jamalxcode/btconmenubar.
package main

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"fyne.io/systray"
)

// version is set at build time: -ldflags "-X main.version=v1.2.3".
var version = "dev"

type app struct {
	mu       sync.Mutex
	cfg      Config
	last     *Quote
	pending  *Release
	updating bool

	refreshNow chan struct{}

	mPrice, mChange, mUpdated, mSource *systray.MenuItem
	mInstall                           *systray.MenuItem
	mCheck                             *systray.MenuItem
}

func main() {
	a := &app{cfg: loadConfig(), refreshNow: make(chan struct{}, 1)}
	systray.Run(a.onReady, func() {})
}

func (a *app) onReady() {
	systray.SetTitle("₿ …")
	setMenuBarFontSize(a.cfg.FontSize)
	systray.SetTooltip("Bitcoin price")

	a.mPrice = systray.AddMenuItem("Loading…", "")
	a.mPrice.Disable()
	a.mChange = systray.AddMenuItem("24h: –", "")
	a.mChange.Disable()
	a.mUpdated = systray.AddMenuItem("Updated: –", "")
	a.mUpdated.Disable()
	a.mSource = systray.AddMenuItem("Source: –", "")
	a.mSource.Disable()
	systray.AddSeparator()

	mRefresh := systray.AddMenuItem("Refresh Now", "Fetch the price again")
	mHeat := systray.AddMenuItem("Open heat.sala.company", "")
	mGecko := systray.AddMenuItem("Open on CoinGecko", "")
	systray.AddSeparator()

	mCurrency := systray.AddMenuItem("Currency", "")
	curItems := map[string]*systray.MenuItem{}
	for _, c := range currencies {
		curItems[c] = mCurrency.AddSubMenuItemCheckbox(strings.ToUpper(c), "", c == a.cfg.Currency)
	}
	mInterval := systray.AddMenuItem("Refresh Every", "")
	ivItems := map[time.Duration]*systray.MenuItem{}
	for _, d := range intervals {
		ivItems[d] = mInterval.AddSubMenuItemCheckbox(humanDuration(d), "", int(d.Seconds()) == a.cfg.IntervalSeconds)
	}
	mFont := systray.AddMenuItem("Text Size", "Size of the price in the menu bar")
	fontItems := map[float64]*systray.MenuItem{}
	for _, f := range fontSizes {
		fontItems[f.Pt] = mFont.AddSubMenuItemCheckbox(f.Label, "", f.Pt == a.cfg.FontSize)
	}
	mShowChange := systray.AddMenuItemCheckbox("Show 24h Change in Menu Bar", "", a.cfg.ShowChange)
	mLogin := systray.AddMenuItemCheckbox("Launch at Login", "", launchAtLoginEnabled())
	systray.AddSeparator()

	a.mInstall = systray.AddMenuItem("", "")
	a.mInstall.Hide()
	a.mCheck = systray.AddMenuItem("Check for Updates…", "Look for a newer version on GitHub")
	mAbout := systray.AddMenuItem("BTCMenuBar "+version, "")
	mQuit := systray.AddMenuItem("Quit", "")

	go a.priceLoop()
	go a.autoUpdateLoop()

	on := func(item *systray.MenuItem, f func()) {
		go func() {
			for range item.ClickedCh {
				f()
			}
		}()
	}
	on(mRefresh, a.kick)
	on(mHeat, func() { openURL("https://heat.sala.company") })
	on(mGecko, func() { openURL("https://www.coingecko.com/en/coins/bitcoin") })
	for c, item := range curItems {
		c := c
		on(item, func() {
			a.mu.Lock()
			a.cfg.Currency = c
			a.save()
			a.mu.Unlock()
			for k, it := range curItems {
				setChecked(it, k == c)
			}
			a.kick()
		})
	}
	for d, item := range ivItems {
		d := d
		on(item, func() {
			a.mu.Lock()
			a.cfg.IntervalSeconds = int(d.Seconds())
			a.save()
			a.mu.Unlock()
			for k, it := range ivItems {
				setChecked(it, k == d)
			}
			a.kick()
		})
	}
	for pt, item := range fontItems {
		pt := pt
		on(item, func() {
			a.mu.Lock()
			a.cfg.FontSize = pt
			a.save()
			a.mu.Unlock()
			for k, it := range fontItems {
				setChecked(it, k == pt)
			}
			setMenuBarFontSize(pt)
		})
	}
	on(mShowChange, func() {
		a.mu.Lock()
		a.cfg.ShowChange = !a.cfg.ShowChange
		a.save()
		show, q := a.cfg.ShowChange, a.last
		a.mu.Unlock()
		setChecked(mShowChange, show)
		if q != nil {
			systray.SetTitle(menuBarTitle(*q, show))
		}
	})
	on(mLogin, func() {
		want := !launchAtLoginEnabled()
		if err := setLaunchAtLogin(want); err != nil {
			alert("BTCMenuBar", "Could not change Launch at Login: "+err.Error())
		}
		setChecked(mLogin, launchAtLoginEnabled())
	})
	on(a.mInstall, func() { a.install(true) })
	on(a.mCheck, func() { a.checkForUpdates(true) })
	on(mAbout, func() { openURL("https://github.com/" + repo) })
	on(mQuit, systray.Quit)
}

func setChecked(it *systray.MenuItem, on bool) {
	if on {
		it.Check()
	} else {
		it.Uncheck()
	}
}

func (a *app) save() { _ = saveConfig(a.cfg) } // caller holds a.mu

func (a *app) kick() {
	select {
	case a.refreshNow <- struct{}{}:
	default:
	}
}

/* ---------- price ---------- */

func (a *app) priceLoop() {
	for {
		a.refresh()
		a.mu.Lock()
		wait := time.Duration(a.cfg.IntervalSeconds) * time.Second
		a.mu.Unlock()
		select {
		case <-time.After(wait):
		case <-a.refreshNow:
		}
	}
}

func (a *app) refresh() {
	a.mu.Lock()
	cur, show, font := a.cfg.Currency, a.cfg.ShowChange, a.cfg.FontSize
	a.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	q, err := fetchQuote(ctx, cur)
	if err != nil {
		a.mu.Lock()
		last := a.last
		a.mu.Unlock()
		if last == nil || last.Currency != cur {
			systray.SetTitle("₿ –")
			a.mPrice.SetTitle("Price unavailable")
		}
		a.mUpdated.SetTitle("Error: " + shorten(err.Error(), 70))
		return
	}
	a.mu.Lock()
	a.last = &q
	a.mu.Unlock()

	systray.SetTitle(menuBarTitle(q, show))
	setMenuBarFontSize(font) // the font sticks, but set it again in case macOS rebuilt the item
	systray.SetTooltip("Bitcoin " + formatPrice(q.Price, q.Currency))
	a.mPrice.SetTitle("Bitcoin: " + formatPrice(q.Price, q.Currency))
	if q.HasChange {
		a.mChange.SetTitle("24h: " + formatChange(q.Change24h))
	} else {
		a.mChange.SetTitle("24h: –")
	}
	a.mUpdated.SetTitle("Updated: " + q.At.Local().Format("Jan 2, 15:04"))
	a.mSource.SetTitle("Source: " + q.Source)
}

/* ---------- updates ---------- */

func (a *app) autoUpdateLoop() {
	time.Sleep(10 * time.Second)
	for {
		a.mu.Lock()
		auto := a.cfg.AutoCheckUpdate
		a.mu.Unlock()
		if auto {
			a.checkForUpdates(false)
		}
		time.Sleep(12 * time.Hour)
	}
}

// checkForUpdates looks for a newer release. interactive = the user clicked the button.
func (a *app) checkForUpdates(interactive bool) {
	if interactive {
		a.mCheck.SetTitle("Checking for Updates…")
		a.mCheck.Disable()
		defer func() {
			a.mCheck.SetTitle("Check for Updates…")
			a.mCheck.Enable()
		}()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	rel, err := latestRelease(ctx)
	if err != nil {
		if interactive {
			alert("BTCMenuBar", "Could not check for updates: "+err.Error())
		}
		return
	}
	if !isNewer(rel.Tag, version) {
		if interactive {
			alert("BTCMenuBar", fmt.Sprintf("You have the latest version (%s).", version))
		}
		return
	}
	a.mu.Lock()
	a.pending = rel
	a.mu.Unlock()
	a.mInstall.SetTitle("⬆ Install Update " + rel.Tag)
	a.mInstall.Show()
	if interactive {
		a.install(true)
	} else {
		notify("BTCMenuBar", "Version "+rel.Tag+" is available. Open the menu to install it.")
	}
}

func (a *app) install(ask bool) {
	a.mu.Lock()
	rel := a.pending
	if rel == nil || a.updating {
		a.mu.Unlock()
		return
	}
	a.updating = true
	a.mu.Unlock()
	defer func() {
		a.mu.Lock()
		a.updating = false
		a.mu.Unlock()
	}()

	if ask && !confirm("BTCMenuBar", fmt.Sprintf("Version %s is available (you have %s). Install it now?", rel.Tag, version), "Install") {
		return
	}
	a.mInstall.SetTitle("Installing " + rel.Tag + "…")
	a.mInstall.Disable()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	appPath, err := installUpdate(ctx, rel)
	if err != nil {
		a.mInstall.SetTitle("⬆ Install Update " + rel.Tag)
		a.mInstall.Enable()
		if confirm("BTCMenuBar", "The update could not be installed: "+err.Error()+"\n\nDownload it from GitHub instead?", "Open GitHub") {
			openURL(rel.PageURL)
		}
		return
	}
	if err := relaunch(appPath); err != nil {
		alert("BTCMenuBar", "Updated to "+rel.Tag+". Please open the app again.")
	}
	systray.Quit()
}

/* ---------- helpers ---------- */

func humanDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%d seconds", int(d.Seconds()))
	}
	if d == time.Minute {
		return "1 minute"
	}
	return fmt.Sprintf("%d minutes", int(d.Minutes()))
}

func shorten(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}
