package main

import (
	"math"
	"sort"
	"sync"
)

const (
	Exact = iota // Transposition table flags
	LowerBound
	UpperBound
)

type Move struct {
	X, Y  int
	Flips [][2]int
}

// TTEntry represents an entry in the transposition table
type TTEntry struct {
	Depth    int
	Eval     float64
	Flag     int // Exact, LowerBound, UpperBound
	BestMove Move
}

var transpositionTable map[uint64]TTEntry
var ttMutex sync.RWMutex

// MoveKey is a comparable type for use as map keys
type MoveKey struct {
	X, Y int
}

var historyTable map[MoveKey]int
var killerMoves []MoveKey

func (g *Game) AIMove() {
	moves := g.ValidMoves(g.current)

	if len(moves) == 0 {
		g.SwitchTurn()

		return
	}

	aiPlayer := g.current

	// Initialize Zobrist hashing and transposition table
	initZobrist()
	transpositionTable = make(map[uint64]TTEntry)
	historyTable = make(map[MoveKey]int)
	killerMoves = make([]MoveKey, g.difficulty)

	// Check for endgame solver activation
	emptySquares := g.CountEmptySquares()

	if emptySquares <= 12 {
		bestMove := g.EndgameSolver(aiPlayer)
		g.MakeMove(bestMove, true)

		return
	}

	// Iterative deepening with time management
	var bestMove Move

	for depth := 1; ; depth++ {
		var currentBestMove Move
		var currentBestScore float64 = math.Inf(-1)

		// Run minimax at current depth
		for _, move := range moves {
			newGame := g.SimulateMove(move, true)
			score := minimax(newGame, depth-1, math.Inf(-1), math.Inf(1), false, aiPlayer)

			if depth == g.difficulty {
				break
			}

			if score > currentBestScore {
				currentBestScore = score
				currentBestMove = move
			}
		}

		if depth == g.difficulty {
			break
		}

		bestMove = currentBestMove
	}

	g.MakeMove(bestMove, true)
}

func minimax(game *Game, depth int, alpha, beta float64, maximizing bool, aiPlayer int) float64 {
	if depth == game.difficulty {
		return 0 // Return neutral value if maximum depth is reached
	}

	hashKey := game.computeZobristHash()

	// Transposition table lookup
	ttMutex.RLock()

	if entry, found := transpositionTable[hashKey]; found && entry.Depth >= depth {
		ttMutex.RUnlock()

		switch entry.Flag {
		case Exact:
			return entry.Eval
		case LowerBound:
			if entry.Eval > alpha {
				alpha = entry.Eval
			}
		case UpperBound:
			if entry.Eval < beta {
				beta = entry.Eval
			}
		}

		if alpha >= beta {
			return entry.Eval
		}
	} else {
		ttMutex.RUnlock()
	}

	if depth == 0 || game.IsGameOver() {
		eval := game.Evaluate(aiPlayer) // Do a final evaluation
		ttMutex.Lock()
		transpositionTable[hashKey] = TTEntry{Depth: depth, Eval: eval, Flag: Exact}
		ttMutex.Unlock()

		return eval
	}

	moves := game.ValidMoves(game.current)

	if len(moves) == 0 {
		game.SwitchTurn()
		eval := minimax(game, depth-1, alpha, beta, !maximizing, aiPlayer)
		game.SwitchTurn()

		return eval
	}

	orderMoves(game, moves, depth, aiPlayer)

	var value float64
	var bestMove Move
	alphaOrig := alpha

	if maximizing {
		value = math.Inf(-1)

		for _, move := range moves {
			if depth == game.difficulty {
				break
			}

			newGame := game.SimulateMove(move, true)
			eval := minimax(newGame, depth-1, alpha, beta, false, aiPlayer)

			if eval > value {
				value = eval
				bestMove = move
			}

			alpha = math.Max(alpha, value)

			if alpha >= beta {
				// Beta cutoff
				moveKey := MoveKey{X: move.X, Y: move.Y}
				historyTable[moveKey] += depth * depth
				killerMoves[depth%game.difficulty] = moveKey

				break
			}
		}
	} else {
		value = math.Inf(1)

		for _, move := range moves {
			if depth == game.difficulty {
				break
			}

			newGame := game.SimulateMove(move, true)
			eval := minimax(newGame, depth-1, alpha, beta, true, aiPlayer)

			if eval < value {
				value = eval
				bestMove = move
			}

			beta = math.Min(beta, value)

			if alpha >= beta {
				// Alpha cutoff
				moveKey := MoveKey{X: move.X, Y: move.Y}
				historyTable[moveKey] += depth * depth
				killerMoves[depth%game.difficulty] = moveKey

				break
			}
		}
	}

	// Store in transposition table
	var flag int

	if value <= alphaOrig {
		flag = UpperBound
	} else if value >= beta {
		flag = LowerBound
	} else {
		flag = Exact
	}

	ttMutex.Lock()
	transpositionTable[hashKey] = TTEntry{Depth: depth, Eval: value, Flag: flag, BestMove: bestMove}
	ttMutex.Unlock()

	return value
}

