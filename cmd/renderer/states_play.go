package main

import (
	"log"

	"partyrogue/game"
)

// postOutcome applies the shared save policy and navigates, logging
// save errors to stderr instead of swallowing them.
func (a *App) postOutcome(outcome game.PostOutcome, err error) {
	if err != nil {
		log.Printf("save failed: %v", err)
	}
	switch outcome {
	case game.PostMenu:
		a.g = nil
		a.state = stateMenu
		a.show(game.RenderMainMenu(a.tuning, a.menu.Selected))
	case game.PostDeath:
		a.state = stateOver
		a.show(a.g.Render())
	}
}

// postAction runs the standard after-action epilogue: redraw, level-up
// overlay, quit-to-menu with save, or death modal. Mirrors cmd/terminal.
func (a *App) postAction() {
	if a.g == nil {
		a.state = statePlaying
		a.show(a.g.Render())
		return
	}
	a.show(a.g.Render())
	if a.g.LevelUpPending != nil {
		a.show(a.g.RenderLevelUp())
	}
	outcome, err := a.g.PostAction()
	a.postOutcome(outcome, err)
}

func (a *App) playKeys(ev keyEvent, k game.Key) {
	switch a.state {
	case statePlaying:
		a.playingKeys(ev, k)
	case stateUseInventory:
		a.useInventoryKeys(k)
	case stateUseMember:
		a.useMemberKeys(k)
	case stateUseTarget:
		a.useTargetKeys(ev, k)
	case stateThrowMenu:
		a.throwMenuKeys(k)
	case stateThrowCursor:
		a.throwCursorKeys(ev, k)
	case stateMerchant:
		a.merchantKeys(k)
	case stateShrine:
		a.shrineKeys(k)
	case stateWizard:
		a.wizardKeys(k)
	case stateWizardAddMember:
		a.wizardAddKeys(k)
	case stateWizardRemoveMember:
		a.wizardRemoveKeys(k, true)
	case stateWizardResurrectMember:
		a.wizardRemoveKeys(k, false)
	case stateOver:
		if k == game.KeyQuit || k == game.KeyEnter {
			a.overToMenu()
		}
	}
}

