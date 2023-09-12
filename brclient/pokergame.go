package main

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"github.com/companyzero/bisonrelay/clientrpc/types"
	"github.com/companyzero/bisonrelay/rpc"
)

var (
	suits  = []string{"Hearts", "Diamonds", "Clubs", "Spades"}
	values = []string{"2", "3", "4", "5", "6", "7", "8", "9", "10", "J", "Q", "K", "A"}
)

type PokerGame struct {
	Players        []rpc.Player
	CommunityCards []rpc.Card
	Pot            int
	CurrentStage   string
	DealerPosition int
	BigBlind       int
	SmallBlind     int
	Deck           []rpc.Card
	BB             float64
	SB             float64
	Button         *rpc.Player
	// CurrentPlayer represents an index of the Players array for the current
	// player to play at the current stage.
	CurrentPlayer int
	Bot           string
}

type Deck struct {
	Cards []rpc.Card
}

func (g *PokerGame) Draw() rpc.Card {
	card := g.Deck[len(g.Deck)-1]
	g.Deck = g.Deck[:len(g.Deck)-1]
	return card
}

func (g *PokerGame) ShuffleDeck() {
	if g.Deck == nil {
		var cards []rpc.Card
		for _, suit := range suits {
			for _, value := range values {
				cards = append(cards, rpc.Card{Suit: suit, Value: value})
			}
		}
		g.Deck = cards
	}
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(g.Deck), func(i, j int) { g.Deck[i], g.Deck[j] = g.Deck[j], g.Deck[i] })
}

func (g *PokerGame) Deal() {
	// Deal two cards to each player
	for _, player := range g.Players {
		player.Hand = append(player.Hand, g.Deck[len(g.Deck)-2], g.Deck[len(g.Deck)-1])
		g.Deck = g.Deck[:len(g.Deck)-2]
	}
}

func (g *PokerGame) Flop() {
	// Deal the flop
	g.CommunityCards = append(g.CommunityCards, g.Deck[len(g.Deck)-3], g.Deck[len(g.Deck)-2], g.Deck[len(g.Deck)-1])
	g.Deck = g.Deck[:len(g.Deck)-3]
}

func (g *PokerGame) Turn() {
	card := g.Deck[0]
	g.CommunityCards = append(g.CommunityCards, card)
	g.Deck = g.Deck[1:]
}

func (g *PokerGame) River() {
	card := g.Deck[0]
	g.CommunityCards = append(g.CommunityCards, card)
	g.Deck = g.Deck[1:]
}

func (g *PokerGame) ProgressDealer() {
	g.DealerPosition = (g.DealerPosition + 1) % len(g.Players)
}

func (g *PokerGame) ProgressBlinds(ctx context.Context, payment types.PaymentsServiceClient) error {
	g.ProgressDealer()

	// smallBlindPosition := (g.DealerPosition + 1) % len(g.Players)
	// bigBlindPosition := g.DealerPosition

	// Pay the small blind
	paymentReq := &types.TipUserRequest{
		DcrAmount:   g.SB,
		User:        g.Bot,
		MaxAttempts: 3,
	}
	fmt.Printf("paymentReq: %+v\n\n", paymentReq)
	paymentResp := &types.TipUserResponse{}
	err := payment.TipUser(ctx, paymentReq, paymentResp)
	if err != nil {
		return err
	}

	fmt.Printf("paymentResp: %+v\n\n", paymentResp)
	// Pay the big blind
	paymentReq.DcrAmount = g.BB
	payment.TipUser(ctx, paymentReq, paymentResp)
	if err != nil {
		return err
	}

	return nil
}

func (g *PokerGame) DetermineWinner() rpc.Player {
	// This is a simplified version and assumes that the player with the highest card wins.
	// In a real poker PokerGame, you would need to implement hand rankings and compare them.
	highestCardValue := 0
	winner := g.Players[0]
	for _, player := range g.Players {
		for _, card := range append(player.Hand, g.CommunityCards...) {
			value := card.Value
			if value == "A" {
				value = "14"
			} else if value == "K" {
				value = "13"
			} else if value == "Q" {
				value = "12"
			} else if value == "J" {
				value = "11"
			}
			intValue, _ := strconv.Atoi(value)
			if intValue > highestCardValue {
				highestCardValue = intValue
				winner = player
			}
		}
	}
	fmt.Printf("The winner is %s with a high card of %d\n", winner.Nick, highestCardValue)

	return winner
}

