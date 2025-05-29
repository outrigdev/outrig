package main

import (
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"math"
	"math/rand"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/outrigdev/outrig"
)

//go:embed frontend/*
var frontendFS embed.FS

const (
	BoardSize = 30

	// Crop timing constants
	GrowTicks   = 5  // Ticks for seed to grow to growing stage
	MatureTicks = 12 // Ticks for growing to mature stage
	WitherTicks = 4  // Ticks for withered crops to disappear

	// Agent search constants
	MaxSearchRadius = 12 // Maximum Manhattan distance to search for targets

	// Cell types
	CellEmpty       = "empty"
	CellMountain    = "mountain"
	CellWheatSeed   = "wheat_seed"
	CellWheatGrow   = "wheat_growing"
	CellWheatMature = "wheat_mature"
	CellWheatWither = "wheat_withered"
	CellCornSeed    = "corn_seed"
	CellCornGrow    = "corn_growing"
	CellCornMature  = "corn_mature"
	CellCornWither  = "corn_withered"
)

type Cell struct {
	Type     string `json:"type"`
	TicksAge int    `json:"ticksage"`
}

type Agent struct {
	ID      int `json:"id"`
	X       int `json:"x"`
	Y       int `json:"y"`
	TargetX int `json:"targetx"`
	TargetY int `json:"targety"`
}

type GameState struct {
	Board  [][]Cell `json:"board"`
	Agents []Agent  `json:"agents"`
	Tick   int      `json:"tick"`
	Paused bool     `json:"paused"`
}

type Game struct {
	mu        sync.RWMutex
	state     GameState
	clients   map[*websocket.Conn]bool
	clientsMu sync.RWMutex
	tickSpeed time.Duration
	stopChan  chan bool
	paused    bool
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func NewGame() *Game {
	g := &Game{
		clients:   make(map[*websocket.Conn]bool),
		tickSpeed: time.Second,
		stopChan:  make(chan bool),
	}

	g.initializeBoard()
	g.state.Agents = make([]Agent, 0) // Initialize empty agents slice

	return g
}

func (g *Game) initializeBoard() {
	g.state.Board = make([][]Cell, BoardSize)
	for i := range g.state.Board {
		g.state.Board[i] = make([]Cell, BoardSize)
		for j := range g.state.Board[i] {
			g.state.Board[i][j] = Cell{Type: CellEmpty, TicksAge: 0}
		}
	}

	// Add some random mountains (about 5% of the board)
	mountainCount := (BoardSize * BoardSize) / 20
	for i := 0; i < mountainCount; i++ {
		x := rand.Intn(BoardSize)
		y := rand.Intn(BoardSize)
		g.state.Board[x][y] = Cell{Type: CellMountain, TicksAge: 0}
	}

	// Add some crop clusters
	clusterCount := 5 + rand.Intn(3) // 3-5 clusters
	for i := 0; i < clusterCount; i++ {
		g.addCropCluster(false) // Allow different growth stages
	}

	log.Printf("Initialized %dx%d board with %d mountains", BoardSize, BoardSize, mountainCount)
}

func (g *Game) addCropCluster(seedsOnly bool) {
	// Pick random center point
	centerX := 2 + rand.Intn(BoardSize-4) // Stay away from edges (reduced restriction)
	centerY := 2 + rand.Intn(BoardSize-4)

	// Pick crop type for this cluster
	cropType := "wheat"
	if rand.Float32() < 0.5 {
		cropType = "corn"
	}

	// Create cluster with radius 2-4
	radius := 2 + rand.Intn(3)
	clusterSize := 8 + rand.Intn(12) // 8-19 cells per cluster

	placedCount := 0
	attempts := 0
	maxAttempts := clusterSize * 3

	for placedCount < clusterSize && attempts < maxAttempts {
		attempts++

		// Generate point within cluster radius using circular distribution
		angle := rand.Float64() * 2 * math.Pi
		distance := rand.Float64() * float64(radius)

		x := centerX + int(distance*math.Cos(angle))
		y := centerY + int(distance*math.Sin(angle))

		// Check bounds
		if x < 0 || x >= BoardSize || y < 0 || y >= BoardSize {
			continue
		}

		// Don't overwrite mountains or existing crops
		if g.state.Board[x][y].Type != CellEmpty {
			continue
		}

		// Pick growth stage
		var cellType string
		var ticksAge int

		if seedsOnly {
			// Only produce seeds
			cellType = cropType + "_seed"
			ticksAge = rand.Intn(3)
		} else {
			// Pick growth stage (weighted toward earlier stages)
			stageRoll := rand.Float32()
			if stageRoll < 0.4 {
				// Seed stage
				cellType = cropType + "_seed"
				ticksAge = rand.Intn(3)
			} else if stageRoll < 0.7 {
				// Growing stage
				cellType = cropType + "_growing"
				ticksAge = rand.Intn(3)
			} else {
				// Mature stage
				cellType = cropType + "_mature"
				ticksAge = rand.Intn(4) // Don't make them too close to withering
			}
		}

		g.state.Board[x][y] = Cell{Type: cellType, TicksAge: ticksAge}
		placedCount++
	}

	log.Printf("Added %s cluster at (%d, %d) with %d crops", cropType, centerX, centerY, placedCount)
}

func (g *Game) initializeAgent(agentID int) {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Find empty spot for agent (no mountains and no other agents)
	var x, y int
	for {
		x = rand.Intn(BoardSize)
		y = rand.Intn(BoardSize)
		if g.state.Board[x][y].Type == CellEmpty && !g.isPositionOccupied(x, y, agentID) {
			break
		}
	}

	agent := Agent{
		ID:      agentID,
		X:       x,
		Y:       y,
		TargetX: -1,
		TargetY: -1,
	}

	// Add agent to the slice
	g.state.Agents = append(g.state.Agents, agent)

	log.Printf("Agent#%d spawned at (%d, %d)", agentID, x, y)
}

func (g *Game) Start() {
	outrig.Go("game-loop").WithTags("game", "simulation").Run(func() {
		g.gameLoop()
	})

	// Start individual agent goroutines
	agentCount := 8 + rand.Intn(5) // 8-12 agents
	for i := 0; i < agentCount; i++ {
		agentID := i + 1
		outrig.Go(fmt.Sprintf("agent-%d", agentID)).WithTags("agent", "simulation").Run(func() {
			g.initializeAgent(agentID)
			g.agentLoop(agentID)
		})
	}
}

func (g *Game) Stop() {
	g.stopChan <- true
}

func (g *Game) SetPaused(paused bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.paused = paused
	log.Printf("Game paused: %v", paused)
}

func (g *Game) IsPaused() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.paused
}

