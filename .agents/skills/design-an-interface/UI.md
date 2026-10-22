# UI Prototype & Interface Design

Generate **several radically different UI variations** on a single route, switchable from a floating bottom bar. The user flips between variants in the browser, picks one (or steals bits from each), then throws the rest away.

If the question is about logic/state rather than what something looks like, use [LOGIC.md](LOGIC.md).

## When this is the right shape

- "What should this page look like?"
- "I want to see a few options for this dashboard before committing."
- "Try a different layout for the settings screen."

## Process

### 1. State the question and pick N
Default to **3 variants**. Write down the plan in a top-of-file comment or prototype intro.

### 2. Generate radically different variants
Draft each variant:
- Page purpose & accessible data.
- Project styling system / CSS architecture.
- Exported component name (e.g. `VariantA`, `VariantB`, `VariantC`).