func (a *App) playingKeys(ev keyEvent, k game.Key) {
	g := a.g
	if g == nil {
		a.state = stateMenu
		a.show(game.RenderMainMenu(a.tuning, a.menu.Selected))
		return
	}
	if g.HelpActive {
		switch k {
		case game.KeyQuit, game.KeyEnter, game.KeyHelp:
			g.HelpActive = false
			a.show(g.Render())
			if g.LevelUpPending != nil {
				a.show(g.RenderLevelUp())
			}
		default:
			a.show(g.RenderHelpOverlay())
		}
		return
	}
	if g.LevelUpPending != nil {
		pick := g.LevelUpPending.Picks[g.LevelUpPending.Current]
		handled := false
		cursorMoved := false
		switch k {
		case game.KeyUp:
			if !pick.IsAffix {
				g.MoveLevelUpCursor(-1)
				cursorMoved = true
			}
		case game.KeyDown:
			if !pick.IsAffix {
				g.MoveLevelUpCursor(1)
				cursorMoved = true
			}
		case game.KeyEnter:
			idx := 0
			if !pick.IsAffix {
				idx = g.LevelUpPending.Cursor
			}
			g.ApplyTalentPick(g.LevelUpPending.Current, idx)
			handled = true
		case game.KeyQuit:
			g.ApplyTalentPick(g.LevelUpPending.Current, 0)
			handled = true
		}
		if !handled && !cursorMoved && !pick.IsAffix {
			switch ev.key {
			case "1":
				g.ApplyTalentPick(g.LevelUpPending.Current, 0)
				handled = true
			case "2":
				if len(pick.Options) > 1 {
					g.ApplyTalentPick(g.LevelUpPending.Current, 1)
					handled = true
				}
			case "3":
				if len(pick.Options) > 2 {
					g.ApplyTalentPick(g.LevelUpPending.Current, 2)
					handled = true
				}
			}
		}
		if handled {
			if g.LevelUpPending == nil {
				a.show(g.Render())
			} else {
				a.show(g.RenderLevelUp())
			}
		} else {
			a.show(g.RenderLevelUp())
		}
		return
	}
	if k == game.KeyWizard && (g.Look == nil || !g.Look.Active) && !g.Over && !g.Quit {
		a.wizardState.Selected = 0
		a.state = stateWizard
		a.show(g.RenderWizardMenu(a.tuning, a.wizardState.Selected))
		return
	}
	if k == game.KeyUse && (g.Look == nil || !g.Look.Active) && !g.Over && !g.Quit && !g.UsePending.Active && !g.ThrowPending.Active {
		if g.TryUseForge() {
			g.EndPlayerTurn("")
			a.show(g.Render())
			if g.LevelUpPending != nil {
				a.show(g.RenderLevelUp())
			}
			return
		}
		entries := g.InventoryUseEntries()
		if len(entries) == 0 {
			g.Logf("No potions or scrolls to use.")
			a.show(g.Render())
			return
		}
		a.useSelected = 0
		a.state = stateUseInventory
		a.show(g.RenderUseMenu(a.useSelected))
		return
	}
	if k == game.KeyThrow && (g.Look == nil || !g.Look.Active) && !g.ThrowPending.Active && !g.UsePending.Active && !g.Over && !g.Quit {
		entries := g.InventoryEntriesByKind("potion")
		if len(entries) == 0 {
			g.Logf("No potions to throw.")
			a.show(g.Render())
			return
		}
		a.throwSelected = 0
		a.state = stateThrowMenu
		a.show(g.RenderThrowMenu(a.throwSelected))
		return
	}
	g.HandleKey(k)
	if g.Merchant.Active {
		a.state = stateMerchant
		a.show(g.RenderMerchantMenu())
		return
	}
	if g.Shrine.Active {
		a.state = stateShrine
		a.show(g.RenderShrineMenu())
		return
	}
	if g.HelpActive {
		a.show(g.RenderHelpOverlay())
	} else {
		a.show(g.Render())
	}
	if g.LevelUpPending != nil {
		a.show(g.RenderLevelUp())
	}
	outcome, err := g.PostAction()
	a.postOutcome(outcome, err)
}

func (a *App) useInventoryKeys(k game.Key) {
	g := a.g
	if g == nil {
		a.state = statePlaying
		a.show(a.g.Render())
		return
	}
	entries := g.InventoryUseEntries()
	switch k {
	case game.KeyUp:
		if len(entries) > 0 {
			a.useSelected--
			if a.useSelected < 0 {
				a.useSelected = len(entries) - 1
			}
		}
		a.show(g.RenderUseMenu(a.useSelected))
	case game.KeyDown:
		if len(entries) > 0 {
			a.useSelected++
			if a.useSelected >= len(entries) {
				a.useSelected = 0
			}
		}
		a.show(g.RenderUseMenu(a.useSelected))
	case game.KeyQuit:
		a.state = statePlaying
		a.show(g.Render())
	case game.KeyEnter:
		if len(entries) > 0 && a.useSelected >= 0 && a.useSelected < len(entries) {
			e := entries[a.useSelected]
			a.useMemberMode = game.UseTargetMode(e.Kind, e.Appearance)
			if a.useMemberMode == "tile" {
				g.StartUse(e.Appearance, e.Kind)
				a.state = stateUseTarget
				a.show(g.Render())
			} else {
				a.useMemberAppearance = e.Appearance
				a.useMemberTitle = e.DisplayName
				a.useMemberSelected = 0
				a.state = stateUseMember
				a.show(g.RenderUseMemberMenu(a.useMemberTitle, a.useMemberMode == "partyChoice", a.useMemberSelected))
			}
		} else {
			a.state = statePlaying
			a.show(g.Render())
		}
	default:
		a.show(g.RenderUseMenu(a.useSelected))
	}
}

