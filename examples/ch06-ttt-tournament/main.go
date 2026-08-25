// ttt-bench re-runs the book ch06 tic-tac-toe tournament on the draw-aware
// halting model (the one that supersedes the pre-draw construction): win
// detectors absorb the turn token and game_active, and a weight-9 move
// counter calls the draw. The ODE evaluator is the chapter's technique —
// one mass-action solve per candidate move — but scored on the declared
// objective win_x - win_o - draw, so a draw counts fully for O.
package main

import (
	"fmt"
	"math/rand"
	"sort"

	"github.com/pflow-xyz/go-pflow/petri"
	"github.com/pflow-xyz/go-pflow/solver"
)

type arc struct {
	from, to string
	weight   int
}

var cells = []string{"00", "01", "02", "10", "11", "12", "20", "21", "22"}

var lines = map[string][]string{
	"row0": {"00", "01", "02"}, "row1": {"10", "11", "12"}, "row2": {"20", "21", "22"},
	"col0": {"00", "10", "20"}, "col1": {"01", "11", "21"}, "col2": {"02", "12", "22"},
	"diag": {"00", "11", "22"}, "anti": {"02", "11", "20"},
}

type model struct {
	places      []string
	transitions []string
	rates       map[string]float64
	inputs      map[string][]arc // transition -> consuming arcs
	outputs     map[string][]arc // transition -> producing arcs
	initial     map[string]int
}

func buildModel() *model {
	m := &model{
		rates:   map[string]float64{},
		inputs:  map[string][]arc{},
		outputs: map[string][]arc{},
		initial: map[string]int{},
	}
	addPlace := func(id string, init int) { m.places = append(m.places, id); m.initial[id] = init }
	addT := func(id string, rate float64) { m.transitions = append(m.transitions, id); m.rates[id] = rate }
	in := func(t, p string, w int) { m.inputs[t] = append(m.inputs[t], arc{p, t, w}) }
	out := func(t, p string, w int) { m.outputs[t] = append(m.outputs[t], arc{t, p, w}) }

	for _, c := range cells {
		addPlace("p"+c, 1)
		addPlace("x"+c, 0)
		addPlace("o"+c, 0)
	}
	addPlace("x_turn", 1)
	addPlace("o_turn", 0)
	addPlace("game_active", 1)
	addPlace("move_tokens", 0)
	addPlace("win_x", 0)
	addPlace("win_o", 0)
	addPlace("draw", 0)

	for _, c := range cells {
		xt, ot := "x_play_"+c, "o_play_"+c
		addT(xt, 1)
		in(xt, "p"+c, 1)
		in(xt, "x_turn", 1)
		out(xt, "x"+c, 1)
		out(xt, "o_turn", 1)
		out(xt, "move_tokens", 1)
		addT(ot, 1)
		in(ot, "p"+c, 1)
		in(ot, "o_turn", 1)
		out(ot, "o"+c, 1)
		out(ot, "x_turn", 1)
		out(ot, "move_tokens", 1)
	}
	// Win detectors: catalytic on the claimed cells (consume and return),
	// absorb the opponent's turn token and game_active — the game halts.
	lineNames := make([]string, 0, len(lines))
	for name := range lines {
		lineNames = append(lineNames, name)
	}
	sort.Strings(lineNames)
	for _, name := range lineNames {
		xs := "x_win_" + name
		addT(xs, 720)
		for _, c := range lines[name] {
			in(xs, "x"+c, 1)
			out(xs, "x"+c, 1)
		}
		in(xs, "o_turn", 1)
		in(xs, "game_active", 1)
		out(xs, "win_x", 1)

		os := "o_win_" + name
		addT(os, 720)
		for _, c := range lines[name] {
			in(os, "o"+c, 1)
			out(os, "o"+c, 1)
		}
		in(os, "x_turn", 1)
		in(os, "game_active", 1)
		out(os, "win_o", 1)
	}
	addT("call_draw", 1)
	in("call_draw", "move_tokens", 9)
	in("call_draw", "game_active", 1)
	out("call_draw", "draw", 1)
	return m
}

