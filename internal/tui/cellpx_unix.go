//go:build unix

package tui

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/x/term"
	"golang.org/x/sys/unix"
)

// cellPixels is the size in device pixels of one terminal cell, from the
// pixel size the terminal reports for its window; zero when it reports none.
func cellPixels() (w, h float64) {
	ws, err := unix.IoctlGetWinsize(int(os.Stdout.Fd()), unix.TIOCGWINSZ)
	if err != nil || ws.Col == 0 || ws.Row == 0 || ws.Xpixel == 0 || ws.Ypixel == 0 {
		return 0, 0
	}
	return float64(ws.Xpixel) / float64(ws.Col), float64(ws.Ypixel) / float64(ws.Row)
}

// ProbeCell asks the terminal for the exact size of a cell (CSI 16 t),
// before anything else reads the keyboard. After a resize the interface asks
// again through the program's own input.
func ProbeCell() (cw, ch int, ok bool) {
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(uintptr(fd)) {
		return 0, 0, false
	}
	state, err := term.MakeRaw(uintptr(fd))
	if err != nil {
		return 0, 0, false
	}
	defer func() { _ = term.Restore(uintptr(fd), state) }()
	if _, err := os.Stdout.WriteString("\x1b[16t\x1b[c"); err != nil {
		return 0, 0, false
	}
	var got []byte
	buf := make([]byte, 256)
	deadline := time.Now().Add(400 * time.Millisecond)
	for time.Now().Before(deadline) {
		fds := []unix.PollFd{{Fd: int32(fd), Events: unix.POLLIN}}
		n, err := unix.Poll(fds, int(time.Until(deadline)/time.Millisecond)+1)
		if err != nil || n == 0 {
			break
		}
		k, err := os.Stdin.Read(buf)
		if err != nil || k == 0 {
			break
		}
		got = append(got, buf[:k]...)
		// The reply to DA1 comes last: ESC [ ? … c.
		if i := strings.LastIndex(string(got), "\x1b[?"); i >= 0 && strings.HasSuffix(string(got), "c") {
			break
		}
	}
	s := string(got)
	i := strings.Index(s, "\x1b[6;")
	if i < 0 {
		return 0, 0, false
	}
	rest := s[i+4:]
	end := strings.IndexByte(rest, 't')
	if end < 0 {
		return 0, 0, false
	}
	parts := strings.Split(rest[:end], ";")
	if len(parts) != 2 {
		return 0, 0, false
	}
	ch, err1 := strconv.Atoi(parts[0])
	cw, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil || cw <= 0 || ch <= 0 {
		return 0, 0, false
	}
	return cw, ch, true
}
