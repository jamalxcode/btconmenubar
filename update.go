package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	repo        = "jamalxcode/btconmenubar"
	releasesAPI = "https://api.github.com/repos/" + repo + "/releases/latest"
	assetName   = "BTCMenuBar.zip"
)

// Release is the latest published GitHub release.
type Release struct {
	Tag      string
	PageURL  string
	AssetURL string
}

func latestRelease(ctx context.Context) (*Release, error) {
	var r struct {
		TagName string `json:"tag_name"`
		HTMLURL string `json:"html_url"`
		Assets  []struct {
			Name string `json:"name"`
			URL  string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if err := getJSON(ctx, releasesAPI, &r); err != nil {
		return nil, err
	}
	rel := &Release{Tag: r.TagName, PageURL: r.HTMLURL}
	for _, a := range r.Assets {
		if a.Name == assetName {
			rel.AssetURL = a.URL
		}
	}
	if rel.Tag == "" {
		return nil, errors.New("no release found")
	}
	return rel, nil
}

// isNewer reports whether version a is newer than b ("v1.2.3" form; "dev" counts as 0.0.0).
func isNewer(a, b string) bool {
	pa, pb := parseVersion(a), parseVersion(b)
	for i := range pa {
		if pa[i] != pb[i] {
			return pa[i] > pb[i]
		}
	}
	return false
}

func parseVersion(v string) [3]int {
	var out [3]int
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	if i := strings.IndexAny(v, "-+"); i >= 0 {
		v = v[:i]
	}
	for i, part := range strings.SplitN(v, ".", 3) {
		n, err := strconv.Atoi(part)
		if err != nil {
			return [3]int{}
		}
		out[i] = n
	}
	return out
}

// bundlePath is the .app the running binary lives in (…/BTCMenuBar.app/Contents/MacOS/btconmenubar).
func bundlePath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	if real, err := filepath.EvalSymlinks(exe); err == nil {
		exe = real
	}
	app := filepath.Dir(filepath.Dir(filepath.Dir(exe)))
	if !strings.HasSuffix(app, ".app") {
		return "", errors.New("not running from an .app bundle")
	}
	return app, nil
}

// installUpdate downloads the release zip, swaps it in for the running .app and returns the
// app path to relaunch. The old app is kept until the new one is in place.
func installUpdate(ctx context.Context, rel *Release) (string, error) {
	if rel.AssetURL == "" {
		return "", fmt.Errorf("release %s has no %s", rel.Tag, assetName)
	}
	app, err := bundlePath()
	if err != nil {
		return "", err
	}
	tmp, err := os.MkdirTemp("", "btconmenubar-update-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmp)

	zipPath := filepath.Join(tmp, assetName)
	if err := download(ctx, rel.AssetURL, zipPath); err != nil {
		return "", fmt.Errorf("download: %w", err)
	}
	unzipped := filepath.Join(tmp, "new")
	if out, err := exec.Command("/usr/bin/ditto", "-x", "-k", zipPath, unzipped).CombinedOutput(); err != nil {
		return "", fmt.Errorf("unzip: %v %s", err, out)
	}
	matches, _ := filepath.Glob(filepath.Join(unzipped, "*.app"))
	if len(matches) == 0 {
		return "", errors.New("no .app in the download")
	}
	newApp := matches[0]
	_ = exec.Command("/usr/bin/xattr", "-dr", "com.apple.quarantine", newApp).Run()

	backup := app + ".old"
	_ = os.RemoveAll(backup)
	if err := os.Rename(app, backup); err != nil {
		return "", fmt.Errorf("cannot replace %s: %w", app, err)
	}
	if out, err := exec.Command("/usr/bin/ditto", newApp, app).CombinedOutput(); err != nil {
		_ = os.RemoveAll(app)
		_ = os.Rename(backup, app)
		return "", fmt.Errorf("copy: %v %s", err, out)
	}
	_ = os.RemoveAll(backup)
	return app, nil
}

func download(ctx context.Context, url, dest string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "btconmenubar/"+version)
	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// relaunch opens the app again a moment after this process quits.
func relaunch(app string) error {
	return exec.Command("/bin/sh", "-c", `sleep 1; /usr/bin/open "$0"`, app).Start()
}

/* ---------- small macOS dialogs via AppleScript ---------- */

func asQuote(s string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(s) + `"`
}

// confirm shows a dialog and reports whether the user picked okButton.
func confirm(title, msg, okButton string) bool {
	script := fmt.Sprintf(`display dialog %s with title %s buttons {"Later", %s} default button %s`,
		asQuote(msg), asQuote(title), asQuote(okButton), asQuote(okButton))
	out, err := exec.Command("/usr/bin/osascript", "-e", script).Output()
	return err == nil && strings.Contains(string(out), "button returned:"+okButton)
}

func alert(title, msg string) {
	script := fmt.Sprintf(`display dialog %s with title %s buttons {"OK"} default button "OK"`, asQuote(msg), asQuote(title))
	_ = exec.Command("/usr/bin/osascript", "-e", script).Run()
}

func notify(title, msg string) {
	script := fmt.Sprintf(`display notification %s with title %s`, asQuote(msg), asQuote(title))
	_ = exec.Command("/usr/bin/osascript", "-e", script).Run()
}

func openURL(url string) { _ = exec.Command("/usr/bin/open", url).Start() }
