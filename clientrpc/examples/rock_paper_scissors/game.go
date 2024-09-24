package main

import (
	"fmt"
	"log"
	"time"

	"github.com/companyzero/bisonrelay/clientrpc/types"
)

func (s *pluginServer) areTwoPlayersReady() bool {
	readyCount := 0
	for _, player := range s.players {
		if player.Ready {
			readyCount++
		}
	}

	return readyCount == 2
}

func (s *pluginServer) startGame() {
	log.Printf("Both players are ready. Starting the game loop!")

	// Notify both players that the game is starting
	for _, player := range s.players {
		if player.Ready {
			player.notifier.Send(&types.PluginStartStreamResponse{
				Message: "Both players are ready. The game has started! Send your move.",
			})
		}
	}

	// Start a loop to continuously check for both players' moves
	for {
		time.Sleep(1 * time.Second) // Sleep for a short period to avoid busy waiting

		s.gameLock.Lock()

		// Check if both players have made their moves
		if s.bothPlayersMoved() {
			// Compare moves and notify the players
			s.compareMoves()
			s.gameLock.Unlock()
			break // Break the loop after processing the result
		}

		s.gameLock.Unlock()
	}
}

// Helper to convert move to string
func (s *pluginServer) moveName(move string) string {
	switch move {
	case Rock:
		return "Rock"
	case Paper:
		return "Paper"
	case Scissors:
		return "Scissors"
	default:
		return "Unknown"
	}
}

func (s *pluginServer) bothPlayersMoved() bool {
	count := 0
	for _, player := range s.players {
		if player.Move != "" {
			count++
		}
	}
	return count == 2
}

// Compare moves of the two players and notify them of the result
func (s *pluginServer) compareMoves() {
	var player1, player2 *Player
	for _, player := range s.players {
		if player1 == nil {
			player1 = &player
		} else {
			player2 = &player
		}
	}

	// Determine the winner
	result := s.determineWinner(player1, player2)

	// Notify both players of the result
	player1.stream.Send(&types.PluginCallActionStreamResponse{
		Response: []byte(result),
	})
	player2.stream.Send(&types.PluginCallActionStreamResponse{
		Response: []byte(result),
	})

	// Reset moves for the next round
	player1.Move = ""
	player2.Move = ""
	s.players[player1.ID] = *player1
	s.players[player2.ID] = *player2
}

// Determine the winner between two players
func (s *pluginServer) determineWinner(player1, player2 *Player) string {
	if player1.Move == player2.Move {
		return fmt.Sprintf("It's a tie! Both players chose %s", s.moveName(player1.Move))
	}

	if (player1.Move == Rock && player2.Move == Scissors) ||
		(player1.Move == Paper && player2.Move == Rock) ||
		(player1.Move == Scissors && player2.Move == Paper) {
		return fmt.Sprintf("Player 1 (%s) wins! %s beats %s", player1.ID, s.moveName(player1.Move), s.moveName(player2.Move))
	}

	return fmt.Sprintf("Player 2 (%s) wins! %s beats %s", player2.ID, s.moveName(player2.Move), s.moveName(player1.Move))
}