func (g *PokerGame) DistributePot() {
	player := g.DetermineWinner()
	player.Chips += g.Pot
	g.Pot = 0
}

func (g *PokerGame) ResetPokerGame() {
	// g.Deck = NewDeck().Cards
	g.Deck = nil
	g.CommunityCards = []rpc.Card{}
	g.Pot = 0
	g.CurrentStage = "pre-flop"
	for _, player := range g.Players {
		// player.Hand = []Card{}
		player.HasActed = false
	}
	g.ShuffleDeck()
	g.Deal()
}

func (g *PokerGame) Showdown() {
	// Show all players' hands
	for _, player := range g.Players {
		fmt.Printf("%s's hand: %v and %v\n", player.Nick, player.Hand[0], player.Hand[1])
	}
	// Determine the winner, distribute the pot and reset the PokerGame
	g.DistributePot()
	g.ResetPokerGame()
}

func (g *PokerGame) ProgressPokerGame() {
	switch g.CurrentStage {
	case "pre-flop":
		if g.AllPlayersActed() {
			g.Flop()
			g.CurrentStage = "flop"
			g.ResetPlayerActions()
		}
	case "flop":
		if g.AllPlayersActed() {
			g.Turn()
			g.CurrentStage = "turn"
			g.ResetPlayerActions()
		}
	case "turn":
		if g.AllPlayersActed() {
			g.River()
			g.CurrentStage = "river"
			g.ResetPlayerActions()
		}
	case "river":
		if g.AllPlayersActed() {
			g.Showdown()
			g.CurrentStage = "showdown"
		}
	case "showdown":
		// Determine the winner and distribute the pot
		g.DetermineWinner()
		g.DistributePot()
		g.ResetPokerGame()
	}
}

func (g *PokerGame) AllPlayersActed() bool {
	for _, player := range g.Players {
		if !player.HasActed {
			return false
		}
	}
	return true
}

func (g *PokerGame) ResetPlayerActions() {
	for _, player := range g.Players {
		player.HasActed = false
	}
}

// type pokerPokerGameWindow struct {
// 	sync.Mutex
// 	uid clientintf.UserID

// 	msgs  []*chatMsg
// 	alias string
// 	me    string // nick of the local user
// 	gc    zkidentity.ShortID

// 	initTime time.Time // When the cw was created and history read.

// 	selEl         *chatMsgEl
// 	selElIndex    int
// 	maxSelectable int

// 	unreadIdx int

// 	PokerGameState *pokerPokerGame
// }

// func (pgw *pokerPokerGameWindow) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
// 	// var cmd tea.Cmd
// 	var cmds []tea.Cmd

// 	// ... handle PokerGame logic and user input here ...

// 	return pgw, batchCmds(cmds)
// }

// func (pgw pokerPokerGameWindow) headerView() string {
// 	headerMsg := pgw.as.styles.header.Render(" Poker PokerGame - ESC to exit")
// 	spaces := pgw.as.styles.header.Render(strings.Repeat(" ",
// 		max(0, pgw.as.winW-lipgloss.Width(headerMsg))))
// 	return headerMsg + spaces
// }

// func (pgw pokerPokerGameWindow) footerView() string {
// 	// ... display PokerGame status or instructions in the footer ...
// 	return pgw.as.footerView("")
// }

// func (pgw pokerPokerGameWindow) View() string {
// 	b := new(strings.Builder)
// 	write := b.WriteString
// 	write(pgw.headerView())
// 	write("\n")
// 	write(pgw.viewport.View())
// 	write("\n")
// 	write(pgw.footerView())
// 	write("\n")
// 	// ... display other PokerGame elements, such as player actions, cards, etc. ...
// 	return b.String()
// }

// func newPokerPokerGameWin(as *appState) (pokerPokerGameWindow, tea.Cmd) {
// 	pgw := pokerPokerGameWindow{
// 		as: as,
// 	}
// 	// ... initialize the PokerGame state and other fields here ...
// 	return pgw, nil
// }
