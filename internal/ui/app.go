// Package ui implements the Win32-based user interface for knowswell.
//
// The package uses pure syscall / golang.org/x/sys/windows — no CGO,
// no external UI framework, no extra DLLs beyond the Windows system
// libraries. All windows use WDA_EXCLUDEFROMCAPTURE so they never
// appear in the user's screen captures.
package ui

import (
	"fmt"
	"sync"
	"sync/atomic"

	"knowswell/internal/capture"
	"knowswell/internal/config"
	"knowswell/internal/llm"
)

// App coordinates the toolbar, chat bar, snip overlay and response overlay.
type App struct {
	cfg       *config.Config
	llm       *llm.Client
	attMu     sync.Mutex
	attList   []llm.Attachment
	llmCancel func()

	streamMu  sync.Mutex
	streamBuf []byte

	// Window handles.
	Toolbar  Win
	ChatBar  Win
	Overlay  Win
	Snip     Win
	Settings Win

	// Toolbar state.
	tbX, tbY, tbW, tbH int
	tbExpanded         bool
	hoverBtn           int

	// Chat bar state.
	cbH        int
	editHandle Win
	attHover   int

	// Snip state.
	snipStartX, snipStartY int
	snipEndX, snipEndY     int
	snipDragging           bool

	// Settings window controls.
	hwAPIKey          Win
	hwModel           Win
	hwEndpoint        Win
	hwSysPrompt       Win
	hwPresets         Win
	hwOpacity         Win // trackbar
	settingsGhostMode bool
	settingsOpacity   int

	// Toolbar animation.
	tbAnimFrame int

	// Overlay RichEdit.
	hwRichEdit   Win
	overlayCopied bool
}

func (a *App) alphaValue() byte {
	o := a.cfg.Opacity
	if o < 10 {
		o = 10
	}
	if o > 100 {
		o = 100
	}
	return byte(o * 255 / 100)
}

func (a *App) applyOpacity() {
	alpha := a.alphaValue()
	for _, hw := range []HWND{a.Toolbar, a.ChatBar, a.Overlay, a.Settings} {
		if hw != 0 {
			pSetLayeredWindowAttributes.Call(uintptr(hw), 0, uintptr(alpha), lwaAlpha)
		}
	}
}

// NewApp constructs the application coordinator. Call Run() to start the UI.
func NewApp(cfg *config.Config) *App {
	c := llm.NewClient(cfg.APIEndpoint, cfg.APIKey, cfg.Model, cfg.MaxTokens)
	c.SystemPrompt = cfg.SystemPrompt
	return &App{cfg: cfg, llm: c}
}

func (a *App) displayAffinity() uintptr {
	if a.cfg.GhostMode {
		return wdaExcludeFromCapture
	}
	return wdaNone
}

// ---------- Attachment helpers ----------

func (a *App) AddAttachment(att llm.Attachment) {
	a.attMu.Lock()
	a.attList = append(a.attList, att)
	a.attMu.Unlock()
	a.RefreshChatBar()
}

func (a *App) RemoveAttachment(idx int) {
	a.attMu.Lock()
	defer a.attMu.Unlock()
	if idx < 0 || idx >= len(a.attList) {
		return
	}
	a.attList = append(a.attList[:idx], a.attList[idx+1:]...)
}

func (a *App) Attachments() []llm.Attachment {
	a.attMu.Lock()
	defer a.attMu.Unlock()
	cp := make([]llm.Attachment, len(a.attList))
	copy(cp, a.attList)
	return cp
}

func (a *App) ClearAttachments() {
	a.attMu.Lock()
	a.attList = nil
	a.attMu.Unlock()
	a.RefreshChatBar()
}

func (a *App) AddSnipFromCapture(x, y, w, h int) error {
	img, err := capture.CaptureRegion(x, y, w, h)
	if err != nil {
		return fmt.Errorf("capture region: %w", err)
	}
	a.AddAttachment(llm.Attachment{
		Image: img,
		Label: fmt.Sprintf("Snip %dx%d", w, h),
	})
	return nil
}

// ---------- Streaming response buffer ----------

func (a *App) AppendStream(chunk string) {
	a.streamMu.Lock()
	a.streamBuf = append(a.streamBuf, chunk...)
	a.streamMu.Unlock()
}

func (a *App) StreamText() string {
	a.streamMu.Lock()
	defer a.streamMu.Unlock()
	return string(a.streamBuf)
}

func (a *App) ResetStream() {
	a.streamMu.Lock()
	a.streamBuf = nil
	a.streamMu.Unlock()
}

// ---------- GC-safe message string handles ----------
// PostMessageW is asynchronous; passing a raw *string pointer as lParam
// is unsafe because the GC may collect the allocation before the message
// is dispatched. We keep a global map that holds live references.

var (
	msgMu   sync.Mutex
	msgMap  = map[uint64]*string{}
	msgSeq  uint64
)

// putMsgString stores s under a unique handle and returns it. The caller
// passes the handle (cast to uintptr) as a PostMessageW lParam.
func putMsgString(s string) uint64 {
	p := new(string)
	*p = s
	id := atomic.AddUint64(&msgSeq, 1)
	msgMu.Lock()
	msgMap[id] = p
	msgMu.Unlock()
	return id
}

// getMsgString retrieves and removes the string stored under handle.
// Returns "" if the handle is unknown (already consumed or never stored).
func getMsgString(handle uint64) string {
	msgMu.Lock()
	p := msgMap[handle]
	delete(msgMap, handle)
	msgMu.Unlock()
	if p == nil {
		return ""
	}
	return *p
}
