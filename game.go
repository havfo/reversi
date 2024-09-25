package main

// Game represents the game state
type Game struct {
	board      *Board
	current    int
	blackAI    bool
	whiteAI    bool
	moveCount  int
	difficulty int
}

// NewGame initializes a new game with the starting position
func NewGame() *Game {
	return &Game{
		board:      NewBoard(),
		current:    Black,
		difficulty: 5,
		moveCount:  4,
	}
}

type GamePhase int

const (
	EarlyGame GamePhase = iota
	MidGame
	LateGame
)

var CellHeuristics = [8][8]int{
	{100, -20, 10, 5, 5, 10, -20, 100},
	{-20, -50, -2, -2, -2, -2, -50, -20},
	{10, -2, 5, 1, 1, 5, -2, 10},
	{5, -2, 1, 1, 1, 1, -2, 5},
	{5, -2, 1, 1, 1, 1, -2, 5},
	{10, -2, 5, 1, 1, 5, -2, 10},
	{-20, -50, -2, -2, -2, -2, -50, -20},
	{100, -20, 10, 5, 5, 10, -20, 100},
}

// GetGamePhase determines the current phase of the game
func (g *Game) GetGamePhase() GamePhase {
	switch {
	case g.moveCount <= 20:
		return EarlyGame
	case g.moveCount > 20 && g.moveCount <= 44:
		return MidGame
	default:
		return LateGame
	}
}

// SwitchTurn switches the current player
func (g *Game) SwitchTurn() {
	if g.current == Black {
		g.current = White
	} else {
		g.current = Black
	}
}

// Opponent returns the opponent of the given player
func Opponent(player int) int {
	if player == Black {
		return White
	}

	return Black
}

// ValidMoves returns a list of valid moves for the specified player
func (g *Game) ValidMoves(player int) []Move {
	var moves []Move
	for x := 0; x < BoardSize; x++ {
		for y := 0; y < BoardSize; y++ {
			if g.board[x][y] == Blank {
				if flips := g.Flips(x, y, player); len(flips) > 0 {
					moves = append(moves, Move{X: x, Y: y, Flips: flips})
				}
			}
		}
	}

	return moves
}

// Flips returns the list of pieces that would be flipped if a piece is placed at (x, y)
func (g *Game) Flips(x, y, player int) [][2]int {
	var totalFlips [][2]int
	opponent := Opponent(player)

	for _, dir := range directions {
		var flips [][2]int
		nx, ny := x+dir.x, y+dir.y

		for nx >= 0 && nx < BoardSize && ny >= 0 && ny < BoardSize && g.board[nx][ny] == opponent {
			flips = append(flips, [2]int{nx, ny})
			nx += dir.x
			ny += dir.y
		}

		if nx >= 0 && nx < BoardSize && ny >= 0 && ny < BoardSize && g.board[nx][ny] == player && len(flips) > 0 {
			totalFlips = append(totalFlips, flips...)
		}
	}

	return totalFlips
}

// MakeMove applies the move to the game state
func (g *Game) MakeMove(move Move, switchTurn bool) {
	g.board[move.X][move.Y] = g.current

	for _, flip := range move.Flips {
		g.board[flip[0]][flip[1]] = g.current
	}

	g.moveCount++

	if switchTurn {
		g.SwitchTurn()
	}
}

// SimulateMove returns a new game state after applying the move
func (g *Game) SimulateMove(move Move, switchTurn bool) *Game {
	newGame := g.Copy()
	newGame.MakeMove(move, switchTurn)

	return newGame
}

// Evaluate evaluates the board using a heuristic function
type ScoreComponents struct {
	TotalScore           float64
	Heuristic            float64
	DiscDiff             float64
	Mobility             float64
	Frontier             float64
	PotentialMobility    float64
	CornerOwnership      float64
	EdgeStability        float64
	WeightHeuristic      float64
	WeightDiscDifference float64
	WeightMobility       float64
	WeightFrontier       float64
	WeightPotentialMob   float64
	WeightCorner         float64
	WeightEdge           float64
}

// EvaluateDetailed evaluates the board and returns the score components
func (g *Game) Evaluate(player int) float64 {
	components := g.EvaluateDetailed(player)
	return components.TotalScore
}

