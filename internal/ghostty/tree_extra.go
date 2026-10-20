package ghostty

import "strings"

// Repeatable keys. Most keys are scalars, where the last assignment wins and
// rewriting that one line is enough. A few, custom-shader among them, stack:
// every line contributes. Changing one of those to a single value means
// switching every other line off, and putting it back means restoring each
// line exactly as it was.

// SetOnly makes key = value the one assignment in effect: the last line that
// assigns it is rewritten in place, every earlier one is commented out with
// reason, and with none the key is appended to the primary file. It reports
// the files it changed.
func (t *Tree) SetOnly(key, value, reason string) []*Document {
	t.mu.Lock()
	defer t.mu.Unlock()
	var changed []*Document
	t.record(func(t *Tree) { changed = t.setOnly(key, value, reason) })
	return changed
}

func (t *Tree) setOnly(key, value, reason string) []*Document {
	all := t.All(key)
	if len(all) == 0 {
		t.Primary.Append(key, value)
		t.index()
		return []*Document{t.Primary}
	}
	var changed []*Document
	for _, e := range all[:len(all)-1] {
		e.Doc.CommentOut(e.Line, reason)
		changed = appendDoc(changed, e.Doc)
	}
	if last := all[len(all)-1]; last.Value != value {
		last.Doc.SetLine(last.Line, key, value)
		changed = appendDoc(changed, last.Doc)
	}
	t.index()
	return changed
}

// KeyLines is every line that assigned a key at one moment, kept verbatim so
// that RestoreKey can put them back exactly.
type KeyLines struct {
	key   string
	lines []keyLine
}

type keyLine struct {
	doc  *Document
	line int
	was  Line
}

// SnapshotKey records every line that assigns key.
func (t *Tree) SnapshotKey(key string) KeyLines {
	t.mu.Lock()
	defer t.mu.Unlock()
	s := KeyLines{key: key}
	for _, e := range t.All(key) {
		s.lines = append(s.lines, keyLine{doc: e.Doc, line: e.Line, was: e.Doc.Lines[e.Line]})
	}
	return s
}

// RestoreKey puts back the lines a snapshot recorded, and comments out with
// reason any line assigning the key that the snapshot did not hold, such as
// one appended since. Edits in between must have kept line numbers, which
// SetOnly, Set and Unset do for the lines they touch.
//
// A replay after the files changed on disk cannot trust line numbers, so it
// puts back the values the snapshot had in effect, in their order, instead.
func (t *Tree) RestoreKey(s KeyLines, reason string) []*Document {
	t.mu.Lock()
	defer t.mu.Unlock()
	var values []string
	for _, l := range s.lines {
		if l.was.Kind == LineEntry && l.was.Key == s.key {
			values = append(values, l.was.Value)
		}
	}
	t.ops = append(t.ops, func(t *Tree) { t.assignOnly(s.key, values, reason) })
	if !t.snapshotHolds(s) {
		return t.assignOnly(s.key, values, reason)
	}
	return t.restoreKey(s, reason)
}

// snapshotHolds reports whether every line a snapshot recorded still holds
// the key, live or commented out: a file read again since may have moved
// them.
func (t *Tree) snapshotHolds(s KeyLines) bool {
	for _, l := range s.lines {
		if l.line >= len(l.doc.Lines) {
			return false
		}
		now := l.doc.Lines[l.line]
		if now.Kind == LineEntry && now.Key == s.key {
			continue
		}
		if now.Kind == LineComment && strings.Contains(now.Raw, s.key) {
			continue
		}
		return false
	}
	return true
}

// assignOnly leaves exactly values in effect for key, in order: kept as they
// are when they already are, otherwise every line is commented out and the
// values are appended to the primary file.
func (t *Tree) assignOnly(key string, values []string, reason string) []*Document {
	all := t.All(key)
	same := len(all) == len(values)
	for i := 0; same && i < len(all); i++ {
		same = all[i].Value == values[i]
	}
	if same {
		return nil
	}
	var changed []*Document
	for _, e := range all {
		e.Doc.CommentOut(e.Line, reason)
		changed = appendDoc(changed, e.Doc)
	}
	for _, v := range values {
		t.Primary.Append(key, v)
		changed = appendDoc(changed, t.Primary)
	}
	t.index()
	return changed
}

func (t *Tree) restoreKey(s KeyLines, reason string) []*Document {
	kept := make(map[*Document]map[int]bool)
	var changed []*Document
	for _, l := range s.lines {
		if l.line >= len(l.doc.Lines) {
			continue
		}
		if kept[l.doc] == nil {
			kept[l.doc] = make(map[int]bool)
		}
		kept[l.doc][l.line] = true
		if l.doc.Lines[l.line] != l.was {
			l.doc.Lines[l.line] = l.was
			l.doc.Dirty = true
			changed = appendDoc(changed, l.doc)
		}
	}
	t.index()
	for _, e := range t.All(s.key) {
		if !kept[e.Doc][e.Line] {
			e.Doc.CommentOut(e.Line, reason)
			changed = appendDoc(changed, e.Doc)
		}
	}
	t.index()
	return changed
}
