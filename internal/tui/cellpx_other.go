//go:build !unix

package tui

func cellPixels() (w, h float64) { return 0, 0 }

// ProbeCell is unsupported here: the cell size is unknown.
func ProbeCell() (cw, ch int, ok bool) { return 0, 0, false }
