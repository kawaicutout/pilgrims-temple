# Gameplay Loop — Party Roguelike

Working title: not yet chosen. This document distills the systems in DESIGN.md
into the loops a player actually navigates. It captures the decision points, not
the rules. The rules remain authoritative in DESIGN.md.

## 1. The core loop

Each run descends through eight floors (default 8) to retrieve the relic at the
base. Between the first floor and the relic, the player repeats one activity:

> Descend → explore the floor → recruit and gain → survive the party.

The party is the resource. The player spends it on risk, gains it from
recruitment features, and loses it irreversibly on member death. Every floor
asks the same question with a different shape: how much of the party is this
floor worth.

Each run races two clocks: **food**, which decides how much time the party
has, and **health**, which decides how much damage it can afford. Party
composition is how the player defends against both at once.

| Stage | What the player does | What changes |
|---|---|---|
| Descend | Moves deeper, chooses direction | Floor difficulty and reward scale up |
| Explore | Maps the floor, finds stairs and features | Knowledge; encounter risk rises with exposure |
| Recruit and gain | Claims shrines, prisons, and upgrades | Party size and power increase, up to 4 members |
| Survive | Fights enemy parties, uses items, rests | The run continues or ends; health drains against the food clock |

## 2. The turn loop

The world advances one actor at a time. One member of each party acts per turn.

The player resolves each turn in the same order:

1. **Select** the acting member (q/w/e/r, free).
2. **Act**: move, bump attack, use an ability, use an item, pick up, interact,
   or wait.
3. **Read the response**: the world acts back, weighted at the active member
   (Section 6.2 of DESIGN.md).
4. **Adjust**: change selection, reposition, or press on.

Selection is free; the action costs the turn. The twist is that selection and
the action both matter. Moving with a fragile member exposes that member to the
next attack. The player chooses, each turn, who pays for the party's progress.

## 3. The decision that defines a turn

Acting draws attacks toward the active member. This concentrates the entire risk
trade in one sentence:

> The player trades progress for exposure, and chooses which member carries that
> exposure every turn.

Under the active-weighted targeting rule, the acting member takes half of all
incoming single-target damage plus an even share of the rest. With four living
members, the actor absorbs 62.5 percent of hits; the other three split 37.5
percent. Acting is a constant drain on the actor.

The result is a party that behaves like an inventory: the player stops thinking
about individual characters as heroes and starts thinking about them as damage
sponges, healers, and ability packages. Spending a turn is spending the member
who acts.

### The four mitigation levers

The player controls exposure through four levers, in rough order of cost:

1. **Act with the durable member.** Take hits on the member built to absorb
   them. Cheap, but the durable member is not always the one whose ability the
   floor needs.
2. **Act with the non-critical member.** Keep a member alive because removing
   its passive would break the party. Cheaper than losing it; still a tax on
   the acting member's health.
3. **End fights faster.** Fewer turns means fewer enemy actions, so fewer hits,
   wherever they land. Offense is a defense against the weighting rule.
4. **Leave the fight.** Fall back, break line of sight, or skip a room. Cheapest,
   but it trades floor progress for safety.

### The composition lever

A separate lever operates across the whole run, not turn to turn: which classes
are in the party. Each class pushes back on one of the two pressures.

- A **fighter** resists hits, lowering the health cost per fight.
- A **cleric** heals over time, lowering the need to rest and the need for
  healing items. Its passive healing only scales if the player spends a talent
  on it, so it is the exception, not the norm.
- A **rogue** or **wizard** opens locked routes and cheapens the dungeon,
  lowering the food and item cost of clearing a floor.
- A **druid** reduces food consumption, buying time on the food clock.
- A **bard** is a meta-class: it boosts the other members and mitigates
  neither pressure directly, multiplying whatever mix is already in the party.

Class passives stack. Most classes mitigate one pressure directly; the bard
mitigates neither directly but multiplies the others. Because party slots and
the food clock are both limited, composition is a budget: the player spends
slots on a mix that survives the run's health pressure inside its food budget.
See Section 5.5 of DESIGN.md.

## 4. How characters double as equipment

The party has two run-level resources, one per member, and a set of per-member
resources.

| Resource | Scope | Pressure | What drives it |
|---|---|---|---|
| Food | Shared clock and ration pool | Time limit | Consumed per world turn (a full round), scaled by member count. Smaller parties run longer |
| Health | Per member | Risk limit | Lost on hits, weighted at the active member; recharges slowly per world turn |
| Class ability and passives | Per member | Mitigation | Which of the two pressures a member softens (Section 3) |
| Talents and affixes | Per member | — | Define an acting member's options and damage profile |
| XP and level | Shared party pool | — | Levels boost each living member; unlock talent picks |
| Items | Shared pack | — | Consumables and one-shot permanent upgrades |
| Gold | Shared purse | — | Found with loot (enemy drops and dungeon finds); spent at merchants |

Food is the run's budget; health is the run's mistake currency. A larger party
is stronger per turn but spends food faster, so the player is always paying the
food clock for power. Power lives in the members, not the inventory.Losing a member removes its class passive, talents, affixes, and applied upgrades at once. That is the permanent consequence: a character is a bundled gear set, and death is losing the bundle. The sole exception is a scarce shrine that restores a dead member fully — class, talents, affixes, and upgrades — at a cost (Section 4.2 of DESIGN.md); it is a recovery mechanic, not the norm.

## 5. Decision density per minute

Roguelikes are judged by how much the player decides per action. The layout is
designed to keep that density high without scaling cost with party size:

- Party size changes the number of options per turn, not the number of actions
  per turn. A party of four does not play four times as fast.
- Enemy parties cost nothing extra to run: one actor per enemy party per turn,
  chosen by AI.
- Selection is free and separate from acting, so browsing options never costs
  a turn.

The goal is a party game without the turn economy of a party game.

## 6. Boundary conditions

- **Level up (default):** each living member gains a small random attribute
  boost; each living member has a 25% chance to receive a talent pick (the
  player chooses from that member's pool). Occasionally a pick is an affix
  gain instead (tunable cadence).
- **Recruitment:** found members enter at the current party level, so they are
  useful immediately. The cap is four.
- **Resurrection:** shrines are chance-placed per floor — a run can see none.
  They recruit a new member or, at a cost randomized per shrine (gold, food,
  or both), restore a dead member: class, talents, affixes, and upgrades
  return, attributes re-scale to the party level, and level-up awards missed
  while dead are not granted.
- **Food:** the party shares one clock and one ration pool. Cost scales with
  living members; resting spends the same per-turn upkeep for its turns, so
  party size trades power against time (Section 3.5 of DESIGN.md).
- **Rest:** a batch of 10 wait turns at an accelerated heal rate; the batch
  ends early only when a hostile appears or hunger advances, crediting the
  completed turns. Wandering encounters are the risk. Rest heals slowly and
  does not scale with party level, so late-game runs drift toward healing
  items and healers.
- **Solo runs:** a carefully managed single member consumes the least food and
  takes no shared-clock pressure, which makes a solo victory possible.
- **Failure:** the run ends when all members die. The death screen shows the
  seed and the score.