// rtf.go — converts markdown text to RTF and sets it into a RichEdit control.
package ui

import (
	"fmt"
	"runtime"
	"strings"
	"unicode/utf8"
	"unsafe"
)

// settextex is the SETTEXTEX struct used with EM_SETTEXTEX.
// The struct is 8 bytes: Flags (DWORD) + Codepage (UINT).
type settextex struct {
	Flags    uint32
	Codepage uint32
}

// streamRTFToRichEdit sets the RichEdit content to markdown text.
// Uses EM_SETTEXTEX for RTF (formatted), with SetWindowTextW as a safe
// fallback so a bad RTF parse never crashes the process.
func streamRTFToRichEdit(hwnd HWND, markdown string) {
	defer func() { recover() }() // catch any fault in this path

	rtf := markdownToRTF(markdown)

	// CP_ACP (Codepage=0): RichEdit auto-detects {\rtf1 and parses as RTF.
	st := &settextex{Flags: 0, Codepage: 0}
	data := append([]byte(rtf), 0) // null-terminated ANSI RTF
	ret, _, _ := pSendMessageW.Call(uintptr(hwnd), emSetTextEx,
		uintptr(unsafe.Pointer(st)),
		uintptr(unsafe.Pointer(&data[0])))
	runtime.KeepAlive(st)
	runtime.KeepAlive(data)

	if ret == 0 {
		// RTF path rejected — fall back to plain Unicode text.
		p := utf16Ptr(markdown)
		pSendMessageW.Call(uintptr(hwnd), wmSetText, 0,
			uintptr(unsafe.Pointer(p)))
		runtime.KeepAlive(p)
	}

	// Move caret to end of document then scroll it into view.
	pSendMessageW.Call(uintptr(hwnd), emSetSel, ^uintptr(0), ^uintptr(0))
	pSendMessageW.Call(uintptr(hwnd), emScrollCaret, 0, 0)
}

const emScrollCaret = 0x00B7

// ---------- Markdown → RTF ----------

const rtfHeader = `{\rtf1\ansi\ansicpg65001\deff0\nouicompat` +
	`{\fonttbl{\f0\fmodern\fcharset0 Courier New;}{\f1\fmodern\fcharset0 Courier New;}}` +
	`{\colortbl ;` +
	`\red160\green255\blue140;` + // cf1 = bright phosphor body text
	`\red0\green255\blue80;` + // cf2 = max-bright green accent / headings
	`\red220\green185\blue0;` + // cf3 = amber inline code
	`\red14\green30\blue14;` + // cf4 = dark surface code-block bg
	`\red80\green160\blue70;` + // cf5 = dim phosphor
	`}` +
	`\viewkind4\uc1\pard\sl300\slmult1\f0\fs26\cf1 `

func markdownToRTF(md string) string {
	var b strings.Builder
	b.WriteString(rtfHeader)

	lines := strings.Split(md, "\n")
	inCode := false

	for _, line := range lines {
		// Code fence toggle.
		if strings.HasPrefix(line, "```") {
			inCode = !inCode
			if inCode {
				// Start code block: monospace, dark surface background.
				b.WriteString(`\pard\sl0\f1\fs22\cf3\cb4 `)
			} else {
				// End code block: back to normal prose.
				b.WriteString(`\cb0\pard\sl300\slmult1\f0\fs26\cf1 `)
			}
			b.WriteString(`\par `)
			continue
		}

		if inCode {
			b.WriteString(rtfEscape(line))
			b.WriteString(`\line `)
			continue
		}

		trimmed := strings.TrimSpace(line)

		switch {
		case strings.HasPrefix(line, "### "):
			b.WriteString(`\pard\sl300\slmult1\b\fs24\cf2 `)
			b.WriteString(rtfInline(line[4:]))
			b.WriteString(`\b0\cf1\par `)

		case strings.HasPrefix(line, "## "):
			b.WriteString(`\pard\sl300\slmult1\b\fs28\cf2 `)
			b.WriteString(rtfInline(line[3:]))
			b.WriteString(`\b0\fs26\cf1\par `)

		case strings.HasPrefix(line, "# "):
			b.WriteString(`\pard\sl300\slmult1\b\fs32\cf2 `)
			b.WriteString(rtfInline(line[2:]))
			b.WriteString(`\b0\fs26\cf1\par `)

		case strings.HasPrefix(line, "---") && trimmed == strings.Repeat("-", len(trimmed)):
			// Horizontal rule — draw as a thin line of underscores.
			b.WriteString(`\pard\sl0\cf5 `)
			b.WriteString(strings.Repeat(`\_`, 55))
			b.WriteString(`\cf1\pard\sl300\slmult1\par `)

		case strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* "):
			b.WriteString(`\pard\sl300\slmult1\li240\fi-240 \bullet  `)
			b.WriteString(rtfInline(line[2:]))
			b.WriteString(`\par `)

		case trimmed == "":
			b.WriteString(`\par `)

		default:
			b.WriteString(`\pard\sl300\slmult1 `)
			b.WriteString(rtfInline(line))
			b.WriteString(`\par `)
		}
	}

	b.WriteString(`}`)
	return b.String()
}

