package main

import (
	"math/rand"
)

var zobristTable [BoardSize][BoardSize][3]uint64
var zobristTurn uint64

func initZobrist() {
	for x := 0; x < BoardSize; x++ {
		for y := 0; y < BoardSize; y++ {
			for k := 0; k < 3; k++ {
				zobristTable[x][y][k] = rand.Uint64()
			}
		}
	}

	zobristTurn = rand.Uint64()
}

func (g *Game) computeZobristHash() uint64 {
	var h uint64

	for x := 0; x < BoardSize; x++ {
		for y := 0; y < BoardSize; y++ {
			piece := g.board[x][y]
			h ^= zobristTable[x][y][piece]
		}
	}

	if g.current == Black {
		h ^= zobristTurn
	}

	return h
}
