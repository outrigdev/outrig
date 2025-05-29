class OutrigAcresGame {
    constructor() {
        this.ws = null;
        this.gameState = null;
        this.speedMultiplier = 1;
        this.speeds = [1, 2, 5];
        this.speedIndex = 0;
        
        this.initializeUI();
        this.connectWebSocket();
    }
    
    initializeUI() {
        this.gameBoard = document.getElementById('gameBoard');
        this.tickCounter = document.getElementById('tickCounter');
        this.disconnectedOverlay = document.getElementById('disconnectedOverlay');
        this.pauseBtn = document.getElementById('pauseBtn');
        
        // Set up pause button event listener
        this.pauseBtn.addEventListener('click', () => {
            this.togglePause();
        });
        
        // Create the 30x30 grid
        this.createBoard();
    }
    
    createBoard() {
        this.gameBoard.innerHTML = '';
        
        // Create 900 cells (30x30)
        for (let row = 0; row < 30; row++) {
            for (let col = 0; col < 30; col++) {
                const cell = document.createElement('div');
                cell.className = 'cell empty';
                cell.dataset.row = row;
                cell.dataset.col = col;
                cell.id = `cell-${row}-${col}`;
                this.gameBoard.appendChild(cell);
            }
        }
    }
    
    connectWebSocket() {
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsUrl = `${protocol}//${window.location.host}/ws`;
        
        this.ws = new WebSocket(wsUrl);
        
        this.ws.onopen = () => {
            console.log('Connected to OutrigAcres server');
            this.disconnectedOverlay.style.display = 'none';
        };
        
        this.ws.onmessage = (event) => {
            try {
                this.gameState = JSON.parse(event.data);
                this.updateDisplay();
            } catch (error) {
                console.error('Error parsing game state:', error);
            }
        };
        
        this.ws.onclose = () => {
            console.log('Disconnected from server');
            this.disconnectedOverlay.style.display = 'block';
            setTimeout(() => this.connectWebSocket(), 2000); // Reconnect after 2 seconds
        };
        
        this.ws.onerror = (error) => {
            console.error('WebSocket error:', error);
            this.disconnectedOverlay.style.display = 'block';
        };
    }
    
    updateDisplay() {
        if (!this.gameState) return;
        
        // Update tick counter
        this.tickCounter.textContent = `Tick: ${this.gameState.tick}`;
        
        // Update pause button state
        this.updatePauseButton();
        
        // Update board cells
        this.updateBoard();
        
        // Update agents
        this.updateAgents();
    }
    
    updateBoard() {
        const board = this.gameState.board;
        
        for (let row = 0; row < 30; row++) {
            for (let col = 0; col < 30; col++) {
                const cell = document.getElementById(`cell-${row}-${col}`);
                const cellData = board[row][col];
                
                // Remove all cell type classes
                cell.className = 'cell';
                
                // Add the current cell type class
                cell.classList.add(cellData.type);
                
                // Add age information as a data attribute for potential debugging
                cell.dataset.age = cellData.ticksage;
            }
        }
    }
    
    updateAgents() {
        // Remove all existing agents
        document.querySelectorAll('.agent').forEach(agent => agent.remove());
        
        // Add current agents
        this.gameState.agents.forEach(agent => {
            const cell = document.getElementById(`cell-${agent.x}-${agent.y}`);
            if (cell) {
                const agentElement = document.createElement('div');
                agentElement.className = `agent agent-${agent.id}`;
                agentElement.textContent = agent.id;
                agentElement.title = `Agent #${agent.id} at (${agent.x}, ${agent.y})`;
                
                // Add target information if available
                if (agent.targetx >= 0 && agent.targety >= 0) {
                    agentElement.title += ` → (${agent.targetx}, ${agent.targety})`;
                }
                
                cell.appendChild(agentElement);
            }
        });
    }
    
    togglePause() {
        if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
            console.log('WebSocket not connected');
            return;
        }
        
        const isPaused = this.gameState && this.gameState.paused;
        const message = {
            type: isPaused ? 'unpause' : 'pause'
        };
        
        this.ws.send(JSON.stringify(message));
    }
    
    updatePauseButton() {
        if (!this.gameState) return;
        
        const isPaused = this.gameState.paused;
        this.pauseBtn.textContent = isPaused ? '▶️ Resume' : '⏸️ Pause';
        
        if (isPaused) {
            this.pauseBtn.classList.add('paused');
        } else {
            this.pauseBtn.classList.remove('paused');
        }
    }
    
    getCellTypeColor(cellType) {
        const colors = {
            'empty': '#27ae60',
            'mountain': '#7f8c8d',
            'wheat_seed': '#f1c40f',
            'wheat_growing': '#f39c12',
            'wheat_mature': '#e67e22',
            'wheat_withered': '#8b4513',
            'corn_seed': '#9b59b6',
            'corn_growing': '#8e44ad',
            'corn_mature': '#663399',
            'corn_withered': '#4a2c4a'
        };
        return colors[cellType] || '#34495e';
    }
    
    // Utility method to get cell info for debugging
    getCellInfo(row, col) {
        if (!this.gameState || !this.gameState.board[row] || !this.gameState.board[row][col]) {
            return null;
        }
        
        const cellData = this.gameState.board[row][col];
        const agentsAtCell = this.gameState.agents.filter(agent => agent.x === row && agent.y === col);
        
        return {
            position: { row, col },
            type: cellData.type,
            age: cellData.ticksage,
            agents: agentsAtCell
        };
    }
}

// Initialize the game when the page loads
document.addEventListener('DOMContentLoaded', () => {
    window.game = new OutrigAcresGame();
    
    // Add click handler for cells to show debug info
    document.addEventListener('click', (event) => {
        if (event.target.classList.contains('cell')) {
            const row = parseInt(event.target.dataset.row);
            const col = parseInt(event.target.dataset.col);
            const cellInfo = window.game.getCellInfo(row, col);
            
            if (cellInfo) {
                console.log('Cell Info:', cellInfo);
            }
        }
    });
    
    // Expose game instance for debugging
    console.log('OutrigAcres game initialized. Access via window.game');
});