func (g *Game) gameLoop() {
	ticker := time.NewTicker(g.tickSpeed)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			g.tick()
		case <-g.stopChan:
			return
		}
	}
}

func (g *Game) agentLoop(agentID int) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			g.updateAgent(agentID)
		case <-g.stopChan:
			return
		}
	}
}

func (g *Game) tick() {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.paused {
		// Still broadcast state to keep clients updated, but don't advance game logic
		g.broadcastState()
		return
	}

	g.state.Tick++

	// Update board cells
	g.updateBoard()

	// Broadcast state to all clients
	g.broadcastState()
}

func (g *Game) updateBoard() {
	// First, age all cells and handle crop growth
	for x := 0; x < BoardSize; x++ {
		for y := 0; y < BoardSize; y++ {
			cell := &g.state.Board[x][y]
			cell.TicksAge++

			switch cell.Type {
			case CellEmpty:
				// Don't spawn seeds here anymore - we'll do it randomly below

			case CellWheatSeed, CellCornSeed:
				if cell.TicksAge >= GrowTicks {
					if cell.Type == CellWheatSeed {
						cell.Type = CellWheatGrow
					} else {
						cell.Type = CellCornGrow
					}
					cell.TicksAge = 0
				}

			case CellWheatGrow, CellCornGrow:
				if cell.TicksAge >= GrowTicks {
					if cell.Type == CellWheatGrow {
						cell.Type = CellWheatMature
					} else {
						cell.Type = CellCornMature
					}
					cell.TicksAge = 0
				}

			case CellWheatMature, CellCornMature:
				if cell.TicksAge >= MatureTicks {
					if cell.Type == CellWheatMature {
						cell.Type = CellWheatWither
						log.Printf("Wheat at (%d, %d) withered", x, y)
					} else {
						cell.Type = CellCornWither
						log.Printf("Corn at (%d, %d) withered", x, y)
					}
					cell.TicksAge = 0
				}

			case CellWheatWither, CellCornWither:
				if cell.TicksAge >= WitherTicks {
					cell.Type = CellEmpty
					cell.TicksAge = 0
				}
			}
		}
	}

	// Spawn new crop cluster (seeds only) every 3 ticks
	if g.state.Tick%3 == 0 {
		g.addCropCluster(true) // Only seeds
	}
}