func (g *Game) EvaluateDetailed(player int) ScoreComponents {
	components := ScoreComponents{}

	phase := g.GetGamePhase()
	opponent := Opponent(player)

	// Adjust weights based on game phase
	switch phase {
	case EarlyGame:
		components.WeightHeuristic = 10.0
		components.WeightDiscDifference = 1.0
		components.WeightMobility = 5.0
		components.WeightFrontier = 5.0
		components.WeightPotentialMob = 5.0
		components.WeightCorner = 25.0
		components.WeightEdge = 5.0
	case MidGame:
		components.WeightHeuristic = 5.0
		components.WeightDiscDifference = 1.0
		components.WeightMobility = 10.0
		components.WeightFrontier = 10.0
		components.WeightPotentialMob = 10.0
		components.WeightCorner = 25.0
		components.WeightEdge = 10.0
	case LateGame:
		components.WeightHeuristic = 1.0
		components.WeightDiscDifference = 25.0
		components.WeightMobility = 1.0
		components.WeightFrontier = 1.0
		components.WeightPotentialMob = 1.0
		components.WeightCorner = 25.0
		components.WeightEdge = 15.0
	}

	// Variables to hold counts
	myHeuristic := 0
	myDiscs, opponentDiscs := 0, 0
	myFrontierDiscs, opponentFrontierDiscs := 0, 0
	myPotentialMobility, opponentPotentialMobility := 0, 0

	// Corner positions
	corners := [][2]int{{0, 0}, {0, 7}, {7, 0}, {7, 7}}
	myCorners := 0
	opponentCorners := 0

	// Directions for frontier detection
	directions := [][2]int{
		{-1, -1}, {-1, 0}, {-1, 1},
		{0, -1} /*{0, 0},*/, {0, 1},
		{1, -1}, {1, 0}, {1, 1},
	}

	// Evaluate board
	for x := 0; x < BoardSize; x++ {
		for y := 0; y < BoardSize; y++ {
			cell := g.board[x][y]

			if cell == Blank {
				// Potential mobility
				for _, dir := range directions {
					nx, ny := x+dir[0], y+dir[1]
					if nx >= 0 && nx < BoardSize && ny >= 0 && ny < BoardSize {
						neighborCell := g.board[nx][ny]
						if neighborCell == opponent {
							myPotentialMobility++

							break
						} else if neighborCell == player {
							opponentPotentialMobility++

							break
						}
					}
				}
			}

			if cell == player || cell == opponent {
				value := CellHeuristics[x][y]

				// Adjust for X-squares and C-squares
				if isXSquare(x, y) {
					cornerX, cornerY := adjacentCorner(x, y)
					cornerCell := g.board[cornerX][cornerY]
					if cornerCell != cell {
						value = -abs(value)
					}
				}
				if isCSquare(x, y) {
					cornerX, cornerY := adjacentCornerC(x, y)
					cornerCell := g.board[cornerX][cornerY]
					if cornerCell != cell {
						value = -abs(value)
					}
				}

				// Count discs and heuristic values
				if cell == player {
					myHeuristic += value
					myDiscs++

					// Check for frontier discs
					if isFrontierDisc(g.board, x, y) {
						myFrontierDiscs++
					}
				} else if cell == opponent {
					opponentDiscs++

					// Check for frontier discs
					if isFrontierDisc(g.board, x, y) {
						opponentFrontierDiscs++
					}
				}
			}
		}
	}

	// Corner ownership
	for _, corner := range corners {
		x, y := corner[0], corner[1]
		cell := g.board[x][y]

		if cell == player {
			myCorners++
		} else if cell == opponent {
			opponentCorners++
		}
	}

	// Edge stability
	myEdgeStability := 0
	opponentEdgeStability := 0
	edges := [][][2]int{
		// Top edge
		{{0, 0}, {1, 0}, {2, 0}, {3, 0}, {4, 0}, {5, 0}, {6, 0}, {7, 0}},
		// Bottom edge
		{{0, 7}, {1, 7}, {2, 7}, {3, 7}, {4, 7}, {5, 7}, {6, 7}, {7, 7}},
		// Left edge
		{{0, 0}, {0, 1}, {0, 2}, {0, 3}, {0, 4}, {0, 5}, {0, 6}, {0, 7}},
		// Right edge
		{{7, 0}, {7, 1}, {7, 2}, {7, 3}, {7, 4}, {7, 5}, {7, 6}, {7, 7}},
	}

	for _, edge := range edges {
		myEdgeCount := 0
		opponentEdgeCount := 0
		for _, pos := range edge {
			x, y := pos[0], pos[1]
			cell := g.board[x][y]

			if cell == player {
				myEdgeCount++
			} else if cell == opponent {
				opponentEdgeCount++
			}
		}
		if myEdgeCount == 8 {
			myEdgeStability += 1
		} else if opponentEdgeCount == 8 {
			opponentEdgeStability += 1
		}
	}

	// Mobility
	myMobility := len(g.ValidMoves(player))
	opponentMobility := len(g.ValidMoves(opponent))

	// Heuristic
	components.Heuristic = float64(myHeuristic)

	// Disc difference
	discSum := float64(myDiscs + opponentDiscs)

	if discSum != 0 {
		components.DiscDiff = 100.0 * float64(myDiscs-opponentDiscs) / discSum
	}

	// Mobility difference
	mobilitySum := float64(myMobility + opponentMobility)

	if mobilitySum != 0 {
		components.Mobility = 100.0 * float64(myMobility-opponentMobility) / mobilitySum
	}

	// Frontier discs
	frontierSum := float64(myFrontierDiscs + opponentFrontierDiscs)

	if frontierSum != 0 {
		components.Frontier = -100.0 * float64(myFrontierDiscs-opponentFrontierDiscs) / frontierSum
	}

	// Potential mobility
	potentialMobilitySum := float64(myPotentialMobility + opponentPotentialMobility)

	if potentialMobilitySum != 0 {
		components.PotentialMobility = 100.0 * float64(myPotentialMobility-opponentPotentialMobility) / potentialMobilitySum
	}

	// Corner ownership
	cornerSum := float64(myCorners + opponentCorners)

	if cornerSum != 0 {
		components.CornerOwnership = 100.0 * float64(myCorners-opponentCorners) / cornerSum
	}

	// Edge stability
	edgeStabilitySum := float64(myEdgeStability + opponentEdgeStability)

	if edgeStabilitySum != 0 {
		components.EdgeStability = 100.0 * float64(myEdgeStability-opponentEdgeStability) / edgeStabilitySum
	}

	// Total score
	components.TotalScore =
		(components.WeightHeuristic * components.Heuristic) +
			(components.WeightDiscDifference * components.DiscDiff) +
			(components.WeightMobility * components.Mobility) +
			(components.WeightFrontier * components.Frontier) +
			(components.WeightPotentialMob * components.PotentialMobility) +
			(components.WeightCorner * components.CornerOwnership) +
			(components.WeightEdge * components.EdgeStability)

	return components
}

