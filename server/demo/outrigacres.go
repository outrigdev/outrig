package demo

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"math"
	"math/rand"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/outrigdev/outrig"
	"github.com/outrigdev/outrig/pkg/utilfn"
)

//go:embed frontend/*
var frontendFS embed.FS

var (
	globalGame *Game
)

const (
	BoardSize = 30

	// Server constants
	PreferredPort = 22005 // Preferred port for the demo server

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
)

type Cell struct {
	Type     string `json:"type"`
	TicksAge int    `json:"ticksage"`
}

type Agent struct {
	ID        int           `json:"id"`
	X         int           `json:"x"`
	Y         int           `json:"y"`
	TargetX   int           `json:"targetx"`
	TargetY   int           `json:"targety"`
	Score     int           `json:"score"`
	MoveSpeed float64       `json:"movespeed"`
	mu        *sync.RWMutex `json:"-"`
	watch     *outrig.Watch `json:"-"`
}

type GameState struct {
	Board  [][]Cell `json:"board"`
	Agents []*Agent `json:"agents"`
	Tick   int      `json:"tick"`
	Paused bool     `json:"paused"`
}

type BoardUpdate struct {
	Type   string   `json:"type"`
	Board  [][]Cell `json:"board"`
	Tick   int      `json:"tick"`
	Paused bool     `json:"paused"`
}

type AgentUpdate struct {
	Type      string `json:"type"`
	AgentID   int    `json:"agentid"`
	X         int    `json:"x"`
	Y         int    `json:"y"`
	Harvested bool   `json:"harvested"`
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
	g.state.Agents = make([]*Agent, 0) // Initialize empty agents slice

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

	// Only use wheat crops
	cropType := "wheat"

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

	// Set move speed - agents 1 and 2 are twice as fast
	var moveSpeed float64 = 1.0
	if agentID == 1 || agentID == 2 {
		moveSpeed = 2.0
	}

	agent := &Agent{
		ID:        agentID,
		X:         x,
		Y:         y,
		TargetX:   -1,
		TargetY:   -1,
		Score:     0,
		MoveSpeed: moveSpeed,
		mu:        &sync.RWMutex{},
	}

	// Set up watch for this agent
	watchName := fmt.Sprintf("agent-%d", agentID)
	agent.watch = outrig.NewWatch(watchName).WithTags("agent", "simulation").AsJSON().PollSync(agent.mu, agent)

	// Add agent to the slice
	g.state.Agents = append(g.state.Agents, agent)

	log.Printf("Agent#%d spawned at (%d, %d) with watch '%s'", agentID, x, y, watchName)
}

func (g *Game) Start() {
	outrig.Go("game-loop").WithTags("game", "simulation").Run(func() {
		g.gameLoop()
	})

	// Start individual agent goroutines
	agentCount := 9 // agents 1-9
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
	// Find the agent to get its move speed
	g.mu.RLock()
	var agent *Agent
	for i := range g.state.Agents {
		if g.state.Agents[i].ID == agentID {
			agent = g.state.Agents[i]
			break
		}
	}
	g.mu.RUnlock()

	if agent == nil {
		return
	}

	// Calculate tick duration: 1 second / move speed
	tickDuration := time.Duration(float64(time.Second) / agent.MoveSpeed)
	ticker := time.NewTicker(tickDuration)
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

			case CellWheatSeed:
				if cell.TicksAge >= GrowTicks {
					cell.Type = CellWheatGrow
					cell.TicksAge = 0
				}

			case CellWheatGrow:
				if cell.TicksAge >= GrowTicks {
					cell.Type = CellWheatMature
					cell.TicksAge = 0
				}

			case CellWheatMature:
				if cell.TicksAge >= MatureTicks {
					cell.Type = CellWheatWither
					log.Printf("Wheat at (%d, %d) withered", x, y)
					cell.TicksAge = 0
				}

			case CellWheatWither:
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
			agent = g.state.Agents[i]
			break
		}
	}

	if agent == nil {
		return
	}

	// Lock the agent for updates
	agent.mu.Lock()
	defer agent.mu.Unlock()

	// Find target if we don't have one or if current target is invalid
	if agent.TargetX == -1 || agent.TargetY == -1 || g.isTargetInvalid(agent) {
		g.findTarget(agent)
	}

	// Move towards target
	g.moveAgent(agent)
}