// rtfInline handles inline markdown within a single line:
// **bold**, *italic*, `code`, and plain text.
func rtfInline(s string) string {
	var b strings.Builder
	i := 0
	for i < len(s) {
		// Bold: **text**
		if i+1 < len(s) && s[i] == '*' && s[i+1] == '*' {
			end := strings.Index(s[i+2:], "**")
			if end >= 0 {
				b.WriteString(`\b `)
				b.WriteString(rtfEscape(s[i+2 : i+2+end]))
				b.WriteString(`\b0 `)
				i = i + 2 + end + 2
				continue
			}
		}
		// Bold: __text__
		if i+1 < len(s) && s[i] == '_' && s[i+1] == '_' {
			end := strings.Index(s[i+2:], "__")
			if end >= 0 {
				b.WriteString(`\b `)
				b.WriteString(rtfEscape(s[i+2 : i+2+end]))
				b.WriteString(`\b0 `)
				i = i + 2 + end + 2
				continue
			}
		}
		// Italic: *text*
		if s[i] == '*' && (i == 0 || s[i-1] != '*') {
			end := strings.IndexByte(s[i+1:], '*')
			if end >= 0 && end > 0 {
				b.WriteString(`\i `)
				b.WriteString(rtfEscape(s[i+1 : i+1+end]))
				b.WriteString(`\i0 `)
				i = i + 1 + end + 1
				continue
			}
		}
		// Italic: _text_
		if s[i] == '_' && (i == 0 || s[i-1] != '_') {
			end := strings.IndexByte(s[i+1:], '_')
			if end >= 0 && end > 0 {
				b.WriteString(`\i `)
				b.WriteString(rtfEscape(s[i+1 : i+1+end]))
				b.WriteString(`\i0 `)
				i = i + 1 + end + 1
				continue
			}
		}
		// Inline code: `text`
		if s[i] == '`' {
			end := strings.IndexByte(s[i+1:], '`')
			if end >= 0 {
				b.WriteString(`\f1\fs22\cf3 `)
				b.WriteString(rtfEscape(s[i+1 : i+1+end]))
				b.WriteString(`\f0\fs26\cf1 `)
				i = i + 1 + end + 1
				continue
			}
		}
		// Plain character.
		r, size := utf8.DecodeRuneInString(s[i:])
		b.WriteString(rtfChar(r))
		i += size
	}
	return b.String()
}

// rtfEscape escapes an entire string (no inline markdown processing).
func rtfEscape(s string) string {
	var b strings.Builder
	for _, r := range s {
		b.WriteString(rtfChar(r))
	}
	return b.String()
}

// rtfChar returns the RTF encoding of a single rune.
func rtfChar(r rune) string {
	switch r {
	case '\\':
		return `\\`
	case '{':
		return `\{`
	case '}':
		return `\}`
	case '\t':
		return `\tab `
	}
	if r < 128 {
		return string(r)
	}
	// Unicode escape: \uN? where N is the decimal code point.
	return fmt.Sprintf(`\u%d?`, r)
}
