# OutrigAcres - Idle Farming Game Demo

A demonstration game showcasing Outrig's real-time debugging capabilities for Go applications.

## Overview

OutrigAcres is a simple idle farming game with:

- 30x30 grid game board
- Different cell types (Empty, Mountain, Wheat, Corn in various growth stages)
- 8-12 autonomous agents that move around and harvest crops
- Real-time WebSocket updates to the frontend
- Integration with Outrig for logging and debugging

## Running the Demo

1. Build the application:

    ```bash
    go build -o outrigacres .
    ```

2. Run the server:

    ```bash
    ./outrigacres
    ```

3. Open your browser and navigate to:
    ```
    http://localhost:8080
    ```

## Game Mechanics

### Board

- **30x30 grid** (900 cells total)
- **Cell Types:**
    - Empty (light green)
    - Mountain (gray, static obstacles)
    - Wheat: Seed → Growing → Mature → Withered → Empty
    - Corn: Seed → Growing → Mature → Withered → Empty

### Cell Lifecycle

- Empty cells have a 10% chance per tick to spawn new seeds
- Seeds take 3 ticks to grow
- Growing crops take 3 ticks to mature
- Mature crops wither after 8 ticks if not harvested
- Withered crops return to empty after 3 ticks

### Agents

- 8-12 agents spawn randomly on the board
- Move 1 square per tick (1 second intervals)
- Prioritize nearest mature crop within radius 5
- Harvest crops instantly upon arrival
- Move randomly if no crops are in range

## Outrig Integration

The game demonstrates Outrig's capabilities:

- **Real-time logging**: All game events are logged (agent movements, harvests, crop changes)
- **Goroutine monitoring**: Game loop runs in a named goroutine using `outrig.Go("game-loop")`
- **WebSocket communication**: Real-time updates between Go backend and JavaScript frontend

## Frontend

- **HTML/CSS/JavaScript** frontend embedded in the Go binary
- **WebSocket connection** for real-time game state updates
- **Visual representation** of the game board with colored tiles and numbered agents
- **Speed controls** (UI placeholder for future server-side implementation)

## File Structure

```
demo/
├── main-outrigacres.go  # Go backend with game logic
├── go.mod               # Go module definition
├── README.md            # This file
├── frontend/
│   ├── index.html       # Main HTML page
│   ├── game.css         # Styling for game board and agents
│   ├── game.js          # JavaScript game client
│   └── assets/          # Directory for future game assets
└── outrigacres          # Compiled binary (after build)
```

## Development Notes

- The frontend files are embedded using `//go:embed frontend/*`
- Game state is synchronized via JSON over WebSocket
- All game logic runs server-side for consistency
- CSS provides simple colored tiles and numbered agent circles
- Future enhancements could include PNG sprites in the assets directory