// ---- discrete firing rule ----

func (m *model) enabled(t string, mk map[string]int) bool {
	for _, a := range m.inputs[t] {
		if mk[a.from] < a.weight {
			return false
		}
	}
	return true
}

func (m *model) fire(t string, mk map[string]int) map[string]int {
	next := make(map[string]int, len(mk))
	for k, v := range mk {
		next[k] = v
	}
	for _, a := range m.inputs[t] {
		next[a.from] -= a.weight
	}
	for _, a := range m.outputs[t] {
		next[a.to] += a.weight
	}
	return next
}

var playerOwned = func() map[string]bool {
	owned := map[string]bool{}
	for _, c := range cells {
		owned["x_play_"+c] = true
		owned["o_play_"+c] = true
	}
	return owned
}()

// fireHouse fires unowned transitions (win detectors, call_draw) to quiescence.
func (m *model) fireHouse(mk map[string]int) map[string]int {
	for {
		fired := false
		for _, t := range m.transitions {
			if !playerOwned[t] && m.enabled(t, mk) {
				mk = m.fire(t, mk)
				fired = true
			}
		}
		if !fired {
			return mk
		}
	}
}

// ---- ODE evaluator ----

func (m *model) toPetri() *petri.PetriNet {
	net := petri.NewPetriNet()
	for _, p := range m.places {
		net.AddPlace(p, m.initial[p], nil, 0, 0, nil)
	}
	for _, t := range m.transitions {
		net.AddTransition(t, "", 0, 0, nil)
	}
	for _, t := range m.transitions {
		for _, a := range m.inputs[t] {
			net.AddArc(a.from, t, a.weight, false)
		}
		for _, a := range m.outputs[t] {
			net.AddArc(t, a.to, a.weight, false)
		}
	}
	return net
}

// objective reads win_x - win_o - draw from a final state; X maximizes it.
func objective(final map[string]float64) float64 {
	return final["win_x"] - final["win_o"] - final["draw"]
}

func (m *model) odeScore(net *petri.PetriNet, mk map[string]int) float64 {
	state := make(map[string]float64, len(mk))
	for k, v := range mk {
		state[k] = float64(v)
	}
	prob := solver.NewProblem(net, state, [2]float64{0, 3.0}, m.rates)
	opts := &solver.Options{
		Dt: 0.2, Dtmin: 1e-4, Dtmax: 1.0,
		Abstol: 1e-4, Reltol: 1e-3, Maxiters: 1000, Adaptive: true,
	}
	sol := solver.Solve(prob, solver.Tsit5(), opts)
	return objective(sol.GetFinalState())
}

// ---- the oracle: exact minimax steered by the ODE prior ----

// odePrior scores each cell once, by the ODE evaluation of X opening there.
// This is the relaxation used as an oracle: it never decides what a position
// is worth — it only decides which move the exact search examines first.
func odePrior(m *model, net *petri.PetriNet) map[string]float64 {
	prior := map[string]float64{}
	empty := make(map[string]int, len(m.initial))
	for k, v := range m.initial {
		empty[k] = v
	}
	for _, c := range cells {
		prior[c] = m.odeScore(net, m.fire("x_play_"+c, empty))
	}
	return prior
}

var searchNodes int

