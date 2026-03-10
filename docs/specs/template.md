# Spec: [Feature / Module Name]

> **Skill target:** software-architect  
> **Status:** Draft | In Review | Approved | Implemented  
> **Version:** 0.1  
> **Last updated:** YYYY-MM-DD  
> **Author:** [name]

---

## 1. Purpose

One or two sentences. What does this feature/module do and why does it exist?

> Example: *"Handles payout processing to influencers after a deliverable is approved. Replaces the manual bank-transfer workflow currently done outside the platform."*

---

## 2. Scope

### In scope
- What this spec covers

### Out of scope
- What this spec explicitly does NOT cover (avoids scope creep)

---

## 3. User Stories

Who uses this, and what do they need?

```
As a [role], I want to [action] so that [outcome].

Example:
As a BrandOwner, I want to trigger a payout after approving a deliverable
so that influencers are paid automatically without manual steps.
```

List all roles that interact with this feature.

---

## 4. Functional Requirements

What the system must do. Be explicit — no ambiguity.

```
REQ-001: [Short label]
  Description: [What must happen]
  Trigger: [What initiates this]
  Outcome: [What the system state looks like after]

REQ-002: ...
```

---

## 5. API Contract

For each endpoint this feature introduces or modifies:

```
[METHOD] /api/[resource]/[action]
  Auth: required ([role]) | public
  Request body: { field: type, ... }
  Response: { field: type, ... }
  Errors:
    - 400: [validation reason]
    - 401: unauthenticated
    - 403: forbidden (wrong role)
    - 404: resource not found
    - 409: [conflict reason]
  Side effects: [e.g. "triggers SendPayoutEmail job"]
  Notes: [any important constraints]
```

---

## 6. Data Model Changes

List all new entities, new fields on existing entities, or schema changes.

```
New entity: [EntityName]
| Field       | Type      | Notes                        |
|-------------|-----------|------------------------------|
| id          | UUID      | PK                           |
| [field]     | [type]    | [constraint / FK / index]    |

Modified entity: [ExistingEntity]
  + [new_field]: [type] — [why it's added]
  ~ [changed_field]: [old type → new type] — [migration note]
```

Relationships to update (if any):
```
[Entity] 1:N [Entity] — [description]
```

---

## 7. Background Jobs

List any async operations this feature requires.

```
Job: [JobName]
  Trigger: [what enqueues this job]
  Input: [fields passed to worker]
  Steps:
    1. [first action]
    2. [second action]
    3. ...
  Retry: [N]x with [backoff strategy]
  Failure: [what happens if all retries fail]
  Idempotency: [how duplicate runs are prevented]
```

---

## 8. Behavior: Normal Flow

Walk through the happy path step by step.

```
1. [Actor] does [action]
2. System responds with [outcome]
3. [Job / side effect] is triggered
4. ...
5. Final state: [describe DB/system state]
```

---

## 9. Behavior: Error Cases

For each known failure mode:

```
Case: [short label]
  Condition: [what triggers this error]
  Expected response: [HTTP code + error payload]
  System state: [is anything persisted? rolled back?]
  User-visible impact: [what the user sees/experiences]
```

---

## 10. Edge Cases

Specific inputs or conditions that need explicit handling:

- [ ] What happens if [value] is null or undefined?
- [ ] What happens if [resource] is already in [status]?
- [ ] What happens if [external service] is unavailable?
- [ ] What happens if [user] triggers this twice in rapid succession?
- [ ] What is the behavior at boundary values (0, max, empty list)?

---

## 11. Invariants

Things that must ALWAYS be true. These are hard constraints the implementation must never violate.

- `[Entity].[field]` is NEVER mutated after [event]
- [Operation] MUST be idempotent — duplicate calls produce no additional side effects
- [Resource] MUST only be accessible to [role] scoped to their `[org/brand]_id`
- [Sensitive field] is NEVER returned in API responses
- [Financial/legal record] is NEVER hard-deleted

---

## 12. Auth & Permissions

Which roles can access this feature, and with what permissions?

```
| Action              | [Role A] | [Role B] | [Role C] |
|---------------------|----------|----------|----------|
| [action 1]          | ✓        | ✗        | ✗        |
| [action 2]          | ✓        | ✓        | ✗        |
| [action 3]          | ✓        | ✗        | read-own |
```

---

## 13. External Integrations

If this feature touches any external service:

```
Integration: [Service name]
  Purpose: [what it does in this feature]
  Pattern: [REST outbound | webhook inbound | SDK | OAuth]
  Key concerns:
    - [idempotency / credential storage / rate limits / etc.]
  Failure mode: [what the system does if this integration is unavailable]
```

---

## 14. Architecture Decision Records (ADRs)

Document decisions made while writing this spec. Not required for small features.

```
ADR-[NNN]: [Short label]
  Decision: [what was decided]
  Alternatives considered: [what else was on the table]
  Rationale: [why this option was chosen]
  Consequences: [what this decision means for the build]
```

---

## 15. Open Questions

Questions that must be answered before implementation begins. Each blocks a specific step.

```
Q: [Question]
  Blocks: [what can't be built until this is resolved]
  Decision needed by: [Phase / Step / date]
  Owner: [who decides]
```

---

## 16. Risks

Top risks to this feature's implementation.

```
Risk: [Short label]
  Impact: [what goes wrong if this happens]
  Likelihood: Low | Medium | High
  Mitigation: [how to prevent or recover]
```

---

## 17. Acceptance Criteria

The definition of "done" for this spec. Each item should be independently verifiable.

- [ ] `[Specific, testable behavior]`
- [ ] All error cases return the correct HTTP status + error code
- [ ] Relevant background jobs are enqueued (not just synchronous)
- [ ] Audit log is written for [financial/legal] state changes
- [ ] [Role] cannot access [resource] belonging to another [org/brand]
- [ ] All edge cases listed in §10 have test coverage

---

## Changelog

| Version | Date       | Author  | Summary                    |
|---------|------------|---------|----------------------------|
| 0.1     | YYYY-MM-DD | [name]  | Initial draft              |