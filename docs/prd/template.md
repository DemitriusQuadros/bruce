# PRD: [Product Name]

> **Skill target:** product-manager-prd  
> **Status:** Draft | In Review | Approved  
> **Version:** 0.1  
> **Last updated:** YYYY-MM-DD  
> **Author:** [name]  
> **Input source:** Raw idea | business-investor-validator output

---

## 📌 1. Product Overview

**One-liner**: [Single sentence — what the product does and for whom.]

**Problem Statement**: [2–3 sentences. Name the user, the specific friction, and the current workaround.]

**Solution**: [2–3 sentences. What the product does to solve it. No buzzwords.]

**Primary User**: [One sentence — role, context, and daily goal of the main user.]

**Stage assumption**: [Raw concept | MVP | Seed-stage. State assumed stage and what that means for scope.]

---

## 🎯 2. Goals & Success Metrics

| Goal | Launch metric | 6-month target |
|------|---------------|----------------|
| [e.g. Acquire early users] | [e.g. 20 paying customers] | [e.g. 150 paying customers] |
| [e.g. Prove activation] | [e.g. 80% complete core action in week 1] | [same or higher] |
| [e.g. Retention signal] | [e.g. 60% active at day 30] | [e.g. 70% monthly retention] |
| [e.g. Revenue] | [e.g. $2K MRR] | [e.g. $15K MRR] |
| [e.g. Reduce user pain] | [e.g. Users report X hrs/week saved] | [e.g. NPS > 40] |

> Focus on outcomes (retention, activation, revenue) — not outputs (features shipped).

---

## 👤 3. User Stories

Group into 2–3 themes. Each story should be specific enough for a developer to estimate.

### Theme 1: [e.g. Onboarding & Setup]

- **As a** [user type], **I want to** [action], **so that** [outcome].
- **As a** [user type], **I want to** [action], **so that** [outcome].

### Theme 2: [e.g. Core Workflow]

- **As a** [user type], **I want to** [action], **so that** [outcome].
- **As a** [user type], **I want to** [action], **so that** [outcome].
- **As a** [user type], **I want to** [action], **so that** [outcome].

### Theme 3: [e.g. Reporting & Admin]

- **As a** [user type], **I want to** [action], **so that** [outcome].
- **As a** [user type], **I want to** [action], **so that** [outcome].

---

## 🧱 4. Feature List with MoSCoW Prioritization

| Feature | Priority | Description | Why |
|---------|----------|-------------|-----|
| [Feature name] | M | [What it does] | [Why it's core] |
| [Feature name] | M | [What it does] | [Why it's core] |
| [Feature name] | S | [What it does] | [Why it's high value but not blocking] |
| [Feature name] | S | [What it does] | [Why it's high value but not blocking] |
| [Feature name] | C | [What it does] | [Why it's a nice-to-have] |
| [Feature name] | W | [What it does] | [Why it's explicitly out of scope for v1] |

**Priority key:**
- **M — Must Have**: Core to the product working at all. Ship on day 1.
- **S — Should Have**: High value, product works without it. Ship in v1.1.
- **C — Could Have**: Nice to have. Queue for later sprints.
- **W — Won't Have (now)**: Explicitly out of scope. Acknowledge and park.

---

## 🗺️ 5. Product Roadmap

### Phase 1 — Core (Weeks 1–6)

**Features included:**
- [Must-Have feature 1]
- [Must-Have feature 2]
- [Must-Have feature 3]

**Milestone / success signal**: [The smallest shippable thing that proves the core value proposition.]

**Risks to watch:**
- [Risk 1 — e.g. third-party API approval timelines]
- [Risk 2 — e.g. key technical decision needed in week 1]

---

### Phase 2 — Growth (Weeks 7–14)

**Features included:**
- [Should-Have feature 1]
- [Should-Have feature 2]
- [Should-Have feature 3]

**Milestone / success signal**: [What "sticky and complete" looks like — user count, engagement metric, or revenue.]

**Risks to watch:**
- [Risk 1 — e.g. edge cases in automated flows]
- [Risk 2]

---

### Phase 3 — Scale (Weeks 15+)

**Features included:**
- [Could-Have feature 1]
- [Could-Have feature 2]
- [Optimization or expansion]

**Milestone / success signal**: [What unlocks the next order of magnitude — revenue, users, or market expansion.]

**Risks to watch:**
- [Risk 1 — e.g. permissions complexity, multi-tenancy]
- [Risk 2]

---

## 🔌 6. Technical Considerations

High-level flags only — point to decisions, not answers. 5–8 bullets max.

- **Key integrations**: [e.g. Stripe for payments, Auth0 for auth, SendGrid for email]
- **Core entities**: [e.g. User, Organization, Campaign, Invoice, Report]
- **Build vs buy**: [e.g. Buy auth, payments infra, email. Build core workflow logic.]
- **Scalability flags**: [e.g. Webhook handling must be idempotent from day 1 — hard to fix later]
- **Data model note**: [Any entity relationship worth flagging early]
- **Platform decision**: [e.g. Web-first vs mobile-first, and why]

---

## 🚀 7. Go-to-Market Steps

1. **Beachhead**: [Name the exact first segment — geography, company size, persona. Be specific.]

2. **First 100 users**: [Single channel for early acquisition — community, outreach, waitlist, partnership. One channel only.]

3. **Activation hook**: [The one action a new user must complete to experience core value. Design the onboarding around it.]

4. **Monetization moment**: [When and how does the first dollar get charged? Name the trigger and the price.]

5. **Feedback loop**: [How you collect structured feedback from early users to inform Phase 2.]

6. **Growth unlock**: [The one lever that, if it works, drives non-linear growth — referrals, SEO, integrations, word-of-mouth.]

---

## ⚠️ 8. Open Questions & Assumptions

### Assumptions to validate in the next 30 days

1. [Question that, if wrong, changes the product significantly]
2. [Question about user behavior or willingness to pay]
3. [Question about the beachhead segment or channel]
4. [Question about pricing model or packaging]
5. [Question about build vs buy or a key technical assumption]

### Deliberate decisions (revisit if wrong)

- Assumed [X] — revisit if [early signal that suggests otherwise]
- Assumed [Y] — revisit if [early signal that suggests otherwise]
- Assumed [Z] — revisit if [early signal that suggests otherwise]

---

## Changelog

| Version | Date       | Author  | Summary                    |
|---------|------------|---------|----------------------------|
| 0.1     | YYYY-MM-DD | [name]  | Initial draft              |