func (a *App) useMemberKeys(k game.Key) {
	g := a.g
	if g == nil {
		a.state = statePlaying
		a.show(a.g.Render())
		return
	}
	allowParty := a.useMemberMode == "partyChoice"
	_, rows := g.UseMemberOptions(allowParty)
	switch k {
	case game.KeyUp:
		if len(rows) > 0 {
			a.useMemberSelected--
			if a.useMemberSelected < 0 {
				a.useMemberSelected = len(rows) - 1
			}
		}
		a.show(g.RenderUseMemberMenu(a.useMemberTitle, allowParty, a.useMemberSelected))
	case game.KeyDown:
		if len(rows) > 0 {
			a.useMemberSelected++
			if a.useMemberSelected >= len(rows) {
				a.useMemberSelected = 0
			}
		}
		a.show(g.RenderUseMemberMenu(a.useMemberTitle, allowParty, a.useMemberSelected))
	case game.KeyQuit:
		a.state = stateUseInventory
		a.show(g.RenderUseMenu(a.useSelected))
	case game.KeyEnter:
		if len(rows) > 0 && a.useMemberSelected >= 0 && a.useMemberSelected < len(rows) {
			g.TryUseAppearanceOnMember(a.useMemberAppearance, rows[a.useMemberSelected])
			a.state = statePlaying
			a.postAction()
		} else {
			a.state = statePlaying
			a.show(g.Render())
		}
	default:
		a.show(g.RenderUseMemberMenu(a.useMemberTitle, allowParty, a.useMemberSelected))
	}
}

func (a *App) useTargetKeys(_ keyEvent, k game.Key) {
	g := a.g
	if g == nil {
		a.state = statePlaying
		a.show(a.g.Render())
		return
	}
	switch k {
	case game.KeyQuit:
		g.CancelUse()
		g.Logf("Cancelled use.")
		a.state = statePlaying
		a.show(g.Render())
	case game.KeyEnter:
		g.UseAt(g.UsePending.Cursor)
		a.state = statePlaying
		a.postAction()
	default:
		handledTurn := g.HandleKey(k)
		a.show(g.Render())
		if handledTurn {
			a.state = statePlaying
			a.postAction()
		}
	}
}

func (a *App) throwMenuKeys(k game.Key) {
	g := a.g
	if g == nil {
		a.state = statePlaying
		a.show(a.g.Render())
		return
	}
	entries := g.InventoryEntriesByKind("potion")
	switch k {
	case game.KeyUp:
		if len(entries) > 0 {
			a.throwSelected--
			if a.throwSelected < 0 {
				a.throwSelected = len(entries) - 1
			}
		}
		a.show(g.RenderThrowMenu(a.throwSelected))
	case game.KeyDown:
		if len(entries) > 0 {
			a.throwSelected++
			if a.throwSelected >= len(entries) {
				a.throwSelected = 0
			}
		}
		a.show(g.RenderThrowMenu(a.throwSelected))
	case game.KeyQuit:
		a.state = statePlaying
		a.show(g.Render())
	case game.KeyEnter:
		if len(entries) > 0 && a.throwSelected >= 0 && a.throwSelected < len(entries) {
			appearance := entries[a.throwSelected].Appearance
			g.StartThrow(appearance)
			a.state = stateThrowCursor
			a.show(g.Render())
		} else {
			a.state = statePlaying
			a.show(g.Render())
		}
	default:
		a.show(g.RenderThrowMenu(a.throwSelected))
	}
}

func (a *App) throwCursorKeys(_ keyEvent, k game.Key) {
	g := a.g
	if g == nil {
		a.state = statePlaying
		a.show(a.g.Render())
		return
	}
	switch k {
	case game.KeyQuit:
		g.CancelThrow()
		g.Logf("Cancelled throw.")
		a.state = statePlaying
		a.show(g.Render())
	case game.KeyEnter:
		g.ThrowAt(g.ThrowPending.Cursor)
		a.state = statePlaying
		a.postAction()
	default:
		handledTurn := g.HandleKey(k)
		a.show(g.Render())
		if handledTurn {
			a.state = statePlaying
			a.postAction()
		}
	}
}

