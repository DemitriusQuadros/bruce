# PRD Example — Influencer Payments SaaS

> This example builds on a business-investor-validator output that scored 75/100 (Green Light)
> for a B2B SaaS platform automating influencer payments and contracts for e-commerce brands.

---

## 📌 Product Overview

**One-liner**: PayFlow helps e-commerce brands automate influencer contracts, payment tracking,
and tax compliance in one place — replacing spreadsheets.

**Problem Statement**: Small and mid-size e-commerce brands running influencer programs manage
payments, contracts, and 1099s manually across spreadsheets, email threads, and PayPal. As
programs scale beyond 10 influencers, this becomes unmanageable and creates compliance risk.

**Solution**: A lightweight SaaS tool where brands can onboard influencers, send contracts,
track deliverables, and trigger payments automatically — with built-in tax form collection and
reporting.

**Primary User**: E-commerce brand owner or marketing manager at a Shopify store doing $500K–$5M
in revenue, running an influencer program with 5–50 creators.

**Stage assumption**: Raw concept — no MVP yet. PRD targets a 6-week first build.

---

## 🎯 Goals & Success Metrics

| Goal | Launch metric | 6-month target |
|------|--------------|----------------|
| Acquire early users | 20 paying brands at launch | 150 paying brands |
| Prove activation | 80% of signups add 1 influencer in week 1 | Same |
| Retention signal | 60% still active at day 30 | 70% monthly retention |
| Revenue | $2K MRR at launch | $15K MRR |
| Reduce manual work | Users report 3+ hrs/week saved | NPS > 40 |

---

## 👤 User Stories

### Theme 1: Onboarding & Setup
- **As a** brand manager, **I want to** connect my payment method once, **so that** I never
  have to re-enter it for each influencer payment.
- **As a** brand manager, **I want to** invite an influencer via email, **so that** they can
  fill in their payment and tax details without me chasing them.

### Theme 2: Core Workflow
- **As a** brand manager, **I want to** send a contract template to an influencer with one click,
  **so that** both parties have a signed agreement before work starts.
- **As a** brand manager, **I want to** mark a deliverable as approved, **so that** payment is
  triggered automatically without manual action.
- **As a** brand manager, **I want to** see all active campaigns and their payment status in one
  dashboard, **so that** I know what's owed and when.

### Theme 3: Compliance & Reporting
- **As a** brand manager, **I want to** collect W-9/W-8 forms from influencers automatically,
  **so that** I'm ready for 1099 filing without scrambling in January.
- **As a** brand manager, **I want to** export a payment report by campaign or time period,
  **so that** my accountant has everything they need.

---

## 🧱 Feature List with MoSCoW Prioritization

| Feature | Priority | Description | Why |
|---------|----------|-------------|-----|
| Influencer onboarding flow | M | Email invite → profile + payment details form | Core to everything else |
| Contract send & e-sign | M | Template-based contract with e-signature | Legal protection from day 1 |
| Payment triggering | M | Manual "approve & pay" button via Stripe Connect | Core value prop |
| Campaign dashboard | M | List of campaigns, influencers, statuses | PM needs visibility |
| Tax form collection (W-9) | M | Auto-request W-9 on onboarding | Compliance is a key pain |
| Payment history & receipts | S | Log of all payments with downloadable receipts | Needed for accounting |
| Deliverable tracking | S | Link posts, mark as delivered/approved | Closes the workflow loop |
| Automated payment on approval | S | Trigger payment when deliverable marked done | Reduces manual steps |
| CSV export / reporting | S | Export payments by campaign or date | Accountant handoff |
| Custom contract templates | C | Let brands create/edit contract templates | Nice but 1 template works for MVP |
| Multi-user brand accounts | C | Invite team members | Single user fine at first |
| Influencer portal | C | Influencer-facing dashboard for their history | Brand-side is priority |
| Xero / QuickBooks sync | W | Direct accounting integration | Too complex for v1 |
| Influencer discovery | W | Find new influencers inside the tool | Different product entirely |

