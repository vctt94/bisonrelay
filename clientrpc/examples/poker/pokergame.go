package main

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"github.com/companyzero/bisonrelay/clientrpc/types"
)

var (
	suits  = []string{"♥ Hearts", "♦ Diamonds", "♣ Clubs", "♠ Spades"}
	values = []string{"2", "3", "4", "5", "6", "7", "8", "9", "10", "J", "Q", "K", "A"}
)

const (
	Draw     = "draw"
	PreFlop  = "pre-flop"
	Flop     = "flop"
	Turn     = "turn"
	River    = "river"
	Showdown = "showdown"
)

type PokerGame struct {
	ID             string   `json:"id"` //gcid
	Players        []Player `json:"players"`
	CommunityCards []Card   `json:"communitycards"`
	CurrentStage   string   `json:"currentstage"`
	CurrentPlayer  int      `json:"currentplayer"`
	Winner         int      `json:"winner"`
	DealerPosition int      `json:"dealerposition"`
	BigBlind       int      `json:"bigblind"`
	SmallBlind     int      `json:"smallblind"`
	Pot            float64  `json:"pot"`
	BB             float64  `json:"bb"`
	SB             float64  `json:"sb"`
	Deck           []Card   `json:"deck"`
	// the bot client responsible to managing the game
	Bot string `json:"bot"`
}

type Player struct {
	ID       string
	Nick     string
	Hand     []Card
	Chips    int
	Bet      int
	IsActive bool
	HasActed bool
}

type Card struct {
	Value string
	Suit  string
}

// Helper function to get the next active position, excluding the bot.
func nextActivePosition(pos int, players []Player) int {
	for {
		pos = (pos + 1) % len(players)
		if players[pos].IsActive {
			break
		}
	}
	return pos
}

func New(id string, players []Player, dealerPosition int, sb, bb float64) *PokerGame {
	smallBlindPosition := nextActivePosition(dealerPosition, players)
	bigBlindPosition := nextActivePosition(smallBlindPosition, players)
	// currentPlayer := nextActivePosition(smallBlindPosition, players)
	// as we start the current stage on the draw, the first player to receive cards
	// is the small blind. In others stages, the first player to act is the one
	// after the big blind.
	currentPlayer := smallBlindPosition

	game := &PokerGame{
		ID:             id,
		Bot:            botId,
		CurrentStage:   Draw,
		Pot:            0,
		DealerPosition: dealerPosition,
		CurrentPlayer:  currentPlayer,
		SmallBlind:     smallBlindPosition,
		BigBlind:       bigBlindPosition,
		BB:             bb,
		SB:             sb,
		Players:        players,
	}

	return game
}

func (g *PokerGame) moveToNextPlayer() {
	g.CurrentPlayer = nextActivePosition(g.CurrentPlayer, g.Players)
}

func (g *PokerGame) ProgressPokerGame() {
	if g.AllPlayersActed() {
		switch g.CurrentStage {
		case Draw:
			g.CurrentStage = PreFlop
			g.CurrentPlayer = nextActivePosition(g.BigBlind, g.Players)
			g.ResetPlayerActions()
		case PreFlop:
			g.Flop()
			g.CurrentStage = Flop
			g.CurrentPlayer = nextActivePosition(g.BigBlind, g.Players)
			g.ResetPlayerActions()
		case Flop:
			g.Turn()
			g.CurrentStage = Turn
			g.CurrentPlayer = nextActivePosition(g.BigBlind, g.Players)
			g.ResetPlayerActions()
		case Turn:
			g.River()
			g.CurrentStage = River
			g.CurrentPlayer = nextActivePosition(g.BigBlind, g.Players)
			g.ResetPlayerActions()
		case River:
			g.Showdown()
			g.CurrentStage = Showdown
			// XXX
			// g.CurrentPlayer after showdown needs to progress blinds and button
		case Showdown:
			g.DetermineWinner()
			g.DistributePot()
			// g.ResetPokerGame()
		}
	} else {
		g.moveToNextPlayer()
	}
}

func (g *PokerGame) Draw() Card {
	card := g.Deck[len(g.Deck)-1]
	g.Deck = g.Deck[:len(g.Deck)-1]
	return card
}

func (g *PokerGame) ShuffleDeck() {
	if g.Deck == nil {
		var cards []Card
		for _, suit := range suits {
			for _, value := range values {
				cards = append(cards, Card{Suit: suit, Value: value})
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
	// paymentReq := &types.TipUserRequest{
	// 	DcrAmount:   g.SB,
	// 	User:        g.Bot,
	// 	MaxAttempts: 3,
	// }
	// fmt.Printf("paymentReq: %+v\n\n", paymentReq)
	// paymentResp := &types.TipUserResponse{}
	// err := payment.TipUser(ctx, paymentReq, paymentResp)
	// if err != nil {
	// 	return err
	// }

	// fmt.Printf("paymentResp: %+v\n\n", paymentResp)
	// // Pay the big blind
	// paymentReq.DcrAmount = g.BB
	// payment.TipUser(ctx, paymentReq, paymentResp)
	// if err != nil {
	// 	return err
	// }

	return nil
}

func (g *PokerGame) DetermineWinner() int {
	// This is a simplified version and assumes that the player with the highest card wins.
	// In a real poker PokerGame, you would need to implement hand rankings and compare them.
	highestCardValue := 0
	winner := 0
	for i, player := range g.Players {
		if !player.IsActive {
			continue
		}
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
				winner = i
			}
		}
	}

	return winner
}

func (g *PokerGame) DistributePot() {
	g.Winner = g.DetermineWinner()
	// player.Chips += g.Pot
}

func (g *PokerGame) ResetPokerGame() {
	// g.Deck = NewDeck().Cards
	g.Deck = nil
	g.CommunityCards = []Card{}
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
		if !player.IsActive {
			continue
		}
		fmt.Printf("%s's hand: %v and %v\n", player.Nick, player.Hand[0], player.Hand[1])
	}
	// Determine the winner, distribute the pot and reset the PokerGame
	g.DistributePot()
}

func (g *PokerGame) AllPlayersActed() bool {
	for _, player := range g.Players {
		if !player.IsActive {
			continue
		}
		if !player.HasActed {
			return false
		}
	}
	return true
}

func (g *PokerGame) ResetPlayerActions() {
	for i, _ := range g.Players {
		g.Players[i].HasActed = false
	}
}
