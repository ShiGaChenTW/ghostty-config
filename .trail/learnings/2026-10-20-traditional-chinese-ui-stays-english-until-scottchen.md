---
title: Traditional Chinese UI stays English until translated
author: Scottchen
date: 2026-10-20
branch: main
touches:
  - cmd/ghostty-config/main.go
  - internal/ghostty/state.go
  - internal/pix/pix.go
  - internal/tui/actions.go
  - internal/tui/app.go
  - internal/tui/commands.go
  - internal/tui/dialogs.go
  - internal/tui/extensions.go
  - internal/tui/fonts.go
  - internal/tui/gallery.go
---

## Decided
The editor language is its own state (en or zh-Hant), not LANG. English drawing stays byte-identical because t() is the identity. Translate at draw time. Command names, config keys, written values, theme names and keycaps stay English.

The image UI embeds a SIL OFL subset named GhosttyConfigSansTC (Noto Sans TC, CFF). pix.runs tries it after the monospace face and before symbol substitutes. Do not register that face in kimg.faceInfo.

## Tried and ruled out
Registering the CJK face in faceInfo. Covers treats an unknown face as covering every rune, which is how specimen fonts are left alone; registering CJK would make a missing glyph look present.

Converting the subset to glyf. golang.org/x/image/font/sfnt draws this CFF font.

Title-casing an already translated key hint with ToUpper on the first byte. That splits a Han character. keyPairs translates first; the pixel footer title-cases only while the language is English.

Calling t() on rail counts. A count can be the config value "default", and that word is also the key-hint catalog entry.

## Watch out
Subset the font after the catalog is stable, and strip Noto / Source Han from the name table. A system fallback may be used only when the font name says Traditional; never a Simplified face. Fixed pixel columns in the welcome screen can overlap in Chinese; that is accepted.
