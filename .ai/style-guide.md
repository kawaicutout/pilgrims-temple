# Style Guide — Party Roguelike

Scope: design documents, technical notes, and user-facing text for this project.

## Domain: technical writing

- Active voice. Passive voice only when the actor is unknown or irrelevant.
- Present tense for system descriptions. Future tense only for planned milestones.
- Third person. Second person only in instructions and procedures.
- No contractions.
- Sentences of 10–20 words. One idea per paragraph, with a topic sentence.
- No filler ("very", "just", "that" as filler), no redundancies, no vague qualifiers.
- Specific numbers over approximations. State defaults and mark them adjustable, e.g. "(default 8 floors)".
- Expand an acronym on first use: WebAssembly (WASM).
- Name the subject. Do not open a sentence with an unclear "it" or "this".
- Opinion-free, emotion-free. State rationale as a cause and effect, not a preference.

## Project terminology

- **party**: one to four characters occupying a single tile.
- **member**: one character within a party.
- **selected member**: the member the player cursor points at; a UI state that costs nothing.
- **active member**: the member who performed the party's most recent action; the target of the 50% weighting rule. For enemy parties, the most recent actor.
- **enemy party**: any non-player group. Enemy parties follow the same rules as the player party.
- Do not use "blob" in formal documents. Do not use "rank" or "front/rear": the design has no ranks.

## Formatting

- Markdown with numbered sections for design documents.
- Tables for enumerated values (weights, key maps, milestones).
- Tunable values appear inline as defaults, collected in a single "Open Questions" section.
