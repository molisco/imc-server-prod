package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/creack/pty"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for dashboard Web Terminal
	},
}

type TerminalMessage struct {
	Type string `json:"type"` // "input", "resize"
	Data string `json:"data"`
	Cols uint16 `json:"cols"`
	Rows uint16 `json:"rows"`
}

// HandleWebTerminalWebSocket provides a full interactive PTY terminal over WebSocket
func HandleWebTerminalWebSocket(w http.ResponseWriter, r *http.Request) {
	// Authenticate session
	cookie, err := r.Cookie("imc_session")
	if err != nil || cookie.Value == "" {
		http.Error(w, "Não autorizado", http.StatusUnauthorized)
		return
	}

	ws, errUpgrade := upgrader.Upgrade(w, r, nil)
	if errUpgrade != nil {
		log.Printf("❌ [WEB TERMINAL] Erro de upgrade WebSocket: %v", errUpgrade)
		return
	}
	defer ws.Close()

	mode := r.URL.Query().Get("mode") // "docker", "ssh", or "shell"
	target := strings.TrimSpace(r.URL.Query().Get("target"))

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("powershell.exe")
	} else {
		// Spawn bash on host system so terminal session stays open
		cmd = exec.Command("bash")
	}

	// Spawn pseudo-terminal (PTY)
	ptmx, errPty := pty.Start(cmd)
	if errPty != nil && runtime.GOOS != "windows" {
		cmd = exec.Command("sh")
		ptmx, errPty = pty.Start(cmd)
	}

	if errPty != nil {
		log.Printf("❌ [WEB TERMINAL] Erro ao iniciar PTY: %v", errPty)
		_ = ws.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("\r\n❌ Erro ao iniciar PTY Terminal: %v\r\n", errPty)))
		return
	}

	defer func() {
		_ = ptmx.Close()
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
	}()

	// PTY -> WebSocket (Output)
	go func() {
		buf := make([]byte, 4096)
		for {
			n, errRead := ptmx.Read(buf)
			if errRead != nil {
				if errRead != io.EOF {
					log.Printf("⚠️ [WEB TERMINAL] PTY read finished: %v", errRead)
				}
				break
			}
			if errWrite := ws.WriteMessage(websocket.TextMessage, buf[:n]); errWrite != nil {
				break
			}
		}
	}()

	// Automatic startup command injection based on mode & target
	go func() {
		time.Sleep(200 * time.Millisecond) // Wait for shell startup
		if mode == "docker" {
			if target != "" && target != "imc-audio-local" {
				// Search for container matching target (name, token, or ID)
				startCmd := fmt.Sprintf("CONTAINER=$(docker ps --format '{{.Names}}' | grep '%s' | head -n 1); if [ -n \"$CONTAINER\" ]; then echo \"🚀 Conectando ao contêiner $CONTAINER...\"; docker exec -it \"$CONTAINER\" sh || docker exec -it \"$CONTAINER\" bash; else echo '⚠️ Contêiner \"%s\" não encontrado diretamente. Listando contêineres ativos:'; docker ps; fi\n", target, target)
				_, _ = ptmx.Write([]byte(startCmd))
			} else {
				// Default docker ps & auto connect to imc-radio or first container
				startCmd := "echo '📦 Contêineres Docker ativos no servidor:'; docker ps; CONTAINER=$(docker ps --format '{{.Names}}' | grep 'imc-radio' | head -n 1); if [ -n \"$CONTAINER\" ]; then echo \"\r\n🚀 Conectando ao contêiner $CONTAINER...\"; docker exec -it \"$CONTAINER\" sh; fi\n"
				_, _ = ptmx.Write([]byte(startCmd))
			}
		} else if mode == "ssh" && target != "" {
			startCmd := fmt.Sprintf("ssh -o StrictHostKeyChecking=no neoroute@%s\n", target)
			_, _ = ptmx.Write([]byte(startCmd))
		}
	}()

	// WebSocket -> PTY (Input & Resizing)
	for {
		_, msgBytes, errRead := ws.ReadMessage()
		if errRead != nil {
			break
		}

		var msg TerminalMessage
		if errJson := json.Unmarshal(msgBytes, &msg); errJson == nil {
			switch msg.Type {
			case "input":
				_, _ = ptmx.Write([]byte(msg.Data))
			case "resize":
				if msg.Cols > 0 && msg.Rows > 0 {
					_ = pty.Setsize(ptmx, &pty.Winsize{
						Cols: msg.Cols,
						Rows: msg.Rows,
					})
				}
			}
		} else {
			// Plain text input fallback
			_, _ = ptmx.Write(msgBytes)
		}
	}
}
