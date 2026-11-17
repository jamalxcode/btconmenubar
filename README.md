# BTCMenuBar

The Bitcoin price in your Mac menu bar, written in Go.

```
₿ $91,234 ▲ 1.23%
```

## Install

```sh
curl -fsSL https://raw.githubusercontent.com/jamalxcode/btconmenubar/main/install.sh | sh
```

Or download `BTCMenuBar.zip` from [Releases](https://github.com/jamalxcode/btconmenubar/releases/latest), unzip it and move `BTCMenuBar.app` to Applications. The app has no Apple Developer ID signature, so the first time either right-click it and choose **Open**, or run `xattr -dr com.apple.quarantine /Applications/BTCMenuBar.app`. The install script does this for you.

Works on macOS 11 or later, on Apple Silicon and Intel.

## Where the price comes from

1. **heat.sala.company** (`https://heat.sala.company/pulse.json`): the BTC price and 24h change that heat stores from CoinGecko every 10 minutes. Used for USD.
2. **CoinGecko** (`api.coingecko.com/api/v3/simple/price`), used directly when heat can't be reached, when its data is more than 30 minutes old, or when you pick a currency other than USD.

The menu shows which source the current number came from and when it was made.

## Menu

- Price, 24h change, time of the number, source
- **Refresh Now**
- Open heat.sala.company / Open on CoinGecko
- **Currency**: USD, EUR, GBP, JPY, AUD, CAD, CHF, AED, SAR, KWD
- **Refresh Every**: 30 s, 1 min (default), 5 min, 15 min
- **Text Size**: Normal (macOS default), Large (16 pt, default), Extra Large (18 pt)
- Show 24h Change in Menu Bar
- Launch at Login (adds a LaunchAgent in `~/Library/LaunchAgents`)
- **Check for Updates…**: looks up the newest GitHub release; if it's newer, asks, downloads `BTCMenuBar.zip`, swaps the app in place and opens it again. The app also checks by itself every 12 hours and shows **⬆ Install Update vX.Y.Z** in the menu when one is out.

Settings are saved in `~/Library/Application Support/BTCMenuBar/config.json`.

## Releasing a new version

Push a tag, or run the **Release** workflow from the Actions tab with a version:

```sh
git tag v1.0.1 && git push origin v1.0.1
```

The workflow builds a universal `.app` on a macOS runner and publishes it as a GitHub release. Everyone's app then offers the update.

## Building

Needs macOS with Xcode command line tools and Go 1.24+.

```sh
go test ./...
scripts/build-app.sh v0.0.0-local      # -> dist/BTCMenuBar.app and dist/BTCMenuBar.zip
open dist/BTCMenuBar.app
```

## Files

| File | What it does |
|---|---|
| `main.go` | Menu bar item, menu, refresh loop, update flow ([fyne.io/systray](https://github.com/fyne-io/systray)) |
| `price.go` | Reads heat.sala.company's pulse.json, falls back to CoinGecko |
| `update.go` | GitHub release check, download and in-place install, macOS dialogs |
| `format.go` | Price and change text |
| `config.go` | Saved settings, Launch at Login |
| `packaging/Info.plist` | App bundle info (`LSUIElement`: no Dock icon) |
| `scripts/build-app.sh` | Builds the universal `.app` and zip |
| `install.sh` | One-line installer |
