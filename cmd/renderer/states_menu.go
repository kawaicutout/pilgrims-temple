package main

import (
	"errors"
	"runtime"

	"partyrogue/game"
)

// errQuit ends RunGame on desktop. Web returns to the menu instead.
var errQuit = errors.New("quit")

// handleKey dispatches one discrete key event to the active state.
func (a *App) handleKey(ev keyEvent) {
	k := game.NormalizeKey(ev.key, ev.code)
	switch a.state {
	case stateMenu:
		a.menuKeys(k)
	case stateMenuHelp:
		if k == game.KeyQuit || k == game.KeyEnter || k == game.KeyHelp {
			a.state = stateMenu
			a.show(game.RenderMainMenu(a.tuning, a.menu.Selected))
		}
	case stateScores:
		a.scoresKeys(k)
	case stateSeed:
		a.seedKeys(ev, k)
	case stateCreationClass:
		a.creationClassKeys(k)
	case stateCreationRace:
		a.creationRaceKeys(k)
	case stateCreationName:
		a.creationNameKeys(ev, k)
	case stateReview:
		a.reviewKeys(k)
	default:
		a.playKeys(ev, k)
	}
}

func (a *App) menuKeys(k game.Key) {
	switch k {
	case game.KeyUp:
		a.menu.Move(-1)
		a.show(game.RenderMainMenu(a.tuning, a.menu.Selected))
	case game.KeyDown:
		a.menu.Move(1)
		a.show(game.RenderMainMenu(a.tuning, a.menu.Selected))
	case game.KeyEnter:
		opts := game.GetMainMenuOptions()
		if a.menu.Selected < 0 || a.menu.Selected >= len(opts) {
			return
		}
		switch opts[a.menu.Selected] {
		case "New Game":
			a.seedState = &game.SeedEntryState{}
			a.drafts = []game.DraftMember{}
			a.reviewCursor = 0
			a.state = stateSeed
			a.show(game.RenderSeedEntry(a.tuning, a.seedState))
		case "Scores":
			a.scoresCursor = 0
			a.state = stateScores
			a.show(game.RenderScoresScreen(a.tuning, a.scoresCursor))
		case "Exit":
			if runtime.GOOS == "js" {
				a.toMenu()
			} else {
				a.quit = true
			}
		case "Load game":
			if lg, err := game.Load(); err == nil && lg != nil {
				a.g = lg
				a.state = statePlaying
				a.show(a.g.Render())
			}
		}
	case game.KeyQuit:
		if runtime.GOOS == "js" {
			a.toMenu()
		} else {
			a.quit = true
		}
	case game.KeyHelp:
		a.show(game.RenderHelpOverlayTuning(a.tuning))
		a.state = stateMenuHelp
	}
}

func (a *App) scoresKeys(k game.Key) {
	switch k {
	case game.KeyUp:
		if sb, err := game.LoadScoreboard(); err == nil && sb != nil && a.scoresCursor > 0 {
			a.scoresCursor--
		}
		a.show(game.RenderScoresScreen(a.tuning, a.scoresCursor))
	case game.KeyDown:
		if sb, err := game.LoadScoreboard(); err == nil && sb != nil && a.scoresCursor < len(sb.Entries)-1 {
			a.scoresCursor++
		}
		a.show(game.RenderScoresScreen(a.tuning, a.scoresCursor))
	case game.KeyEnter, game.KeyQuit:
		a.state = stateMenu
		a.show(game.RenderMainMenu(a.tuning, a.menu.Selected))
	default:
		a.show(game.RenderScoresScreen(a.tuning, a.scoresCursor))
	}
}

func (a *App) seedKeys(ev keyEvent, k game.Key) {
	if ev.key == "Backspace" {
		if a.seedState != nil {
			a.seedState.Backspace()
			a.show(game.RenderSeedEntry(a.tuning, a.seedState))
		}
		return
	}
	if ev.code == "" && len([]rune(ev.key)) == 1 {
		if a.seedState != nil {
			for _, r := range ev.key {
				a.seedState.AppendRune(r)
			}
			a.show(game.RenderSeedEntry(a.tuning, a.seedState))
		}
		return
	}
	switch k {
	case game.KeyEnter:
		a.racePick = game.NewRacePickState(len(a.drafts), len(a.drafts))
		if a.racePick == nil {
			a.show(game.RenderSeedEntry(a.tuning, a.seedState))
			return
		}
		a.state = stateCreationRace
		a.show(game.RenderRacePick(a.tuning, a.racePick))
	case game.KeyQuit:
		a.state = stateMenu
		a.show(game.RenderMainMenu(a.tuning, a.menu.Selected))
	default:
		a.show(game.RenderSeedEntry(a.tuning, a.seedState))
	}
}

func (a *App) creationClassKeys(k game.Key) {
	if a.classPick == nil {
		a.state = stateSeed
		a.show(game.RenderSeedEntry(a.tuning, a.seedState))
		return
	}
	switch k {
	case game.KeyUp:
		a.classPick.Move(-1)
		a.show(game.RenderClassPick(a.tuning, a.classPick))
	case game.KeyDown:
		a.classPick.Move(1)
		a.show(game.RenderClassPick(a.tuning, a.classPick))
	case game.KeyEnter:
		a.slotClass = a.classPick.Choice()
		if a.slotClass == "" {
			a.show(game.RenderClassPick(a.tuning, a.classPick))
			return
		}
		a.nameEntry = &game.NameEntryState{Class: a.slotClass, Race: a.slotRace}
		a.state = stateCreationName
		a.show(game.RenderNameEntry(a.tuning, a.slotClass, a.slotRace, ""))
	case game.KeyQuit:
		a.state = stateCreationRace
		if a.racePick != nil {
			a.show(game.RenderRacePick(a.tuning, a.racePick))
		} else {
			a.show(game.RenderSeedEntry(a.tuning, a.seedState))
		}
	default:
		a.show(game.RenderClassPick(a.tuning, a.classPick))
	}
}

