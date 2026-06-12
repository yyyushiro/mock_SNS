# /project:refine — Spec Refinement & GitHub Issue Creation

## Purpose
Refine a spec through dialogue until all ambiguities are resolved,
then create a GitHub Issue.

## CRITICAL RULES
- Do NOT write or modify any code at any point in this flow
- Do NOT create a GitHub Issue without explicit user approval

## Input
Read the spec from $ARGUMENTS (e.g. @spec.md).
If no file is provided, ask for the following one by one:
1. 現在の問題
2. 解決策
3. 具体的な実装案

## Phase 1: Understanding & Detection
Identify ambiguities, risks, and missing considerations and list them.
Do NOT propose any solution at this phase.

## Phase 2: Refinement
I propose solutions to problems, so judge if they are feasible and effective.
Regardless of my solutions are good or bad, refine the solutions in detail.

Repeat this process until we find the good solutions to all problems.

## Phase 3: Issue Creation Approval
When all questions are resolved, generate the Issue in this format:

---
**Title:** 〇〇〇

**Background** (if needed)

**Solution**

**Implementation Plan**
(In checklist)

**Considerations**
---

Then ask if you can post it.

## Phase 4: Issue Posting
After user confirms, run:
gh issue create --title "<title>" --body "<body>"

Report the created Issue URL to the user and end the session.