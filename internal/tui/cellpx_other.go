//go:build !unix

package tui

func cellPixels() (w, h float64) { return 0, 0 }

func termPixels() (cols, rows, xpix, ypix int) { return 0, 0, 0, 0 }

// ProbeCell is unsupported here: the cell size is unknown.
func ProbeCell() (cw, ch, padX, padY int, ok bool) { return 0, 0, 0, 0, false }