func orderMoves(game *Game, moves []Move, depth int, aiPlayer int) {
	type MoveEval struct {
		move    Move
		moveKey MoveKey
		eval    float64
	}

	moveEvals := make([]MoveEval, len(moves))

	for i, move := range moves {
		newGame := game.SimulateMove(move, false)
		eval := newGame.Evaluate(aiPlayer)
		moveKey := MoveKey{X: move.X, Y: move.Y}
		moveEvals[i] = MoveEval{
			move:    move,
			moveKey: moveKey,
			eval:    eval,
		}
	}

	// Prioritize moves based on safety and evaluation score
	sort.Slice(moveEvals, func(i, j int) bool {
		// Killer move priority
		killerMoveKey := killerMoves[depth%game.difficulty]

		if moveEvals[i].moveKey == killerMoveKey {
			return true
		}

		if moveEvals[j].moveKey == killerMoveKey {
			return false
		}

		// History heuristic
		hi := historyTable[moveEvals[i].moveKey]
		hj := historyTable[moveEvals[j].moveKey]

		if hi != hj {
			return hi > hj
		}

		return moveEvals[i].eval > moveEvals[j].eval
	})

	for i, me := range moveEvals {
		moves[i] = me.move
	}
}

func isXSquareMove(x, y int) bool {
	xSquares := [][2]int{{1, 1}, {1, BoardSize - 2}, {BoardSize - 2, 1}, {BoardSize - 2, BoardSize - 2}}

	for _, xSquare := range xSquares {
		if x == xSquare[0] && y == xSquare[1] {
			return true
		}
	}

	return false
}

func isEdgePosition(x, y int) bool {
	if x == 0 || x == BoardSize-1 || y == 0 || y == BoardSize-1 {
		// Check if it's not a corner
		if !((x == 0 && y == 0) || (x == 0 && y == BoardSize-1) || (x == BoardSize-1 && y == 0) || (x == BoardSize-1 && y == BoardSize-1)) {
			return true
		}
	}

	return false
}

func (g *Game) EndgameSolver(aiPlayer int) Move {
	// Perform exhaustive search to the end of the game
	moves := g.ValidMoves(g.current)
	if len(moves) == 0 {
		g.SwitchTurn()

		return g.EndgameSolver(aiPlayer)
	}

	var bestMove Move
	var bestScore float64 = math.Inf(-1)

	for _, move := range moves {
		newGame := g.SimulateMove(move, true)
		score := minimaxEndgame(newGame, math.Inf(-1), math.Inf(1), false, aiPlayer)

		if score > bestScore {
			bestScore = score
			bestMove = move
		}
	}

	return bestMove
}

func minimaxEndgame(game *Game, alpha, beta float64, maximizing bool, aiPlayer int) float64 {
	if game.IsGameOver() {
		return game.EvaluateEndgame(aiPlayer)
	}

	moves := game.ValidMoves(game.current)

	if len(moves) == 0 {
		game.SwitchTurn()
		eval := minimaxEndgame(game, alpha, beta, !maximizing, aiPlayer)
		game.SwitchTurn()

		return eval
	}

	if maximizing {
		value := math.Inf(-1)

		for _, move := range moves {
			newGame := game.SimulateMove(move, true)
			eval := minimaxEndgame(newGame, alpha, beta, false, aiPlayer)
			value = math.Max(value, eval)
			alpha = math.Max(alpha, value)

			if alpha >= beta {
				break
			}
		}

		return value
	} else {
		value := math.Inf(1)

		for _, move := range moves {
			newGame := game.SimulateMove(move, true)
			eval := minimaxEndgame(newGame, alpha, beta, true, aiPlayer)
			value = math.Min(value, eval)
			beta = math.Min(beta, value)

			if alpha >= beta {
				break
			}
		}

		return value
	}
}

func (g *Game) EvaluateEndgame(aiPlayer int) float64 {
	// If the game is over, evaluate based on the number of pieces and return a high score for winning, low score for losing and 0 for draw
	winner := g.GetWinner()

	if winner == aiPlayer {
		return 100
	} else if winner == Opponent(aiPlayer) {
		return -100
	} else {
		return 0
	}
}

func (g *Game) CountEmptySquares() int {
	count := 0
	for x := 0; x < BoardSize; x++ {
		for y := 0; y < BoardSize; y++ {
			if g.board[x][y] == Blank {
				count++
			}
		}
	}

	return count
}

func (g *Game) IsGameOver() bool {
	return len(g.ValidMoves(Black)) == 0 && len(g.ValidMoves(White)) == 0
}