// minimax is exact alpha-beta over the discrete net: enabled transitions are
// the moves, house transitions fire between turns, a missing turn token is a
// leaf, and the leaf value is the declared objective. prior orders moves;
// nil means declared order.
func (m *model) minimax(mk map[string]int, alpha, beta float64, prior map[string]float64) float64 {
	searchNodes++
	mk = m.fireHouse(mk)
	prefix, maximizes := "", false
	switch {
	case mk["x_turn"] > 0:
		prefix, maximizes = "x_play_", true
	case mk["o_turn"] > 0:
		prefix = "o_play_"
	default:
		// Game-theoretic leaf: win +1, loss -1, draw 0. The declared
		// objective folds the draw into the defender's win (X indifferent
		// between drawing and losing) — right for "O denies X", wrong for
		// demonstrating perfect play, where the three outcomes must rank.
		return float64(mk["win_x"] - mk["win_o"])
	}
	var moves []string
	for _, c := range cells {
		if m.enabled(prefix+c, mk) {
			moves = append(moves, prefix+c)
		}
	}
	if len(moves) == 0 {
		// A turn token with no legal move: the board is full and call_draw
		// has already fired. Score the outcome places (draw = 0).
		return float64(mk["win_x"] - mk["win_o"])
	}
	if prior != nil {
		sort.SliceStable(moves, func(i, j int) bool {
			return prior[moves[i][len(prefix):]] > prior[moves[j][len(prefix):]]
		})
	}
	if maximizes {
		v := -2.0
		for _, mv := range moves {
			v = max(v, m.minimax(m.fire(mv, mk), alpha, beta, prior))
			alpha = max(alpha, v)
			if alpha >= beta {
				break
			}
		}
		return v
	}
	v := 2.0
	for _, mv := range moves {
		v = min(v, m.minimax(m.fire(mv, mk), alpha, beta, prior))
		beta = min(beta, v)
		if alpha >= beta {
			break
		}
	}
	return v
}

// oracleMove plays exact minimax with the ODE prior ordering the search,
// choosing uniformly among equal-optimal moves so repeated games vary.
func oracleMove(prior map[string]float64) func(*model, *petri.PetriNet, map[string]int, []string, bool, *rand.Rand) string {
	return func(m *model, _ *petri.PetriNet, mk map[string]int, moves []string, maximizes bool, rng *rand.Rand) string {
		var best []string
		bestV := 0.0
		for i, mv := range moves {
			v := m.minimax(m.fire(mv, mk), -2, 2, prior)
			if !maximizes {
				v = -v
			}
			if i == 0 || v > bestV {
				best, bestV = []string{mv}, v
			} else if v == bestV {
				best = append(best, mv)
			}
		}
		return best[rng.Intn(len(best))]
	}
}

// ---- game loop ----

type strategy func(m *model, net *petri.PetriNet, mk map[string]int, moves []string, maximizes bool, rng *rand.Rand) string

func randomMove(_ *model, _ *petri.PetriNet, _ map[string]int, moves []string, _ bool, rng *rand.Rand) string {
	return moves[rng.Intn(len(moves))]
}

// odeMove applies each candidate discretely (house transitions included, so an
// immediate win or draw is scored exactly), then solves the relaxation from the
// resulting state. Deterministic: ties break by declared move order.
func odeMove(m *model, net *petri.PetriNet, mk map[string]int, moves []string, maximizes bool, _ *rand.Rand) string {
	best, bestScore := "", 0.0
	for i, mv := range moves {
		after := m.fireHouse(m.fire(mv, mk))
		var score float64
		if after["x_turn"] == 0 && after["o_turn"] == 0 {
			// Terminal: score the objective exactly, no solve needed.
			final := make(map[string]float64, len(after))
			for k, v := range after {
				final[k] = float64(v)
			}
			score = objective(final)
		} else {
			score = m.odeScore(net, after)
		}
		if !maximizes {
			score = -score
		}
		if i == 0 || score > bestScore {
			best, bestScore = mv, score
		}
	}
	return best
}

