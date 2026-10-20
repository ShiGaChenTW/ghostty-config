package tui

import (
	"context"
	"errors"
	"fmt"

	tea "charm.land/bubbletea/v2"

	"github.com/vitruves/ghostty-config/internal/update"
)

// A newer release is offered in a dialog when the tool starts: Install
// replaces this binary with it, Next time leaves things as they are until the
// next check.

// installRelease puts a release in place of this binary; a variable so tests
// never download one.
var installRelease = update.Install

// offerUpdate opens the update dialog, or holds it until the dialog already
// open is closed.
func (m *Model) offerUpdate(latest string) {
	m.updateLatest = latest
	if m.overlay == overlayNone {
		m.overlay = overlayUpdate
		return
	}
	m.updatePending = true
}

// showPendingUpdate opens the update dialog held back by another one.
func (m *Model) showPendingUpdate() {
	if m.updatePending && m.overlay == overlayNone {
		m.updatePending = false
		m.overlay = overlayUpdate
	}
}

// startUpdate downloads and installs the release on offer in the background.
func (m *Model) startUpdate() tea.Cmd {
	v := m.updateLatest
	m.overlay = overlayBusy
	m.busyText = "Installing ghostty-config " + v + "…"
	return tea.Batch(m.spin.Tick, func() tea.Msg {
		path, err := installRelease(context.Background(), v)
		return updateInstalledMsg{version: v, path: path, err: err}
	})
}

// finishUpdate says how the update went.
func (m *Model) finishUpdate(msg updateInstalledMsg) tea.Cmd {
	m.overlay = overlayNone
	if msg.err != nil {
		why := msg.err.Error()
		if errors.Is(msg.err, update.ErrUnsupported) {
			why = "there is no ready-made binary for this system"
		}
		m.openMessage(fmt.Sprintf("Could not update to %s: %s. Install it from a shell instead: %s", msg.version, why, update.InstallHint))
		return nil
	}
	m.updateNote = fmt.Sprintf("ghostty-config is now %s: run it again to use the new version.", msg.version)
	m.openMessage(fmt.Sprintf("Updated to %s in %s. The new version starts the next time you open ghostty-config.", msg.version, shortenPath(msg.path)))
	return nil
}