func (g *Game) updateAgent(agentID int) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.paused {
		return
	}

	// Find the agent by ID
	var agent *Agent
	for i := range g.state.Agents {
		if g.state.Agents[i].ID == agentID {
			agent = &g.state.Agents[i]
			break
		}
	}

	if agent == nil {
		return
	}

	// Find target if we don't have one
	if agent.TargetX == -1 || agent.TargetY == -1 {
		g.findTarget(agent)
	}

	// Move towards target
	g.moveAgent(agent)

	// Check if we reached target and can harvest
	if agent.X == agent.TargetX && agent.Y == agent.TargetY {
		cell := &g.state.Board[agent.X][agent.Y]
		if cell.Type == CellWheatMature || cell.Type == CellCornMature {
			cropType := "Wheat"
			if cell.Type == CellCornMature {
				cropType = "Corn"
			}
			cell.Type = CellEmpty
			cell.TicksAge = 0
			log.Printf("Agent#%d harvested %s at (%d, %d)", agent.ID, cropType, agent.X, agent.Y)
		}
		// Clear target
		agent.TargetX = -1
		agent.TargetY = -1
	}
}

func (g *Game) findTarget(agent *Agent) {
	// Search in expanding Manhattan distance rings for efficiency
	// Priority: mature > growing > seed
	
	// Try to find mature crops first
	if x, y := g.findCropAtDistance(agent, []string{CellWheatMature, CellCornMature}); x != -1 {
		agent.TargetX = x
		agent.TargetY = y
		return
	}
	
	// If no mature crops, try growing crops
	if x, y := g.findCropAtDistance(agent, []string{CellWheatGrow, CellCornGrow}); x != -1 {
		agent.TargetX = x
		agent.TargetY = y
		return
	}
	
	// If no growing crops, try seed crops
	if x, y := g.findCropAtDistance(agent, []string{CellWheatSeed, CellCornSeed}); x != -1 {
		agent.TargetX = x
		agent.TargetY = y
		return
	}
	
	// No crops found, move randomly
	agent.TargetX = rand.Intn(BoardSize)
	agent.TargetY = rand.Intn(BoardSize)
}

func (g *Game) findCropAtDistance(agent *Agent, cropTypes []string) (int, int) {
	// Search in expanding Manhattan distance rings from 1 to MaxSearchRadius
	for dist := 1; dist <= MaxSearchRadius; dist++ {
		// Search all positions at exactly this Manhattan distance
		for dx := -dist; dx <= dist; dx++ {
			for dy := -dist; dy <= dist; dy++ {
				// Only check positions that are exactly at this Manhattan distance
				if abs(dx)+abs(dy) != dist {
					continue
				}
				
				x := agent.X + dx
				y := agent.Y + dy
				
				// Check bounds
				if x < 0 || x >= BoardSize || y < 0 || y >= BoardSize {
					continue
				}
				
				cell := g.state.Board[x][y]
				for _, cropType := range cropTypes {
					if cell.Type == cropType {
						return x, y
					}
				}
			}
		}
	}
	
	return -1, -1
}

func (g *Game) isPositionOccupied(x, y int, excludeAgentID int) bool {
	for _, otherAgent := range g.state.Agents {
		if otherAgent.ID != excludeAgentID && otherAgent.X == x && otherAgent.Y == y {
			return true
		}
	}
	return false
}

