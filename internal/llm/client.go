// Package llm implements a streaming OpenAI-compatible chat client.
//
// The client speaks the chat-completion SSE protocol used by OpenAI,
// Ollama, LM Studio, Groq and any other backend that exposes the same
// /v1/chat/completions endpoint. The streamed delta tokens are delivered
// to the caller via a callback, which the UI uses to update the on-screen
// overlay in real time.
package llm

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"io"
	"net/http"
	"os"
	"time"
)

// Client is an OpenAI-compatible chat-completion client.
type Client struct {
	Endpoint     string
	APIKey       string
	Model        string
	MaxTokens    int
	SystemPrompt string
	HTTPClient   *http.Client
}

// Attachment is a single piece of media to send to the model alongside
// the text query. Exactly one of Image or FilePath should be set.
type Attachment struct {
	Image    image.Image
	FilePath string
	Label    string
}

// NewClient constructs a Client with sensible timeouts.
func NewClient(endpoint, apiKey, model string, maxTokens int) *Client {
	return &Client{
		Endpoint:   endpoint,
		APIKey:     apiKey,
		Model:      model,
		MaxTokens:  maxTokens,
		HTTPClient: &http.Client{Timeout: 120 * time.Second},
	}
}

// ---------- Wire types (OpenAI chat completion) ----------

type imageContent struct {
	Type     string          `json:"type"`
	Text     string          `json:"text,omitempty"`
	ImageURL *imageURLDetail `json:"image_url,omitempty"`
}

type imageURLDetail struct {
	URL    string `json:"url"`
	Detail string `json:"detail"`
}

type chatMessage struct {
	Role    string         `json:"role"`
	Content []imageContent `json:"content"`
}

type chatRequest struct {
	Model     string        `json:"model"`
	Messages  []chatMessage `json:"messages"`
	MaxTokens int           `json:"max_tokens"`
	Stream    bool          `json:"stream"`
}

type streamChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
}

type apiErrorResp struct {
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}

// ---------- Encoding helpers ----------

// encodeImageToBase64 encodes an image.Image to a base64 PNG data URL.
func encodeImageToBase64(img image.Image) (string, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

// encodeFileToBase64 reads a file and returns its raw base64 encoding.
func encodeFileToBase64(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

// ---------- Public API ----------

// AskStream sends the query + attachments to the chat-completion endpoint
// and invokes onChunk for each delta token received. onChunk is called
// from the calling goroutine, so the UI is responsible for marshalling
// back to its own thread (e.g. via PostMessage).
func (c *Client) AskStream(
	ctx context.Context,
	query string,
	attachments []Attachment,
	onChunk func(string),
) error {
	contents := make([]imageContent, 0, len(attachments)+1)

	for _, att := range attachments {
		var b64 string
		var err error
		switch {
		case att.Image != nil:
			b64, err = encodeImageToBase64(att.Image)
		case att.FilePath != "":
			b64, err = encodeFileToBase64(att.FilePath)
		default:
			continue
		}
		if err != nil || b64 == "" {
			continue
		}
		contents = append(contents, imageContent{
			Type: "image_url",
			ImageURL: &imageURLDetail{
				URL:    "data:image/png;base64," + b64,
				Detail: "high",
			},
		})
	}

	contents = append(contents, imageContent{Type: "text", Text: query})

	messages := make([]chatMessage, 0, 2)
	if c.SystemPrompt != "" {
		messages = append(messages, chatMessage{
			Role:    "system",
			Content: []imageContent{{Type: "text", Text: c.SystemPrompt}},
		})
	}
	messages = append(messages, chatMessage{Role: "user", Content: contents})

	body, err := json.Marshal(chatRequest{
		Model:     c.Model,
		Messages:  messages,
		MaxTokens: c.MaxTokens,
		Stream:    true,
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.Endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}
	req.Header.Set("Accept", "text/event-stream")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		var ae apiErrorResp
		if json.Unmarshal(raw, &ae) == nil && ae.Error.Message != "" {
			return fmt.Errorf("API error: %s", ae.Error.Message)
		}
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(raw))
	}

	return parseSSEStream(ctx, resp.Body, onChunk)
}

// parseSSEStream reads the response body line-by-line, dispatching
// "data: <json>" lines to the chunk callback until "[DONE]" or EOF.
func parseSSEStream(ctx context.Context, r io.Reader, onChunk func(string)) error {
	reader := newSSEReader(r)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line, ok, err := reader.NextLine()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		if !ok {
			continue
		}
		if !bytes.HasPrefix(line, []byte("data: ")) {
			continue
		}
		payload := bytes.TrimPrefix(line, []byte("data: "))
		if string(payload) == "[DONE]" {
			return nil
		}
		var sc streamChunk
		if err := json.Unmarshal(payload, &sc); err != nil {
			// Some backends emit comment lines; ignore malformed chunks.
			continue
		}
		if len(sc.Choices) > 0 {
			t := sc.Choices[0].Delta.Content
			if t != "" {
				onChunk(t)
			}
		}
	}
}

// ---------- Minimal SSE line reader ----------

// sseReader splits an io.Reader on '\n' while preserving partial lines
// across Read calls. It returns (line, true, nil) for each complete line,
// (nil, false, nil) on successful reads with no full line yet, and
// (nil, false, err) on terminal errors / EOF.
type sseReader struct {
	r         io.Reader
	buf       []byte
	leftover  []byte
	readChunk int
}

func newSSEReader(r io.Reader) *sseReader {
	return &sseReader{r: r, readChunk: 4096}
}

func (s *sseReader) NextLine() ([]byte, bool, error) {
	for {
		if idx := bytes.IndexByte(s.leftover, '\n'); idx >= 0 {
			line := s.leftover[:idx]
			s.leftover = s.leftover[idx+1:]
			return bytes.TrimRight(line, "\r"), true, nil
		}
		if s.buf == nil {
			s.buf = make([]byte, s.readChunk)
		}
		n, err := s.r.Read(s.buf)
		if n > 0 {
			s.leftover = append(s.leftover, s.buf[:n]...)
		}
		if err != nil {
			if err == io.EOF {
				if n > 0 {
					// Data and EOF arrived together — loop back to scan
					// leftover for \n before treating it as a final line.
					continue
				}
				if len(s.leftover) > 0 {
					line := s.leftover
					s.leftover = nil
					return bytes.TrimRight(line, "\r"), true, nil
				}
				return nil, false, io.EOF
			}
			return nil, false, err
		}
	}
}