// playGame returns "X", "O", or "draw".
func (m *model) playGame(net *petri.PetriNet, xs, os strategy, rng *rand.Rand) string {
	mk := make(map[string]int, len(m.initial))
	for k, v := range m.initial {
		mk[k] = v
	}
	for {
		mk = m.fireHouse(mk)
		if mk["win_x"] > 0 {
			return "X"
		}
		if mk["win_o"] > 0 {
			return "O"
		}
		if mk["draw"] > 0 {
			return "draw"
		}
		var prefix string
		var strat strategy
		var maximizes bool
		switch {
		case mk["x_turn"] > 0:
			prefix, strat, maximizes = "x_play_", xs, true
		case mk["o_turn"] > 0:
			prefix, strat, maximizes = "o_play_", os, false
		default:
			return "draw" // unreachable: halting absorbed the turn without an outcome
		}
		var moves []string
		for _, c := range cells {
			if m.enabled(prefix+c, mk) {
				moves = append(moves, prefix+c)
			}
		}
		if len(moves) == 0 {
			return "draw"
		}
		mk = m.fire(strat(m, net, mk, moves, maximizes, rng), mk)
	}
}

func main() {
	m := buildModel()
	net := m.toPetri()

	run := func(name string, xs, os strategy, games int, seed int64) {
		rng := rand.New(rand.NewSource(seed))
		wins := map[string]int{}
		for i := 0; i < games; i++ {
			wins[m.playGame(net, xs, os, rng)]++
		}
		fmt.Printf("%-16s games=%d  X wins: %d (%.1f%%)  O wins: %d (%.1f%%)  draws: %d (%.1f%%)\n",
			name, games,
			wins["X"], 100*float64(wins["X"])/float64(games),
			wins["O"], 100*float64(wins["O"])/float64(games),
			wins["draw"], 100*float64(wins["draw"])/float64(games))
	}

	prior := odePrior(m, net)
	oracle := oracleMove(prior)

	run("ODE vs Random", odeMove, randomMove, 100, 11)
	run("Random vs ODE", randomMove, odeMove, 100, 11)
	run("Oracle vs Oracle", oracle, oracle, 100, 11)
	run("ODE vs Oracle", odeMove, oracle, 100, 11)
	run("Oracle vs ODE", oracle, odeMove, 100, 11)

	// What the prior buys the search: nodes expanded from the empty board.
	empty := make(map[string]int, len(m.initial))
	for k, v := range m.initial {
		empty[k] = v
	}
	searchNodes = 0
	m.minimax(empty, -2, 2, nil)
	unordered := searchNodes
	searchNodes = 0
	v := m.minimax(empty, -2, 2, prior)
	fmt.Printf("\nexact minimax from the empty board: value %.0f (the draw)\n", v)
	fmt.Printf("nodes expanded: %d declared order, %d with the ODE prior ordering\n", unordered, searchNodes)
	// Both sides use the same evaluator; scores are deterministic up to
	// floating-point summation order (map iteration inside the solver), so
	// sequences vary slightly run to run — the outcome does not.
	run("ODE vs ODE", odeMove, odeMove, 100, 11)

	// Trace the single deterministic ODE-vs-ODE game move by move.
	fmt.Println("\nODE vs ODE trace:")
	mk := make(map[string]int, len(m.initial))
	for k, v := range m.initial {
		mk[k] = v
	}
	rng := rand.New(rand.NewSource(11))
	for ply := 1; ; ply++ {
		mk = m.fireHouse(mk)
		if mk["win_x"] > 0 || mk["win_o"] > 0 || mk["draw"] > 0 {
			fmt.Printf("  outcome: win_x=%d win_o=%d draw=%d\n", mk["win_x"], mk["win_o"], mk["draw"])
			break
		}
		var prefix string
		maximizes := false
		if mk["x_turn"] > 0 {
			prefix, maximizes = "x_play_", true
		} else {
			prefix = "o_play_"
		}
		var moves []string
		for _, c := range cells {
			if m.enabled(prefix+c, mk) {
				moves = append(moves, prefix+c)
			}
		}
		mv := odeMove(m, net, mk, moves, maximizes, rng)
		fmt.Printf("  ply %d: %s\n", ply, mv)
		mk = m.fire(mv, mk)
	}
}
