---
name: design-an-interface
description: Build, prototype, and design interfaces (UI/UX layout variants & state machine logic). Use when the user wants to sanity-check whether a state model or logic feels right, or explore what a UI should look like.
---

# Design an Interface & Prototype

An interface design prototype is **throwaway code that answers a design question**. The question decides the shape.

## Pick a branch

Identify which question is being answered, using the user's prompt, the surrounding code, or by asking if the user is around:

- **"What should this look like?"** → [UI.md](UI.md). Generate several radically different UI variations on a single route, switchable via a URL search param and a floating bottom bar.
- **"Does this logic / state model feel right?"** → [LOGIC.md](LOGIC.md). Build a single shareable HTML file (free-play buttons plus tabbed guided walkthroughs) that pushes the state machine through cases that are hard to reason about on paper.

## Rules that apply to both

1. **Throwaway from day one, and clearly marked as such.** Locate the prototype code close to where it will actually be used so context is obvious.
2. **Prioritize Visual Excellence & Rich Aesthetics:** Follow modern web design best practices (HSL tailored colors, sleek dark modes, glassmorphism, dynamic animations, modern typography from Google Fonts).
