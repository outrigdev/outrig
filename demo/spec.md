# OutrigAcres - Idle Farming Game - Demo App Specification

## Overview

A simple idle farming game demonstrating the capabilities of **Outrig** for debugging and observing Go applications.

## Tech Stack

- Runs a Go Backend (so that we can integrate Outrig), separate go.mod file to isolate dependencies
- Simple frontend (embedded in go app, in development we should read directly, in prod we embed)
    - index.html
    - game.js
    - game.css
    - assets/ directory for tiles/sprites/svgs
- Agents will work with GoRoutines + Synchronization against the board state
- The board updates itself every second (tick)
- The agents move on their own every second (tick)
- Even though we say "tick" they are not coordinated, e.g. agents/board can update at different times during the second, no issue
- Updates are transmitted very simply through a websocket interface, that just updates the visual board state.

## Game Board

- Size: **30x30** grid (900 cells)
- Cell Types:

    - **Empty** (light green)
    - **Mountain** (randomly placed, non-growable, static obstacle)
    - **Crop Tiles** (two types: Wheat and Corn)

        - Wheat Seed / Growing / Mature / Withered
        - Corn Seed / Growing / Mature / Withered

## Cell Lifecycle

- **Empty → Seed**

    - Randomly spawn: 1–3 new seeds per tick with a probability of \~10% per empty cell

- **Seed → Growing**: after **3 ticks**
- **Growing → Mature**: after additional **3 ticks**
- **Mature → Withered**: after **8 ticks** if not harvested
- **Withered → Empty**: after **3 ticks**

## Agents

- Count: Initially **8–12**, user-adjustable (add/remove)
- Movement: **1 square per tick**
- Pathfinding:

    - Prioritize nearest mature crop within radius **5** (Manhattan distance)
    - Move directly towards target crop
    - If no mature crops in radius, move randomly

- Harvesting: Immediate upon arrival at mature crop, resets cell to **Empty**

## Game Timing

- Default Tick Duration: **1 second**
- Adjustable game speed: **1x, 2x, 5x** controls provided in UI

## Logging for Outrig Integration

- Example logs:

    - `Agent#3 moved to (12, 17)`
    - `Agent#7 harvested Wheat at (15, 20)`
    - `New Corn seed spawned at (22, 9)`
    - `Wheat at (14, 14) withered`

## User Interface (Frontend - React & WebSockets)

- Grid display clearly shows cell state:

    - Color/icon clearly indicates crop stage
    - Distinct representation for mountains

- Agents visually represented with IDs
- No separate stats sidebar; monitoring fully done through **Outrig**

## Monitoring via Outrig

- Real-time log streaming
- Goroutine tracking
- Atomic value inspection: total harvested value, agent states, cell state counts, etc.
