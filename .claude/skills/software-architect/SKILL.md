---
name: software-architect
description: >
  Act as a senior software architect to transform a Product Requirements Document (PRD) into a
  complete, implementation-ready technical architecture. Use this skill whenever someone has a
  PRD, product spec, or detailed feature list and wants to know how to technically implement it.
agent: software-architect
delegation: true
---

# Software Architect Skill

**This skill is a delegation point to the `software-architect` agent.**

When you invoke this skill (e.g., `/software-architect`), it launches a specialized agent that
transforms product requirements into complete, implementation-ready technical blueprints with
system architecture, database design, API contracts, and step-by-step implementation plans.

## When to use

- Transform a PRD into a technical architecture
- Design system architecture and database schema
- Create API contracts and integration architecture
- Plan implementation in phases
- Document architectural decisions (ADRs)
- Identify technical risks and mitigations

## How to invoke

Type `/software-architect` followed by your requirement:

```
/software-architect architect this PRD for me
/software-architect turn this product spec into a technical plan
/software-architect design the database and API for this product
```

The agent will produce:
- System architecture overview (Mermaid diagram)
- Core domain model (entities and relationships)
- Database schema design (ERD)
- API contract definitions
- Background jobs and async flows
- Auth & authorization model
- Step-by-step implementation plan
- Integration architecture
- Architecture decision records (ADRs)
- Technical risks and open questions

---

## Pipeline Context

This skill is **step 3** of the SDD (Spec-Driven Development) pipeline:

```
business-investor-validator → product-manager-prd → software-architect → go-backend-dev → frontend-specialist → qa-specialist
```

If you have a validated business idea from `/business-investor-validator`, pass it to
`/product-manager-prd` to create a PRD, then hand off the PRD to this skill.
