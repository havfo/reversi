// board.go
package main

type Board [BoardSize][BoardSize]int

// NewBoard initializes the board with starting positions
func NewBoard() *Board {
	b := &Board{}
	mid := BoardSize / 2
	b[mid-1][mid-1], b[mid][mid] = White, White
	b[mid-1][mid], b[mid][mid-1] = Black, Black

	return b
}

// Copy creates a deep copy of the board (used in AI to simulate moves)
func (b *Board) Copy() *Board {
	newBoard := *b

	return &newBoard
}

func (g *Game) GetWinner() int {
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
	
	if blackCount > whiteCount {
		return Black
	} else if whiteCount > blackCount {
		return White
	}

	return Blank
}

func (g *Game) PlayerName(player int) string {
	if player == Black {
		return "Black"
	} else if player == White {
		return "White"
	}

	return "Unknown"
}
