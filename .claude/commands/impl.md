# /project:impl — GitHub Issue Implementation

## Purpose
Read a GitHub Issue and implement it.

## CRITICAL RULES
- Do not implement anything beyond the scope of the Issue
- After implementation, do not close the Issue automatically

## Input
Read the Issue number from $ARGUMENTS.
Run: gh issue view <number>
Parse the Implementation Plan checklist from the Issue body.

## Phase 1: Confirmation
Summarize what you are about to implement.
Ask me any question if you feel any ambiguity, risk, or missing consideration.
Ask me "Start Implementation?"

## Phase 2: Implementation
After user confirms, implement according to the Issue's Implementation Plan.
Check off each item mentally as you go.

## Phase 3: Summary
Output a summary of all changes:

### Summary
- Changed:
- Implemented: 
- More considerations (if exists): 

Ask: "Close the issue?"
If yes: gh issue close <number>