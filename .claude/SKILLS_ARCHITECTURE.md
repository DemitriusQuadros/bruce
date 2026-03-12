# Skills Architecture: Delegation Pattern v1.0

## Overview

This document describes the unified skills architecture used in your Claude Code environment. All 6 SDD (Spec-Driven Development) pipeline skills follow the **Skills → Agents Delegation Pattern**.

---

## The Pattern

```
User Command          Local Skill             Cloud/Remote Agent
    │                    │                          │
    ├─ /go-backend-dev → go-backend-dev/SKILL.md → go-backend-dev agent
    │                    (entry point)             (execution)
    │
    ├─ /software-architect → software-architect/SKILL.md → software-architect agent
    │
    ├─ /product-manager-prd → product-manager-prd/SKILL.md → product-manager-prd agent
    │
    ├─ /business-investor-validator → business-investor-validator/SKILL.md → business-investor-validator agent
    │
    ├─ /frontend-specialist → frontend-specialist/SKILL.md → frontend-specialist agent
    │
    └─ /qa-specialist → qa-specialist/SKILL.md → qa-specialist agent
```

### Key Points

1. **Skills** are thin entry points:
   - Located in `~/.claude/skills/<skill-name>/SKILL.md`
   - Include full documentation and training material for local context
   - Tagged with `agent: <agent-name>` and `delegation: true` in frontmatter
   - Allow users to invoke functionality via slash commands (e.g., `/go-backend-dev`)

2. **Agents** are specialized execution engines:
   - Live in the cloud or as registered agent types
   - Invoked by skills when the user requires specialized execution
   - Have full access to codebase, tools, and autonomous execution capabilities
   - Include their own training material and system prompts

3. **Single Source of Truth**:
   - Skill content is maintained locally for discoverability and context
   - Agent behavior is maintained centrally for consistency
   - Both reference the same canonical implementation

---

## SDD Pipeline (Step-by-Step)

```
Step 1: Validate the idea
  Command: /business-investor-validator
  Input: Raw business idea or concept
  Output: Validation scorecard with strengths, risks, and clarity

Step 2: Create product spec
  Command: /product-manager-prd
  Input: Validated idea (optional; raw ideas also work)
  Output: Full PRD with features, user stories, roadmap

Step 3: Design the system
  Command: /software-architect
  Input: PRD from Step 2
  Output: Technical blueprint (architecture, database, API, implementation plan)

Step 4: Implement the backend
  Command: /go-backend-dev
  Input: Technical blueprint from Step 3
  Output: Working Go backend code for the project

Step 5: Build the frontend
  Command: /frontend-specialist
  Input: API contracts from Step 4
  Output: UI pages, components, and full frontend application

Step 6: Write end-to-end tests
  Command: /qa-specialist
  Input: Specs from Step 3 and API from Step 4
  Output: BDD test scenarios in Gherkin + godog implementation
```

---

## When to Use Each Skill

| Skill | When to Use | Example |
|---|---|---|
| `business-investor-validator` | Validate a new business idea | "validate my SaaS concept" |
| `product-manager-prd` | Turn validated idea into specs | "build a PRD from my validation" |
| `software-architect` | Design technical system from PRD | "architect this PRD" |
| `go-backend-dev` | Implement backend from architecture | "create the user service module" |
| `frontend-specialist` | Build UI against the API | "build a dashboard" |
| `qa-specialist` | Write BDD tests from specs | "generate E2E tests for this feature" |

---

## Implementation Details

### Skill Frontmatter

Each SKILL.md now includes delegation metadata:

```yaml
---
name: go-backend-dev
description: >
  Senior Go backend developer agent...
agent: go-backend-dev
delegation: true
---
```

The `agent` field specifies which agent to invoke when the skill is used.

### Benefits of This Pattern

✅ **Single codebase** — No content duplication between skill and agent
✅ **Discoverability** — Slash commands make skills easy to find
✅ **Rich context** — Local SKILL.md provides training and references
✅ **Scalability** — Agents can be updated independently
✅ **Flexibility** — Skills can be used locally or delegate to cloud agents
✅ **Consistency** — Both share the same implementation model

---

## Configuration

### Settings

- **Global settings**: `~/.claude/settings.json` (unchanged)
- **Skills directory**: `~/.claude/skills/` (6 SDD skills)
- **Agent metadata**: Each SKILL.md has `agent:` and `delegation:` fields

### Future Enhancements

- Skills can be used both standalone (local execution) and delegated (agent execution)
- Cloud-based agent registry can be connected for remote execution
- Per-skill configuration can be added to `settings.json` if needed

---

## Testing the Setup

Verify the delegation pattern is working:

```bash
# Check skill metadata
grep -A2 "^agent:" ~/.claude/skills/*/SKILL.md

# Should output:
# ~/.claude/skills/go-backend-dev/SKILL.md-agent: go-backend-dev
# ~/.claude/skills/software-architect/SKILL.md-agent: software-architect
# ... (6 total)
```

---

## References

- **Spec-Driven Development (SDD)**: Full pipeline from idea → validation → PRD → architecture → implementation → testing
- **Skills System**: Claude Code slash commands at `~/.claude/skills/`
- **Agents**: Specialized execution engines with autonomous capabilities
- **Delegation Pattern**: Skills invoke agents for specialized execution

---

**Last Updated**: 2026-03-12
**Version**: 1.0
**Status**: Production Ready