// Helper functions

// isXSquare checks if a position is an X-square
func isXSquare(x, y int) bool {
	return (x == 1 && y == 1) || (x == 1 && y == 6) ||
		(x == 6 && y == 1) || (x == 6 && y == 6)
}

// isCSquare checks if a position is a C-square
func isCSquare(x, y int) bool {
	return (x == 0 && y == 1) || (x == 1 && y == 0) ||
		(x == 0 && y == 6) || (x == 1 && y == 7) ||
		(x == 6 && y == 0) || (x == 7 && y == 1) ||
		(x == 6 && y == 7) || (x == 7 && y == 6)
}

// adjacentCorner returns the corner position adjacent to an X-square
func adjacentCorner(x, y int) (int, int) {
	switch {
	case x == 1 && y == 1:
		return 0, 0
	case x == 1 && y == 6:
		return 0, 7
	case x == 6 && y == 1:
		return 7, 0
	case x == 6 && y == 6:
		return 7, 7
	default:
		return -1, -1 // Not an X-square
	}
}

// adjacentCornerC returns the corner position adjacent to a C-square
func adjacentCornerC(x, y int) (int, int) {
	switch {
	case x == 0 && y == 1:
		return 0, 0
	case x == 1 && y == 0:
		return 0, 0
	case x == 0 && y == 6:
		return 0, 7
	case x == 1 && y == 7:
		return 0, 7
	case x == 6 && y == 0:
		return 7, 0
	case x == 7 && y == 1:
		return 7, 0
	case x == 6 && y == 7:
		return 7, 7
	case x == 7 && y == 6:
		return 7, 7
	default:
		return -1, -1 // Not a C-square
	}
}

// isFrontierDisc checks if a disc is a frontier disc
func isFrontierDisc(board *Board, x, y int) bool {
	for _, dir := range directions {
		nx, ny := x+dir.x, y+dir.y
		if nx >= 0 && nx < BoardSize && ny >= 0 && ny < BoardSize {
			if board[nx][ny] == Blank {
				return true
			}
		}
	}
	return false
}

// Helper function to get absolute value
func abs(a int) int {
	if a < 0 {
		return -a
	}

	return a
}

// Copy creates a deep copy of the game state
func (g *Game) Copy() *Game {
	return &Game{
		board:      g.board.Copy(),
		current:    g.current,
		moveCount:  g.moveCount,
		difficulty: g.difficulty,
	}
}

// Reset resets the game state to the initial state
func (g *Game) Reset() {
	g.board = NewBoard()
	g.current = Black
	g.moveCount = 4
}

// GetScore returns the score of the game
func (g *Game) GetScore() (int, int) {
	blackCount, whiteCount := 0, 0

	for x := 0; x < BoardSize; x++ {
		for y := 0; y < BoardSize; y++ {
			if g.board[x][y] == Black {
				blackCount++
			} else if g.board[x][y] == White {
				whiteCount++
			}
		}
	}

	return blackCount, whiteCount
}