func (a *App) creationRaceKeys(k game.Key) {
	if a.racePick == nil {
		a.state = stateSeed
		a.show(game.RenderSeedEntry(a.tuning, a.seedState))
		return
	}
	switch k {
	case game.KeyUp:
		a.racePick.Move(-1)
		a.show(game.RenderRacePick(a.tuning, a.racePick))
	case game.KeyDown:
		a.racePick.Move(1)
		a.show(game.RenderRacePick(a.tuning, a.racePick))
	case game.KeyEnter:
		a.slotRace = a.racePick.Choice()
		if a.slotRace == "" {
			a.show(game.RenderRacePick(a.tuning, a.racePick))
			return
		}
		var err error
		a.classPick, err = game.NewClassPickState(len(a.drafts), len(a.drafts))
		if err != nil || a.classPick == nil {
			a.show(game.RenderRacePick(a.tuning, a.racePick))
			return
		}
		a.state = stateCreationClass
		a.show(game.RenderClassPick(a.tuning, a.classPick))
	case game.KeyQuit:
		a.state = stateSeed
		a.show(game.RenderSeedEntry(a.tuning, a.seedState))
	default:
		a.show(game.RenderRacePick(a.tuning, a.racePick))
	}
}

func (a *App) creationNameKeys(ev keyEvent, k game.Key) {
	if a.nameEntry == nil {
		a.state = stateCreationClass
		if a.classPick != nil {
			a.show(game.RenderClassPick(a.tuning, a.classPick))
		} else {
			a.show(game.RenderMainMenu(a.tuning, a.menu.Selected))
		}
		return
	}
	if ev.key == "Backspace" {
		a.nameEntry.Backspace()
		a.show(game.RenderNameEntry(a.tuning, a.nameEntry.Class, a.nameEntry.Race, a.nameEntry.Text))
		return
	}
	if ev.code == "" && len([]rune(ev.key)) == 1 {
		for _, r := range ev.key {
			a.nameEntry.AppendRune(r)
		}
		a.show(game.RenderNameEntry(a.tuning, a.nameEntry.Class, a.nameEntry.Race, a.nameEntry.Text))
		return
	}
	switch k {
	case game.KeyEnter:
		a.drafts = append(a.drafts, game.DraftMember{Class: a.nameEntry.Class, Race: a.nameEntry.Race, Name: a.nameEntry.Text})
		a.reviewCursor = 0
		a.state = stateReview
		a.show(game.RenderReview(a.tuning, a.drafts, a.reviewCursor))
	case game.KeyQuit:
		a.state = stateCreationClass
		if a.classPick != nil {
			a.show(game.RenderClassPick(a.tuning, a.classPick))
		} else {
			a.show(game.RenderMainMenu(a.tuning, a.menu.Selected))
		}
	default:
		a.show(game.RenderNameEntry(a.tuning, a.nameEntry.Class, a.nameEntry.Race, a.nameEntry.Text))
	}
}

func (a *App) reviewKeys(k game.Key) {
	rows := game.ReviewRows(a.drafts)
	switch k {
	case game.KeyUp:
		if len(rows) > 0 {
			a.reviewCursor--
			if a.reviewCursor < 0 {
				a.reviewCursor = len(rows) - 1
			}
		}
		a.show(game.RenderReview(a.tuning, a.drafts, a.reviewCursor))
	case game.KeyDown:
		if len(rows) > 0 {
			a.reviewCursor++
			if a.reviewCursor >= len(rows) {
				a.reviewCursor = 0
			}
		}
		a.show(game.RenderReview(a.tuning, a.drafts, a.reviewCursor))
	case game.KeyEnter:
		if len(rows) == 0 || a.reviewCursor < 0 || a.reviewCursor >= len(rows) {
			a.show(game.RenderReview(a.tuning, a.drafts, a.reviewCursor))
			return
		}
		row := rows[a.reviewCursor]
		switch row.Action {
		case "discard":
			if row.Index >= 0 && row.Index < len(a.drafts) {
				a.drafts = append(a.drafts[:row.Index], a.drafts[row.Index+1:]...)
			}
			if a.reviewCursor >= len(game.ReviewRows(a.drafts)) {
				a.reviewCursor = len(game.ReviewRows(a.drafts)) - 1
			}
			if a.reviewCursor < 0 {
				a.reviewCursor = 0
			}
			a.show(game.RenderReview(a.tuning, a.drafts, a.reviewCursor))
		case "add":
			a.racePick = game.NewRacePickState(len(a.drafts), len(a.drafts))
			if a.racePick == nil {
				a.show(game.RenderReview(a.tuning, a.drafts, a.reviewCursor))
				return
			}
			a.state = stateCreationRace
			a.show(game.RenderRacePick(a.tuning, a.racePick))
		case "begin":
			classes := make([]string, len(a.drafts))
			races := make([]string, len(a.drafts))
			names := make([]string, len(a.drafts))
			for i, d := range a.drafts {
				classes[i], races[i], names[i] = d.Class, d.Race, d.Name
			}
			seed := a.seedState.SeedOr(seedNow())
			a.g = game.NewGameFull(seed, a.tuning, classes, races, names)
			a.state = statePlaying
			a.show(a.g.Render())
		default:
			a.show(game.RenderReview(a.tuning, a.drafts, a.reviewCursor))
		}
	case game.KeyQuit:
		a.state = stateMenu
		a.show(game.RenderMainMenu(a.tuning, a.menu.Selected))
	default:
		a.show(game.RenderReview(a.tuning, a.drafts, a.reviewCursor))
	}
}