func (a *App) merchantKeys(k game.Key) {
	g := a.g
	if g == nil {
		a.state = statePlaying
		a.show(a.g.Render())
		return
	}
	switch k {
	case game.KeyUp:
		if g.Merchant.Active && len(g.Merchant.Wares) > 0 {
			g.Merchant.Selected--
			if g.Merchant.Selected < 0 {
				g.Merchant.Selected = len(g.Merchant.Wares) - 1
			}
		}
		a.show(g.RenderMerchantMenu())
	case game.KeyDown:
		if g.Merchant.Active && len(g.Merchant.Wares) > 0 {
			g.Merchant.Selected++
			if g.Merchant.Selected >= len(g.Merchant.Wares) {
				g.Merchant.Selected = 0
			}
		}
		a.show(g.RenderMerchantMenu())
	case game.KeyQuit:
		g.CancelMerchant()
		a.state = statePlaying
		a.show(g.Render())
	case game.KeyEnter:
		if g.Merchant.Active {
			sel := g.Merchant.Selected
			if g.BuySelectedMerchant(sel) {
				g.EndPlayerTurn("")
				a.state = statePlaying
				a.postAction()
			} else {
				a.show(g.RenderMerchantMenu())
			}
		} else {
			a.state = statePlaying
			a.show(g.Render())
		}
	default:
		a.show(g.RenderMerchantMenu())
	}
}

func (a *App) shrineKeys(k game.Key) {
	g := a.g
	if g == nil {
		a.state = statePlaying
		a.show(a.g.Render())
		return
	}
	switch k {
	case game.KeyUp:
		if g.Shrine.Active {
			g.Shrine.Selected--
			if g.Shrine.Selected < 0 {
				g.Shrine.Selected = 3
			}
		}
		a.show(g.RenderShrineMenu())
	case game.KeyDown:
		if g.Shrine.Active {
			g.Shrine.Selected++
			if g.Shrine.Selected > 3 {
				g.Shrine.Selected = 0
			}
		}
		a.show(g.RenderShrineMenu())
	case game.KeyQuit:
		g.CancelShrine()
		a.state = statePlaying
		a.show(g.Render())
	case game.KeyEnter:
		if g.Shrine.Active {
			sel := g.Shrine.Selected
			if sel == 3 {
				g.CancelShrine()
				a.state = statePlaying
				a.show(g.Render())
			} else {
				if g.ExecuteShrineChoice(sel) {
					g.EndPlayerTurn("")
					a.state = statePlaying
					a.postAction()
				} else {
					a.show(g.RenderShrineMenu())
				}
			}
		} else {
			a.state = statePlaying
			a.show(g.Render())
		}
	default:
		a.show(g.RenderShrineMenu())
	}
}

func (a *App) wizardKeys(k game.Key) {
	g := a.g
	if g == nil {
		return
	}
	switch k {
	case game.KeyUp:
		a.wizardState.Move(-1)
		a.show(g.RenderWizardMenu(a.tuning, a.wizardState.Selected))
	case game.KeyDown:
		a.wizardState.Move(1)
		a.show(g.RenderWizardMenu(a.tuning, a.wizardState.Selected))
	case game.KeyQuit:
		a.state = statePlaying
		a.show(g.Render())
		if g.LevelUpPending != nil {
			a.show(g.RenderLevelUp())
		}
	case game.KeyEnter:
		opt := game.WizardOptions[a.wizardState.Selected]
		switch opt.ID {
		case "add_member":
			if len(g.Party.Members) >= 4 {
				g.WizardAddMember()
				a.state = statePlaying
				a.show(g.Render())
				break
			}
			var err error
			a.wizardAddCS, err = game.NewCharSelect()
			if err != nil {
				g.WizardAddMember()
				a.state = statePlaying
				a.show(g.Render())
				break
			}
			a.wizardAddCS.Picks = []string{}
			a.state = stateWizardAddMember
		case "remove_member":
			a.wizardRemoveIdx = g.Party.Selected
			found := -1
			for i, m := range g.Party.Members {
				if m.IsAlive() {
					found = i
					break
				}
			}
			if found >= 0 {
				a.wizardRemoveIdx = found
			}
			a.state = stateWizardRemoveMember
			a.show(g.Render())
		case "resurrect":
			wizardResurrectIdx := g.Party.FirstDead()
			if wizardResurrectIdx < 0 {
				g.Logf("Wizard: Resurrect - no fallen pilgrims")
				a.state = statePlaying
				a.show(g.Render())
				break
			}
			a.wizardRemoveIdx = wizardResurrectIdx
			a.state = stateWizardResurrectMember
			a.show(g.Render())
		case "instant_level", "spawn_loot", "food_1000", "full_heal", "reveal_all":
			g.WizardExecute(opt.ID)
			a.state = statePlaying
			a.show(g.Render())
			if g.LevelUpPending != nil {
				a.show(g.RenderLevelUp())
			}
		default:
			g.WizardExecute(opt.ID)
			a.state = statePlaying
			a.show(g.Render())
		}
	}
}

