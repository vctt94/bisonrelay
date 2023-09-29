package main

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// escapeNick returns s escaped from chars that don't don't belong in a nick.
func escapeNick(s string) string {
	return strings.Map(func(r rune) rune {
		if !strconv.IsPrint(r) {
			return -1
		}
		if r == utf8.RuneError {
			return -1
		}
		return r
	}, s)
}

// escapeContent returns s escaped from chars that don't belong in content.
func escapeContent(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return r
		}
		if !strconv.IsGraphic(r) {
			return -1
		}
		if r == utf8.RuneError {
			return -1
		}
		return r
	}, s)
}

func formatGameState(game *PokerGame) string {
	return fmt.Sprintf("\n|---------------\n"+
		"Current Stage: %s\n"+
		"Community Cards: %v\n"+
		"Pot: %f\n"+
		"Current Player: %s\n"+
		"---------------|\n",
		game.CurrentStage, game.CommunityCards, game.Pot, game.Players[game.CurrentPlayer].Nick)
}