func (g *Game) moveAgent(agent *Agent) {
	oldX, oldY := agent.X, agent.Y

	// Try to move one step towards target
	newX, newY := oldX, oldY
	if agent.X < agent.TargetX {
		newX++
	} else if agent.X > agent.TargetX {
		newX--
	} else if agent.Y < agent.TargetY {
		newY++
	} else if agent.Y > agent.TargetY {
		newY--
	}

	// Check bounds
	if newX < 0 {
		newX = 0
	}
	if newX >= BoardSize {
		newX = BoardSize - 1
	}
	if newY < 0 {
		newY = 0
	}
	if newY >= BoardSize {
		newY = BoardSize - 1
	}

	// Check if we hit a mountain or another agent, try alternative directions
	if g.state.Board[newX][newY].Type == CellMountain || g.isPositionOccupied(newX, newY, agent.ID) {
		// Try alternative directions in order: up, down, left, right
		directions := []struct{ dx, dy int }{
			{0, -1}, // up
			{0, 1},  // down
			{-1, 0}, // left
			{1, 0},  // right
		}

		moved := false
		for _, dir := range directions {
			altX := oldX + dir.dx
			altY := oldY + dir.dy

			// Check bounds
			if altX >= 0 && altX < BoardSize && altY >= 0 && altY < BoardSize {
				// Check if this direction is clear (no mountain and no other agent)
				if g.state.Board[altX][altY].Type != CellMountain && !g.isPositionOccupied(altX, altY, agent.ID) {
					agent.X = altX
					agent.Y = altY
					moved = true
					break
				}
			}
		}

		// If no direction worked, clear target to find a new one
		if !moved {
			agent.TargetX = -1
			agent.TargetY = -1
		}
	} else {
		// Move to the intended position
		agent.X = newX
		agent.Y = newY
	}

	if agent.X != oldX || agent.Y != oldY {
		log.Printf("Agent#%d moved to (%d, %d)", agent.ID, agent.X, agent.Y)
	}
}

func (g *Game) broadcastState() {
	g.clientsMu.RLock()
	defer g.clientsMu.RUnlock()

	// Update pause status in state before broadcasting
	g.state.Paused = g.paused

	data, err := json.Marshal(g.state)
	if err != nil {
		log.Printf("Error marshaling game state: %v", err)
		return
	}

	for client := range g.clients {
		err := client.WriteMessage(websocket.TextMessage, data)
		if err != nil {
			log.Printf("Error sending to client: %v", err)
			client.Close()
			delete(g.clients, client)
		}
	}
}

func (g *Game) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}
	defer conn.Close()

	g.clientsMu.Lock()
	g.clients[conn] = true
	g.clientsMu.Unlock()

	// Send initial state
	g.mu.RLock()
	data, _ := json.Marshal(g.state)
	g.mu.RUnlock()
	conn.WriteMessage(websocket.TextMessage, data)

	// Handle incoming messages
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			break
		}

		// Parse message as JSON
		var msg map[string]interface{}
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("Error parsing WebSocket message: %v", err)
			continue
		}

		// Handle different message types
		if msgType, ok := msg["type"].(string); ok {
			switch msgType {
			case "pause":
				g.SetPaused(true)
			case "unpause":
				g.SetPaused(false)
			default:
				log.Printf("Unknown message type: %s", msgType)
			}
		}
	}

	g.clientsMu.Lock()
	delete(g.clients, conn)
	g.clientsMu.Unlock()
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func main() {
	// Parse command line flags
	devMode := flag.Bool("dev", false, "Run in development mode (serve files from disk)")
	flag.Parse()

	// Initialize Outrig
	outrig.Init("OutrigAcres", nil)
	outrig.SetGoRoutineName("main")

	game := NewGame()
	game.Start()

	// Serve frontend files
	if *devMode {
		log.Printf("Running in development mode - serving files from disk")
		fileServer := http.FileServer(http.Dir("./frontend/"))
		http.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Add cache control headers to prevent caching in dev mode
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
			w.Header().Set("Pragma", "no-cache")
			w.Header().Set("Expires", "0")
			fileServer.ServeHTTP(w, r)
		}))
	} else {
		frontendSubFS, _ := fs.Sub(frontendFS, "frontend")
		http.Handle("/", http.FileServer(http.FS(frontendSubFS)))
	}

	// WebSocket endpoint
	http.HandleFunc("/ws", game.handleWebSocket)

	// Get an available port
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		log.Fatal(err)
	}

	port := listener.Addr().(*net.TCPAddr).Port
	url := fmt.Sprintf("http://localhost:%d", port)

	log.Printf("OutrigAcres demo server starting on port %d", port)
	log.Printf("Game available at: %s", url)

	// Open browser on macOS
	if runtime.GOOS == "darwin" {
		outrig.Go("browser-opener").WithTags("browser", "utility").Run(func() {
			time.Sleep(500 * time.Millisecond) // Give server a moment to start
			exec.Command("open", url).Start()
		})
	}

	log.Fatal(http.Serve(listener, nil))
}