func (a *App) wizardAddKeys(k game.Key) {
	g := a.g
	if a.wizardAddCS == nil {
		a.state = stateWizard
		if g != nil {
			a.show(g.RenderWizardMenu(a.tuning, a.wizardState.Selected))
		}
		return
	}
	switch k {
	case game.KeyUp:
		a.wizardAddCS.Move(-1)
		a.show(game.RenderCharSelect(a.tuning, a.wizardAddCS))
	case game.KeyDown:
		a.wizardAddCS.Move(1)
		a.show(game.RenderCharSelect(a.tuning, a.wizardAddCS))
	case game.KeyEnter:
		a.wizardAddCS.Select()
		if len(a.wizardAddCS.Picks) > 0 {
			classID := a.wizardAddCS.Picks[0]
			g.SetWizard()
			if len(g.Party.Members) < 4 {
				tmp := game.GeneratePartyWithClasses(g.RNG, []string{classID}, 1)
				if tmp != nil && len(tmp.Members) > 0 {
					m := tmp.Members[0]
					for lvl := 1; lvl < g.Level; lvl++ {
						m.MaxHP += 1 + g.RNG.IntN(2)
						if g.RNG.IntN(2) == 0 {
							m.ATK[0]++
							m.ATK[1]++
						}
						if g.RNG.IntN(4) == 0 {
							m.DEF++
						}
					}
					m.HP = m.MaxHP
					m.Alive = true
					g.Party.Members = append(g.Party.Members, m)
					g.Party.EnsureSelection()
					g.Logf("Wizard: Add Party Member -> %s the %s joined", m.Name, m.Class)
				}
			} else {
				g.Logf("Wizard: Add Party Member - party full")
			}
			a.wizardAddCS = nil
			a.state = statePlaying
			a.show(g.Render())
		}
	case game.KeyQuit:
		if a.wizardAddCS.Back() {
			a.wizardAddCS = nil
			a.state = statePlaying
			if g != nil {
				a.show(g.Render())
			}
		} else {
			a.show(game.RenderCharSelect(a.tuning, a.wizardAddCS))
		}
	}
}

func (a *App) wizardRemoveKeys(k game.Key, living bool) {
	g := a.g
	if g == nil {
		return
	}
	step := func(dir int) {
		for range g.Party.Members {
			a.wizardRemoveIdx += dir
			if a.wizardRemoveIdx < 0 {
				a.wizardRemoveIdx = len(g.Party.Members) - 1
			}
			if a.wizardRemoveIdx >= len(g.Party.Members) {
				a.wizardRemoveIdx = 0
			}
			if g.Party.Members[a.wizardRemoveIdx].IsAlive() == living {
				break
			}
		}
		g.Party.Selected = a.wizardRemoveIdx
		a.show(g.Render())
	}
	switch k {
	case game.KeyUp:
		step(-1)
	case game.KeyDown:
		step(1)
	case game.KeyEnter:
		if living {
			g.WizardRemoveMember(a.wizardRemoveIdx)
		} else {
			g.WizardResurrectMember(a.wizardRemoveIdx)
		}
		a.state = statePlaying
		a.show(g.Render())
		if g.LevelUpPending != nil {
			a.show(g.RenderLevelUp())
		}
	case game.KeyQuit:
		a.state = stateWizard
		a.show(g.RenderWizardMenu(a.tuning, a.wizardState.Selected))
	}
}
