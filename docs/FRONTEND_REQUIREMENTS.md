# Frontend Requirements & Design Guidelines

This document outlines the specific UI/UX requirements and implementation details established for the frontend. Future updates or visual refreshes must adhere to these guidelines to prevent regressions.

## 1. Model Cards

### Layout & Styling
- **Medals**: Must be **centered** horizontally at the top of the card (`left: 50%`, `transform: translateX(-50%)`).
- **Rationale**:
  - Must be visually separated from the answer text (e.g., top border).
  - Style: Italicized, smaller font, muted color.
  - Position: Aligned to the **bottom** of the card content area.
  - Height: Max height of approx. 8.5 lines with scroll.
- **Error State**:
  - If a model response starts with "Error:":
    - Card border must turn **red**.
    - Response text must turn **red**.
  - State must automatically clear (recover) if the next round is successful.

### Interaction
- **Swapping**: Interactive swapping (clicking a gallery card to swap with hero) is **DISABLED**. The layout is static after the winner is announced.
- **Round Navigation**: Users must be able to navigate back to *any* round up to the current active round, even if it is not yet completed (filled).

## 2. Model Selector

- **Visibility**: The dropdown itself should be invisible (opacity 0) but cover the model name/arrow area to trigger the native OS picker.
- **Display**:
  - Show only the **Model Name** and a small **Down Arrow**.
  - **Provider Badges** (e.g., xAI, OpenAI) must be visible.
  - **Costs** in the header must be **hidden** (display: none).
- **Dropdown Options**: Must always show full details: `Model Name ($Price)`.

## 3. Cost Display

- **Visibility**: Cost indicators must be **completely hidden** (`display: none`) by default.
- **Behavior**: They should only appear when a non-zero cost is available. Empty placeholders or artifacts are not allowed.

## 4. Discussion Section

- **Bubble Alignment**:
  - Alignment must be **consistent** based on the model pair, not message order.
  - For a pair "ModelA-ModelB":
    - ModelA messages always align **Left**.
    - ModelB messages always align **Right**.
  - Do *not* use `:nth-child` for alignment.

## 5. Input Area

- **Random Question Button**:
  - Position: **Top-Right** corner of the text area.
  - Must overlap the text area content slightly.
  - Must not interfere with the "Launch Discussion" button.

## 6. General

- **Newlines**: Markdown rendering must preserve newlines in model answers.
