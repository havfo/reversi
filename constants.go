package main

const (
	Blank     = 0
	Black     = 1
	White     = 2
	BoardSize = 8
)

// Directions for checking valid moves
var directions = []struct{ x, y int }{
	{-1, -1}, {-1, 0}, {-1, 1},
	{0, -1},           {0, 1},
	{1, -1},  {1, 0},  {1, 1},
}
