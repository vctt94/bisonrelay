package main

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"github.com/companyzero/bisonrelay/clientrpc/types"
	"github.com/companyzero/bisonrelay/rpc"
	"github.com/companyzero/bisonrelay/zkidentity"
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
	Players        []rpc.Player
	CommunityCards []rpc.Card
	CurrentStage   string
	DealerPosition int
	BigBlind       int
	SmallBlind     int
	Deck           []rpc.Card
	Pot            float64
	BB             float64
	SB             float64
	// CurrentPlayer represents an index of the Players array for the current
	// player to play at the current stage.
	CurrentPlayer int
	Winner        zkidentity.ShortID
	Bot           zkidentity.ShortID
}

type Deck struct {
	Cards []rpc.Card
}

// Helper function to get the next active position, excluding the bot.
func nextActivePosition(pos int, players []rpc.Player, botId zkidentity.ShortID) int {
	for {
		pos = (pos + 1) % len(players)
		if players[pos].IsActive && players[pos].ID != botId {
			break
		}
	}
	return pos
}

func Start(players []rpc.Player, botId zkidentity.ShortID, dealerPosition int, sb, bb float64) *PokerGame {
	// Set bot as not active.
	for i := range players {
		if players[i].ID == botId {
			players[i].IsActive = false
			break
		}
	}

	smallBlindPosition := nextActivePosition(dealerPosition, players, botId)
	bigBlindPosition := nextActivePosition(smallBlindPosition, players, botId)
	// currentPlayer := nextActivePosition(smallBlindPosition, players, botId)
	// as we start the current stage on the draw, the first player to receive cards
	// is the small blind. In others stages, the first player to act is the one
	// after the big blind.
	currentPlayer := smallBlindPosition

	game := &PokerGame{
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
	game.ShuffleDeck()

	return game
}

func (g *PokerGame) moveToNextPlayer() {
	g.CurrentPlayer = nextActivePosition(g.CurrentPlayer, g.Players, g.Bot)
}

func (g *PokerGame) ProgressPokerGame() {
	if g.AllPlayersActed() {
		switch g.CurrentStage {
		case Draw:
			g.CurrentStage = PreFlop
			g.CurrentPlayer = nextActivePosition(g.BigBlind, g.Players, g.Bot)
			g.ResetPlayerActions()
		case PreFlop:
			g.Flop()
			g.CurrentStage = Flop
			g.CurrentPlayer = nextActivePosition(g.BigBlind, g.Players, g.Bot)
			g.ResetPlayerActions()
		case Flop:
			g.Turn()
			g.CurrentStage = Turn
			g.CurrentPlayer = nextActivePosition(g.BigBlind, g.Players, g.Bot)
			g.ResetPlayerActions()
		case Turn:
			g.River()
			g.CurrentStage = River
			g.CurrentPlayer = nextActivePosition(g.BigBlind, g.Players, g.Bot)
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

func (g *PokerGame) DetermineWinner() rpc.Player {
	// This is a simplified version and assumes that the player with the highest card wins.
	// In a real poker PokerGame, you would need to implement hand rankings and compare them.
	highestCardValue := 0
	winner := g.Players[0]
	for _, player := range g.Players {
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
				winner = player
			}
		}
	}
	fmt.Printf("The winner is %s with a high card of %d\n", winner.Nick, highestCardValue)

	return winner
}

func (g *PokerGame) DistributePot() {
	g.Winner = g.DetermineWinner().ID
	// player.Chips += g.Pot
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
