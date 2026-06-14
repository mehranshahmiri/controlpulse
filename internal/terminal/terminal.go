package terminal

import (
	"encoding/json"
	"os"
	"os/exec"

	"github.com/creack/pty"
	"github.com/gofiber/contrib/websocket"
)

// Message defines the structure of data sent from the frontend
type Message struct {
	Type string `json:"type"` // "input" or "resize"
	Data string `json:"data,omitempty"`
	Cols int    `json:"cols,omitempty"`
	Rows int    `json:"rows,omitempty"`
}

// Handler manages the WebSocket connection and PTY session
func Handler(c *websocket.Conn) {
	// 1. Create the command (bash)
	cmd := exec.Command("/bin/bash")

	// Set environment variables for the shell
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")

	// 2. Start PTY
	ptmx, err := pty.Start(cmd)
	if err != nil {
		c.WriteMessage(websocket.TextMessage, []byte("Failed to start PTY: "+err.Error()))
		return
	}
	defer func() {
		_ = ptmx.Close()
		_ = cmd.Process.Kill()
	}()

	// 3. Handle PTY Output -> WebSocket (Read from shell, send to browser)
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := ptmx.Read(buf)
			if err != nil {
				return
			}
			if err := c.WriteMessage(websocket.TextMessage, buf[:n]); err != nil {
				return
			}
		}
	}()

	// 4. Handle WebSocket Input -> PTY (Read from browser, write to shell)
	for {
		_, msg, err := c.ReadMessage()
		if err != nil {
			break
		}

		var payload Message
		if err := json.Unmarshal(msg, &payload); err != nil {
			continue
		}

		switch payload.Type {
		case "input":
			ptmx.Write([]byte(payload.Data))
		case "resize":
			pty.Setsize(ptmx, &pty.Winsize{
				Rows: uint16(payload.Rows),
				Cols: uint16(payload.Cols),
			})
		}
	}
}
