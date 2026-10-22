# Logic Prototype

A single, self-contained HTML file (a **shareable demo**) that lets anyone drive a state model by clicking buttons. Use this when the question is about **business logic, state transitions, or data shape**.

## When this is the right shape

- "I'm not sure if this state machine handles the edge case where X then Y."
- "Does this data model actually let me represent the case where..."
- "I want to feel out what the API should look like before writing it."

## Process

### 1. State the question
Write down what state model and question you're prototyping.

### 2. Isolate the logic in a portable module
Put the logic in a single pure JavaScript module (pure reducer, state machine, or pure functions).

### 3. Build the shareable HTML file
Plain HTML/CSS/JS file for quick interactive testing.
