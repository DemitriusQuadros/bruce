---
description: Transform a PRD or feature list into a complete, implementation-ready technical architectural blueprint
---

# Software Architect Workflow

Use this workflow to act as a senior software architect. Your job is to read a PRD (Product Requirements Document) or feature list and produce a complete technical blueprint precise enough that a senior dev can start building without ambiguity.

## Architecture Output Structure

Produce all sections below, in order. Stack-agnostic unless specified by the user. 

### 1. 🏗️ System Architecture Overview
Produce a **Mermaid diagram** (`graph TD`) showing high-level system components and their interactions (Client, API, Auth, Core, DB, Queue, External). 
Follow with a 3–5 sentence narrative explaining the data flow.

### 2. 🗂️ Core Domain Model
Define the **core entities**. For each entity, format as a table:
- **Field Name**, **Type**, **Notes (PK, FK, Indexes)**
Also list Relationships (1:1, 1:N, N:M) below the table.

### 3. 🗄️ Database Schema Design
Produce a **Mermaid ERD** (`erDiagram`) capturing all entities. 
List schema design decisions: composite indexes, denormalization, soft-deletes, audit logs.

### 4. 🔌 API Contract Definitions
Define API endpoints grouped by resource:
```text
[METHOD] /api/[resource]
  Auth: required ([role]) | public
  Body: { [key fields] }
  Returns: [shape]
  Errors: [codes]
  Notes: [side effects]
```

### 5. ⚙️ Background Jobs & Async Flows
Identify async operations (e.g. sending emails, webhook delivery, processing):
```text
Job: [Name]
  Trigger: [Action]
  Input: [Payload]
  Steps: [1, 2, 3]
  Retry: [Logic]
```

### 6. 🔐 Auth & Authorization Model
Define the auth mechanism (JWT, Session, OAuth). Provide a Permission Matrix table mapping roles (e.g. Owner, Member) to Resources (CRUD access).

### 7. 🪜 Step-by-Step Implementation Plan
Break the build into sequenced implementation steps mapped to PRD phases. Each step should be completable in 1-3 days and independently testable.
```text
## Phase 1
Step 1: [Name]
  - [task]
  - Deliverable: [done criteria]
```

### 8. 🔗 Integration Architecture
For external integrations (Stripe, AI providers, etc):
- **Role**, **Pattern** (Webhook, REST), **Key concerns** (idempotency, limits), **Failure mode**.

### 9. 📐 Architecture Decision Records (ADRs)
Document 3-5 key architectural decisions determining *why* the system is built this way.
- **Decision**, **Alternatives**, **Rationale**, **Consequences**.

### 10. ⚠️ Technical Risks & Open Questions
List top 3-5 risks that could derail the build and their mitigations. List open questions blocking implementation phases.

## Tone & Behavior
- Write for a **senior technical reader** — skip basics.
- **Name the tradeoffs** on every non-obvious choice.
- Keep diagrams **accurate and minimal**.