---

## 🗺️ Product Roadmap

### Phase 1 — Core (Weeks 1–6)
**Features**: Influencer onboarding, contract send/sign, manual payment trigger, W-9 collection,
campaign dashboard.

**Milestone**: First brand completes a full cycle — onboards influencer, sends contract,
triggers payment — end to end.

**Risks**: Stripe Connect approval can take 1–2 weeks; apply early. E-sign library choice
(DocuSign API vs HelloSign vs custom) needs a decision in week 1.

---

### Phase 2 — Growth (Weeks 7–14)
**Features**: Deliverable tracking, automated payment on approval, payment history, CSV export,
receipt generation.

**Milestone**: 50 brands onboarded; average brand has 5+ influencers in the system.

**Risks**: Payment automation edge cases (failed payments, partial approvals) need careful
error handling. Don't rush this.

---

### Phase 3 — Scale (Weeks 15+)
**Features**: Custom contract templates, multi-user accounts, influencer portal, deeper reporting.

**Milestone**: $15K MRR; brands refer other brands organically.

**Risks**: Multi-user permissions add complexity — scope carefully before building.

---

## 🔌 Technical Considerations

- **Payments**: Use Stripe Connect (Express accounts) — handles payouts to influencers, 1099-K
  reporting, and compliance. Apply for platform access early.
- **E-signature**: Use HelloSign (Dropbox Sign) API or DocuSign — don't build custom signing.
- **Core entities**: Brand, Campaign, Influencer, Contract, Deliverable, Payment, TaxForm
- **Auth**: Use Clerk or Auth0 — don't build auth from scratch
- **File storage**: S3 or Cloudflare R2 for contracts and tax forms
- **Build vs buy**: Buy e-sign, payments infra, auth. Build campaign management and workflow logic.
- **Scalability flag**: Payment webhook handling must be idempotent from day 1 — duplicate
  payment bugs are catastrophic and hard to fix later.

---

## 🚀 Go-to-Market Steps

1. **Beachhead**: Shopify store owners in the $1M–$5M revenue range, running active influencer
   programs, in the US (W-9 compliance is a US-specific pain that competitors underserve).

2. **First 100 users**: Direct outreach in Shopify Facebook Groups, r/entrepreneur, and
   ecommerce Slack communities. DM founders who post about influencer marketing frustrations.
   Offer free 3-month access in exchange for weekly feedback calls.

3. **Activation hook**: The moment an influencer receives and signs a contract inside the tool.
   Design the onboarding to reach this moment in under 10 minutes.

4. **Monetization moment**: After the first payment is triggered. Charge a flat monthly fee
   ($49/mo for up to 20 influencers) — don't add per-transaction fees at launch to reduce
   friction. Upgrade to $149/mo for unlimited.

5. **Feedback loop**: Weekly 20-min calls with the first 10 brands. Build a Slack channel for
   beta users. Track which steps in the flow cause drop-off via Posthog or Mixpanel.

6. **Growth unlock**: Influencer network effect — when an influencer gets paid through PayFlow
   by one brand, they can share their profile link with other brands. Organic creator-side
   growth drives brand-side demand.

---

## ⚠️ Open Questions & Assumptions

**Assumptions to validate in 30 days**:
1. Are brands willing to pay $49/mo before they've used the product? (Validate with pre-sales)
2. Is W-9/compliance pain strong enough to be the lead value prop, or is it payment speed?
3. Do influencers resist sharing payment details with a new tool, or is it frictionless?
4. Is the primary buyer the brand owner or a marketing manager? (Affects sales motion)
5. Is Shopify the right beachhead, or do WooCommerce/Amazon brands have the same pain?

**Deliberate decisions (revisit if wrong)**:
- Assumed US-only for v1 due to W-9 framing — revisit if international demand appears early
- Assumed single brand user — revisit if first customers are agencies managing multiple brands
- Assumed flat fee pricing — revisit if transaction volume varies wildly between customers
