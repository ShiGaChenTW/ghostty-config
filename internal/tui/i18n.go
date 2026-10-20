package tui

import (
	"fmt"
	"strings"

	"github.com/vitruves/ghostty-config/internal/collection"
)

// Languages the interface can show. English is the default and the source
// language: a missing translation is the English string, never another
// language. zh-Hans and zh-CN are not Traditional and do not select it.
const (
	langEN = "en"
	langZh = "zh-Hant"
)

// NormalizeLang maps a -lang value or a saved preference onto en or zh-Hant.
// An empty result means the value is not a language this tool speaks.
func NormalizeLang(s string) string {
	folded := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(s), "_", "-"))
	switch folded {
	case "en", "english":
		return langEN
	case "zh-hant", "zh-tw", "zh-hk", "zh-mo":
		return langZh
	default:
		return ""
	}
}

// lang is the language of this session: -lang wins, then the saved choice.
func (m *Model) lang() string {
	if m == nil {
		return langEN
	}
	if m.langOverride != "" {
		return m.langOverride
	}
	if m.state != nil {
		if n := NormalizeLang(m.state.Language); n != "" {
			return n
		}
	}
	return langEN
}

func (m *Model) zh() bool { return m.lang() == langZh }

// t translates an exact English source string. English, and any string that
// is not a catalog key, comes back unchanged.
func (m *Model) t(s string) string {
	if s == "" || m == nil || !m.zh() {
		return s
	}
	if got, ok := zhHant[s]; ok {
		return got
	}
	return s
}

// tf translates a format, then fills it. The arguments stay as they are:
// theme names, paths, counts and errors are not catalog keys.
func (m *Model) tf(format string, args ...any) string {
	return fmt.Sprintf(m.t(format), args...)
}

// sentence is s plus a period, or the catalog's complete sentence when s
// itself translates. Callers that lowercase and append "." keep that English
// text whenever t returns s.
func (m *Model) sentence(s string) string {
	if got := m.t(s); got != s {
		return got
	}
	return s + "."
}

// commandDesc is a command's description as shown without a forced period.
// The collection's description is a format until here, so the count is filled
// in the language of this session.
func (m *Model) commandDesc(c *command) string {
	if c == nil {
		return ""
	}
	if c.name == "collection" {
		n := len(collection.Collection)
		if m.t(c.desc) != c.desc {
			return m.tf(c.desc, n)
		}
		return fmt.Sprintf(c.desc, n)
	}
	return m.t(c.desc)
}

// commandSentence is commandDesc with the period the text panels append.
func (m *Model) commandSentence(c *command) string {
	if c == nil {
		return ""
	}
	d := m.commandDesc(c)
	if c.name == "collection" {
		if m.t(c.desc) != c.desc {
			return d
		}
		return d + "."
	}
	if m.t(c.desc) != c.desc {
		return d
	}
	return c.desc + "."
}

// commandTitle is the menu name of a command. English stays the capitalised
// command word. Traditional uses the setting's label when the command is one,
// so "Background opacity" is the key rather than "Opacity".
func (m *Model) commandTitle(c *command) string {
	if c == nil {
		return ""
	}
	if !m.zh() {
		return title(c.name)
	}
	if c.set != nil {
		return m.t(c.set.label)
	}
	return m.t(title(c.name))
}

// keepsEnglish reports labels that are data, not copy: theme and font names,
// palette slots, config values, and preset or shader identifiers.
func (m *Model) keepsEnglish(o option) bool {
	// A command row may carry a theme swatch. Its label is still the command,
	// not the theme's name. Theme, font and slot rows have no command.
	if o.cmd != nil {
		return false
	}
	if o.theme != nil || o.family != nil || o.slot != "" {
		return true
	}
	if m.cmd != nil && m.cmd.set != nil && o.cmd == nil {
		return true
	}
	if m.cmd != nil && o.cmd == nil && !o.header {
		switch m.cmd.name {
		case "preset", "shader", "pack", "style", "size":
			return true
		}
	}
	return false
}

// shownLabel is the label a row draws.
func (m *Model) shownLabel(o option) string {
	if m.keepsEnglish(o) {
		return o.label
	}
	if m.zh() && o.cmd != nil && o.cmd.set != nil {
		return m.t(o.cmd.set.label)
	}
	return m.t(o.label)
}

// shownDetail is the note a row draws. Theme readouts and config values stay
// as stored; "current" is the one value word that is copy.
func (m *Model) shownDetail(o option) string {
	if o.theme != nil || o.family != nil || o.slot != "" {
		return o.detail
	}
	if o.detail == "current" {
		return m.t("current")
	}
	if m.cmd != nil && m.cmd.set != nil && o.cmd == nil {
		return o.detail
	}
	return m.t(o.detail)
}