func (g *Game) findTarget(agent *Agent) {
	// Search in expanding Manhattan distance rings for efficiency
	// Priority: mature > growing > seed

	// Try to find mature crops first
	if x, y := g.findCropAtDistance(agent, []string{CellWheatMature}); x != -1 {
		agent.TargetX = x
		agent.TargetY = y
		return
	}

	// If no mature crops, try growing crops
	if x, y := g.findCropAtDistance(agent, []string{CellWheatGrow}); x != -1 {
		agent.TargetX = x
		agent.TargetY = y
		return
	}

	// If no growing crops, try seed crops
	if x, y := g.findCropAtDistance(agent, []string{CellWheatSeed}); x != -1 {
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

func (g *Game) harvestAtPosition(agent *Agent) bool {
	cell := &g.state.Board[agent.X][agent.Y]
	if cell.Type == CellWheatMature {
		cell.Type = CellEmpty
		cell.TicksAge = 0
		agent.Score++
		log.Printf("Agent#%d harvested Wheat at (%d, %d) - Score: %d", agent.ID, agent.X, agent.Y, agent.Score)
		// Clear target since we harvested something
		agent.TargetX = -1
		agent.TargetY = -1
		return true
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
		// Try alternative directions in random order to avoid loops
		directions := []struct{ dx, dy int }{
			{0, -1}, // up
			{0, 1},  // down
			{-1, 0}, // left
			{1, 0},  // right
		}

		moved := false
		for _, i := range rand.Perm(len(directions)) {
			dir := directions[i]
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
		// Check for harvest after moving
		harvested := g.harvestAtPosition(agent)
		// Broadcast agent update when position changes
		g.broadcastAgentUpdate(agent, harvested)
	}

	// If we reached our target, clear it to find a new one
	if agent.X == agent.TargetX && agent.Y == agent.TargetY {
		agent.TargetX = -1
		agent.TargetY = -1
	}
}

func (g *Game) broadcastMessage(data []byte) {
	g.clientsMu.RLock()
	defer g.clientsMu.RUnlock()

	for client := range g.clients {
		err := client.WriteMessage(websocket.TextMessage, data)
		if err != nil {
			log.Printf("Error sending message to client: %v", err)
			client.Close()
			delete(g.clients, client)
		}
	}
}

func (g *Game) broadcastState() {
	// Send board update only
	boardUpdate := BoardUpdate{
		Type:   "board",
		Board:  g.state.Board,
		Tick:   g.state.Tick,
		Paused: g.paused,
	}

	data, err := json.Marshal(boardUpdate)
	if err != nil {
		log.Printf("Error marshaling board update: %v", err)
		return
	}

	g.broadcastMessage(data)
}

func (g *Game) broadcastAgentUpdate(agent *Agent, harvested bool) {
	agentUpdate := AgentUpdate{
		Type:      "agent",
		AgentID:   agent.ID,
		X:         agent.X,
		Y:         agent.Y,
		Harvested: harvested,
	}

	data, err := json.Marshal(agentUpdate)
	if err != nil {
		log.Printf("Error marshaling agent update: %v", err)
		return
	}

	g.broadcastMessage(data)
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

	// Send initial board state
	g.mu.RLock()
	boardUpdate := BoardUpdate{
		Type:   "board",
		Board:  g.state.Board,
		Tick:   g.state.Tick,
		Paused: g.paused,
	}
	boardData, _ := json.Marshal(boardUpdate)
	g.mu.RUnlock()
	conn.WriteMessage(websocket.TextMessage, boardData)

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

func getBoardCellCounts() map[string]int {
	if globalGame == nil {
		return make(map[string]int)
	}

	globalGame.mu.RLock()
	defer globalGame.mu.RUnlock()

	counts := make(map[string]int)

	for x := 0; x < BoardSize; x++ {
		for y := 0; y < BoardSize; y++ {
			cellType := globalGame.state.Board[x][y].Type
			counts[cellType]++
		}
	}

	return counts
}

func getTotalScore() int {
	if globalGame == nil {
		return 0
	}

	globalGame.mu.RLock()
	defer globalGame.mu.RUnlock()

	totalScore := 0
	for _, agent := range globalGame.state.Agents {
		if agent != nil {
			agent.mu.RLock()
			totalScore += agent.Score
			agent.mu.RUnlock()
		}
	}

	return totalScore
}

func (g *Game) isTargetInvalid(agent *Agent) bool {
	// Check if target coordinates are valid
	if agent.TargetX < 0 || agent.TargetX >= BoardSize || agent.TargetY < 0 || agent.TargetY >= BoardSize {
		return true
	}

	// Check if target cell is withered or empty
	targetCell := g.state.Board[agent.TargetX][agent.TargetY]
	return targetCell.Type == CellWheatWither || targetCell.Type == CellEmpty || targetCell.Type == CellMountain
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func RunOutrigAcres(devMode bool, noBrowserLaunch bool, port int) {

	// Initialize Outrig
	outrig.Init("OutrigAcres", nil)
	outrig.SetGoRoutineName("main")

	globalGame = NewGame()

	// Set up global board watch to track cell type counts
	outrig.NewWatch("gameboard-cells").WithTags("board", "simulation").AsJSON().PollFunc(getBoardCellCounts)

	// Set up total score watch to track sum of all agent scores
	outrig.NewWatch("totalscore").WithTags("score", "simulation").PollFunc(getTotalScore)

	globalGame.Start()

	// Serve frontend files
	if devMode {
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
	http.HandleFunc("/ws", globalGame.handleWebSocket)

	// Use specified port (default or from flag)
	if port == 0 {
		port = PreferredPort
	}

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Printf("Could not bind to port %d: %v", port, err)
		os.Exit(1)
	}

	url := fmt.Sprintf("http://localhost:%d", port)

	// Set up watch for the game URL
	outrig.NewWatch("game-url").Static(url)

	log.Printf("OutrigAcres demo launched, available at: %s", url)
	if !noBrowserLaunch {
		err := utilfn.LaunchUrl(url)
		if err != nil {
			log.Printf("Failed to open browser: %v", err)
		}
	}
	log.Fatal(http.Serve(listener, nil))
}
