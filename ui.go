package main

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func (g *Game) StartUI() {
	app := tview.NewApplication()

	// Variables to store selected options
	var playerColorOption string
	var difficultyOption string
	var showValidMoves = true

	// Start with the start screen
	var showStartScreen func()
	var startGame func()

	showStartScreen = func() {
		form := tview.NewForm()
		form.
			AddDropDown("Choose your color", []string{"Black", "White"}, 0, func(option string, index int) {
				playerColorOption = option
			}).
			AddDropDown("Difficulty", []string{"Easy", "Medium", "Hard", "Brutal", "Extreme"}, 1, func(option string, index int) {
				difficultyOption = option
			}).
			AddCheckbox("Show valid moves", true, func(checked bool) {
				showValidMoves = checked
			}).
			AddButton("Start Game", func() {
				// Set AI flags based on player color
				if playerColorOption == "Black" {
					g.blackAI = false
					g.whiteAI = true
				} else {
					g.blackAI = true
					g.whiteAI = false
				}

				// Set difficulty (assuming difficulty levels map to some settings)
				switch difficultyOption {
				case "Easy":
					g.difficulty = 2
				case "Medium":
					g.difficulty = 3
				case "Hard":
					g.difficulty = 5
				case "Brutal":
					g.difficulty = 7
				case "Extreme":
					g.difficulty = 9
				default:
					g.difficulty = 3 // Default to Medium
				}

				startGame()
			}).
			AddButton("Quit", func() {
				app.Stop()
			})
		form.SetBorder(true).SetTitle("Reversi").SetTitleAlign(tview.AlignCenter)

		app.SetRoot(form, true).SetFocus(form)
	}

	// Now define startGame, which will set up the board and start the game
	startGame = func() {
		g.Reset()

		boardTable := tview.NewTable()

		boardTable.SetSelectable(true, true)
		boardTable.SetBorder(true)
		boardTable.SetTitleAlign(tview.AlignLeft)
		boardTable.SetTitleColor(tcell.ColorGreen)
		boardTable.SetBorderColor(tcell.ColorGreen)
		boardTable.SetBorders(true)

		// Create a new TextView to display the score components
		scoreBox := tview.NewTextView()
		scoreBox.SetBorder(true)
		scoreBox.SetTitle("Score")

		// Create a Flex layout to arrange board and scoreBox side by side
		flex := tview.NewFlex().
			AddItem(boardTable, 0, 1, true).
			AddItem(scoreBox, 60, 1, false)

		updateBoard := func() {
			for y := 0; y < BoardSize; y++ {
				for x := 0; x < BoardSize; x++ {
					symbol := getPieceSymbol(g.board[x][y])

					cell := tview.NewTableCell(symbol)
					cell.SetAlign(tview.AlignCenter)

					boardTable.SetCell(y, x, cell)

					if g.board[x][y] == Blank && showValidMoves {
						if flips := g.Flips(x, y, g.current); len(flips) > 0 {
							// Highlight valid moves
							validCell := tview.NewTableCell("· ")
							validCell.SetAlign(tview.AlignCenter)
							validCell.SetTextColor(tcell.ColorGreen)
							boardTable.SetCell(y, x, validCell)
						}
					}
				}
			}

			// Update the title with the current player
			boardTable.SetTitle(fmt.Sprintf(" Reversi - %s's turn ", g.PlayerName(g.current)))

			// Update the status box with the current score
			blackScore, whiteScore := g.GetScore()
			scoreText := fmt.Sprintf("Black: %d\nWhite: %d", blackScore, whiteScore)
			scoreBox.SetText(scoreText)
		}

		updateBoard()

		var (
			AIThinking   int32 // Atomic boolean for AI thinking status
			spinnerIndex int
			spinners     = []string{"|", "/", "-", "\\"}
		)

		// Function to handle turns
		var processNextTurn func()

		processNextTurn = func() {
			// Check if the game is over
			if g.IsGameOver() {
				winner := g.PlayerName(g.GetWinner())

				// Create the ASCII art
				asciiArt :=
`⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢀⣀⣠⣴⣶⣶⣶⣶⣾⣿⣿⣶⣶⣶⣤⣀⣀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣀⣤⣶⣿⣿⡿⠿⠛⠛⠋⠉⠉⠉⠉⠉⠉⠛⠛⠻⢿⣿⣿⣷⣦⣄⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣀⣴⣿⣿⠿⠛⠉⠁⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠉⠙⠻⣿⣿⣶⣄⡀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣠⣾⣿⡿⠋⠁⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠙⠻⣿⣿⣦⡀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣠⣾⣿⠟⠁⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠈⠻⣿⣿⣦⡀⠀⢀⣠⣤⣶⡄⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⢀⣼⣿⡿⠁⠀⠀⠀⠀⠀⠀⠀⠀⠀⣀⣤⡶⠶⠛⠛⠋⠉⠉⠛⠛⠳⠶⢤⣄⡀⠀⠀⠀⠀⠀⠀⠀⣀⣨⣿⣿⣿⡿⠿⠛⠛⢿⣿⡀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⢠⣾⣿⠟⠀⠀⠀⠀⠀⠀⠀⠀⣀⣴⠞⠋⠁⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠉⠛⢶⣄⣠⣤⣶⣶⡿⠿⠟⠛⠉⠁⢀⣀⠀⠀⠘⣿⣇⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⢠⣿⣿⠏⠀⠀⠀⠀⠀⠀⠀⣠⡾⠛⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣀⣠⣴⣶⣾⣿⠿⠟⠋⠉⠀⢀⣀⣴⡴⣾⣿⣿⣿⣷⡄⠀⢹⣿⡄⠀⠀⠀
⠀⠀⠀⠀⠀⠀⢀⣾⣿⠇⠀⠀⠀⠀⠀⠀⠀⣴⠟⠁⠀⠀⠀⠀⠀⠀⠀⠀⣀⣤⣤⣶⣾⣿⠿⠛⠋⠉⠀⠀⠀⢠⣶⣾⣿⣿⠿⠟⠃⢹⣿⡄⠈⣿⣧⠀⠀⣿⣧⠀⠀⠀
⠀⠀⠀⠀⠀⠀⣼⣿⡏⠀⠀⠀⠀⠀⠀⠀⣼⠋⠀⠀⠀⢀⣀⣤⣤⣶⣾⡿⠿⠛⠛⠉⠁⢀⣤⣤⣄⠸⣿⣆⠀⠸⣿⡇⢿⣧⠀⢀⣠⠈⣿⣷⣶⣿⡏⠀⠀⠸⣿⣇⠀⠀
⠀⠀⠀⠀⠀⢠⣿⣿⠁⠀⠀⠀⠀⠀⠀⣸⣏⣠⣤⣶⣿⡿⠿⠟⠛⠉⠀⠀⠀⠀⠀⠀⠀⣾⣿⠻⣿⣧⠹⣿⣆⠀⣿⡇⠸⣿⣿⣿⠿⠇⠸⣿⡏⢻⣿⣆⠀⠀⢿⣿⡀⠀
⠀⠀⠀⠀⠀⢸⣿⡟⠀⠀⣀⣠⣴⣶⣿⣿⠿⠟⠋⠉⠀⢀⣠⣤⣶⣾⣧⠀⠀⠀⠀⠀⠀⣿⣿⠀⠈⣿⣧⠘⣿⣆⣿⣿⠀⢻⣿⡄⠀⠀⣀⢻⣿⡀⠙⣿⣷⠄⠘⣿⣧⠀
⠀⠀⠀⢀⣀⣾⣿⣷⣾⡿⠿⠛⠋⠉⠀⠀⣤⣤⠀⢸⣿⡌⣿⣿⠋⠉⠁⠀⠀⠀⠀⠀⠀⠸⣿⣇⠀⢸⣿⡇⠘⣿⣾⣿⠀⠈⣿⣷⣾⣿⡿⠮⠛⠃⠀⢀⣀⣠⣤⣾⣿⠄
⣴⣶⣾⡿⠿⠛⠛⠉⠁⠀⠀⠀⣶⣿⣆⠀⢹⣿⣧⣸⣿⣧⠸⣿⣦⣤⣶⣆⠀⠀⠀⠀⠀⠀⢻⣿⣄⢀⣿⡇⠀⠘⣿⣿⡆⠀⠘⠛⠉⠁⣀⣠⣤⣶⣾⣿⠿⠟⠛⠉⠁⠀
⢹⣿⣇⠀⠀⠀⣴⣿⣿⣷⡄⠀⢹⡿⣿⣆⠈⣿⣿⣿⣿⢿⣆⢻⣿⠟⠋⠉⠀⠀⠀⠀⠀⠀⠀⠻⣿⣿⡿⠃⠀⠀⠀⠀⣀⣠⣴⣶⣿⣿⠿⠟⠋⢩⣿⣿⠀⠀⠀⠀⠀⠀
⠀⢿⣿⡀⠀⠀⣿⣟⠈⠛⠋⠀⢸⣿⠈⣿⣆⠸⣿⣻⣿⡾⣿⡌⣿⣇⣀⣤⣴⡆⠀⠀⠀⠀⠀⠀⠀⠀⣀⣤⣤⣶⣾⣿⠿⣿⠛⠉⠁⠀⠀⠀⠀⢸⣿⡟⠀⠀⠀⠀⠀⠀
⠀⠸⣿⣧⠀⠀⢻⣿⠀⣶⣾⣧⠘⣿⣷⣾⣿⣆⢿⣧⠉⠁⢻⣧⢹⣿⡿⠿⠛⠃⠀⢀⣀⣤⣴⣶⡾⡿⠟⠋⠛⠉⠁⠀⣰⠏⠀⠀⠀⠀⠀⠀⠀⣿⣿⠇⠀⠀⠀⠀⠀⠀
⠀⠀⢹⣿⡄⠀⠈⣿⣧⠙⠙⣿⡆⣿⣟⠉⠘⣿⣾⣿⡆⠀⠀⠛⠁⢀⣀⣠⣤⣶⣿⡿⠿⠛⠋⠁⠀⠀⠀⠀⠀⠀⠀⣴⠟⠀⠀⠀⠀⠀⠀⠀⣸⣿⡏⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⣿⣷⠀⠀⠘⣿⣷⣴⣿⡏⣿⡿⠀⠀⠈⠉⢀⣀⣠⣴⣶⣿⡿⠿⠛⠋⠁⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣠⡾⠋⠀⠀⠀⠀⠀⠀⠀⣰⣿⡟⠀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠸⣿⡆⠀⠀⠈⠛⠛⠋⠀⢀⣀⣤⣴⣶⣿⡿⠿⣿⣋⠉⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢀⣠⡾⠋⠀⠀⠀⠀⠀⠀⠀⠀⣼⣿⡿⠁⠀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⢻⣿⡄⢀⣀⣤⣴⣶⣿⠿⠿⠛⠋⠉⠀⠀⠀⠈⠙⠳⢦⣤⣀⣀⠀⠀⠀⠀⠀⢀⣀⣠⣤⠶⠛⠉⠀⠀⠀⠀⠀⠀⠀⠀⢀⣼⣿⡿⠁⠀⠀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠈⣿⣿⡿⠿⠛⠙⢿⣿⣷⣄⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠉⠉⠙⠛⠛⠛⠛⠉⠉⠁⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢀⣴⣿⣿⠏⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠙⢿⣿⣷⣄⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢀⣴⣿⣿⠟⠁⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠙⠿⣿⣷⣦⣄⡀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣀⣤⣾⣿⡿⠛⠁⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠈⠛⠿⢿⣿⣷⣦⣄⣀⡀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣀⣠⣤⣶⣿⣿⡿⠟⠉⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠈⠙⠻⠿⣿⣿⣿⣿⣷⣶⣶⣶⣾⣿⣿⣿⣿⡿⠿⠛⠉⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀ ⠀⠉⠉⠉⠉⠉⠉⠉⠉⠉⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀`

				blackScore, whiteScore := g.GetScore()

				modal := tview.NewModal().
SetText(fmt.Sprintf("%s\nGame Over!\n%s wins!\nWhite score: %d\nBlack score: %d", asciiArt, winner, whiteScore, blackScore)).
					AddButtons([]string{"New Game", "Quit"}).
					SetDoneFunc(func(buttonIndex int, buttonLabel string) {
						if buttonLabel == "New Game" {
							showStartScreen()
						} else {
							app.Stop()
						}
					})

				app.SetRoot(modal, false).SetFocus(modal)

				return
			}

			// Check if current player has any valid moves
			if len(g.ValidMoves(g.current)) == 0 {
				// Current player has no valid moves
				g.SwitchTurn()
				updateBoard()
				// Process next turn
				processNextTurn()

				return
			}

			if (g.current == Black && g.blackAI) || (g.current == White && g.whiteAI) {
				// AI's turn
				atomic.StoreInt32(&AIThinking, 1)
				spinnerIndex = 0

				// Start the spinner goroutine
				ticker := time.NewTicker(100 * time.Millisecond)
				go func() {
					for {
						select {
						case <-ticker.C:
							if atomic.LoadInt32(&AIThinking) == 0 {
								ticker.Stop()

								return
							}
							spinner := spinners[spinnerIndex%len(spinners)]
							spinnerIndex++
							app.QueueUpdateDraw(func() {
								boardTable.SetTitle(fmt.Sprintf(" Reversi - %s's turn %s ", g.PlayerName(g.current), spinner))
							})
						}
					}
				}()

				// Start the AI move in a goroutine
				go func() {
					g.AIMove()

					atomic.StoreInt32(&AIThinking, 0)

					app.QueueUpdateDraw(func() {
						updateBoard()
						boardTable.SetTitle(fmt.Sprintf(" Reversi - %s's turn ", g.PlayerName(g.current)))
						// After the AI move, process the next turn
						processNextTurn()
					})
				}()
			} else {
				// Human's turn
				updateBoard()
			}
		}

		boardTable.SetSelectedFunc(func(row, column int) {
			// Block input if AI is thinking
			if atomic.LoadInt32(&AIThinking) == 1 {
				return
			}

			if g.board[column][row] != Blank {
				return
			}

			if flips := g.Flips(column, row, g.current); len(flips) > 0 {
				g.MakeMove(Move{X: column, Y: row, Flips: flips}, true)
				updateBoard()

				// Process the next turn
				processNextTurn()
			} else {
				// Invalid move
				return
			}
		})

		if g.current == Black && g.blackAI {
			// If it's AI's turn, start the AI move
			processNextTurn()
		}

		app.SetRoot(flex, true)
	}

	showStartScreen()

	if err := app.Run(); err != nil {
		panic(err)
	}
}

func getPieceSymbol(piece int) string {
	switch piece {
	case Black:
		return " ⚫ "
	case White:
		return " ⚪ "
	default:
		return "    "
	}
}
