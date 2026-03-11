---
name: business-investor-validator
description: >
  Validate a business idea from an investor's perspective using hard criteria. Use this skill
  whenever a founder describes a startup idea, business concept, or venture and wants feedback,
  validation, or a reality check. Triggers on phrases like "validate my idea", "what do you think
  of my startup", "is this a good business idea", "review my concept", "would investors like this",
  "pitch feedback", or any time someone describes a business they want to build or are building.
  Also trigger when users ask for "investor perspective", "business viability", or "market analysis"
  of an idea. Always use this skill even if the idea is vague or early-stage — the skill handles
  all stages from raw concept to seed-stage.
---

# Business Investor Validator

A rigorous, investor-grade evaluation framework for business ideas. The goal is to help founders
stress-test their ideas with hard criteria — the same lens a sharp early-stage investor would use —
and walk away with clarity on what's strong, what's weak, and how to strengthen it.

---

## Core Evaluation Framework

Score each of the 4 dimensions on a scale of **1–10**, then compute a **weighted total score** out
of 100. The weights reflect what early-stage investors care most about.

| Dimension | Weight | What to assess |
|---|---|---|
| **Market Size & TAM** | 25% | Is the addressable market large enough to build a big company? |
| **Revenue Model & Unit Economics** | 25% | Is there a clear path to making money, with healthy margins? |
| **Scalability & Defensibility** | 25% | Can it grow without linear cost? Does it have a moat? |
| **Competitive Landscape** | 25% | Is the space crowded? Does the idea have a differentiated wedge? |

> **Weighted Score** = (Sum of scores × weights) × 10 → gives a 0–100 result.

---

## Scoring Rubrics

### 1. Market Size & TAM (25%)
- **9–10**: Billion-dollar+ global market with clear tailwinds
- **7–8**: $100M–$1B market, or niche but fast-growing
- **5–6**: $10M–$100M market, or unclear sizing
- **3–4**: Small/local market, limited ceiling
- **1–2**: Tiny, shrinking, or speculative market

**Key questions to probe:**
- What is the TAM, SAM, and SOM?
- Is the market growing or declining?
- Is the founder targeting a real, large segment or a micro-niche?
- Are there macro trends supporting this?

---

### 2. Revenue Model & Unit Economics (25%)
- **9–10**: Recurring revenue (SaaS/subscription), clear LTV > 3× CAC, high margins (>70%)
- **7–8**: Solid revenue model, reasonable margins, monetization tested
- **5–6**: Revenue model exists but thin margins or unclear unit economics
- **3–4**: Monetization vague or dependent on future pivots
- **1–2**: No clear revenue model or "we'll figure it out later"

**Key questions to probe:**
- How does the company make money?
- What's the pricing model?
- What are likely CAC and LTV estimates?
- Are margins sustainable at scale?

---

### 3. Scalability & Defensibility (25%)
- **9–10**: Network effects, data moats, or tech IP; scales without proportional cost increase
- **7–8**: Strong distribution or brand advantage; scalable ops
- **5–6**: Can scale but faces execution risk; some defensibility
- **3–4**: Easily replicated; growth requires proportional headcount/cost
- **1–2**: Highly manual, no moat, commodity play

**Key questions to probe:**
- What happens if this succeeds and a big player copies it?
- Does the product get better as more users join?
- Are there switching costs or lock-in?

---

### 4. Competitive Landscape (25%)
- **9–10**: Blue ocean or clear differentiated wedge; incumbents are slow to respond
- **7–8**: Competitive but founder has a unique angle or unfair advantage
- **5–6**: Crowded space; differentiation is present but not obvious
- **3–4**: Heavily competed; difficult to win without massive capital
- **1–2**: Dominated by entrenched players; no clear path to winning

**Key questions to probe:**
- Who are the top 3–5 competitors?
- What is the founder's unique insight or wedge?
- Why now? What's changed in the market?

---

## Idea Stage Adjustment

Calibrate expectations based on the maturity of the idea. Be tougher on later-stage ideas.

| Stage | What to expect |
|---|---|
| **Raw concept** | Ideas are fine to be unproven; score potential, not traction |
| **MVP / early traction** | Expect some early user signal; score execution quality |
| **Seed stage (some revenue)** | Expect data: CAC, retention, growth rate; score rigorously |

When the stage is unclear, ask or infer from context.

---

## Output Format

Always produce a structured output in this order:

### 1. 🎯 Idea Summary (2–3 sentences)
Restate the idea in investor language. Show you understood it.

### 2. 📊 Scorecard
A table with each dimension, score (X/10), and a 1-sentence rationale.

```
| Dimension                        | Score | Rationale |
|----------------------------------|-------|-----------|
| Market Size & TAM                | X/10  | ...       |
| Revenue Model & Unit Economics   | X/10  | ...       |
| Scalability & Defensibility      | X/10  | ...       |
| Competitive Landscape            | X/10  | ...       |
| **TOTAL (weighted)**             | XX/100 |          |
```

### 3. ✅ Verdict
Use one of these three tiers, then give 2–3 sentences of reasoning:
- 🟢 **Green Light** (75–100): Strong fundamentals. Worth pursuing seriously.
- 🟡 **Conditional** (50–74): Promising but has material gaps. Address before going all-in.
- 🔴 **Red Flag** (0–49): Significant issues. Reconsider core assumptions.

### 4. 💪 Strengths (2–3 bullet points)
What's genuinely working about this idea.

### 5. ⚠️ Key Risks & Gaps (2–4 bullet points)
Be direct and honest. This is where investors kill deals. Don't soften.

### 6. 🔧 How to Strengthen It
For each weak area (score < 7), give **1–2 concrete, actionable suggestions** the founder can
actually act on. Frame as "To improve [dimension], you should…"

### 7. 💡 Clarifying Questions
List 2–3 questions that, if answered, would most change the evaluation. These help the founder
sharpen their own thinking.

---

## Tone & Behavior Guidelines

- Be **honest and direct** — sugar-coating hurts founders more than it helps them.
- Be **constructive** — every critique should be paired with a path forward.
- Avoid generic advice like "do market research" — be specific to the idea.
- If critical information is missing (e.g., no revenue model mentioned), **flag it clearly** rather
  than assuming or downgrading silently.
- If the idea is genuinely strong, say so confidently — don't hedge everything.
- Think like a **first-principles investor**: does this solve a real problem at scale, with a clear
  business model, in a market worth winning?

---

## Example Invocation Phrases

This skill should activate when a user says things like:
- "Validate my startup idea: [idea]"
- "I'm building [X], what do you think from an investor perspective?"
- "Would VCs fund this? [description]"
- "Help me stress-test my business concept"
- "Is this a good idea to pursue?"
- "Here's my pitch: [pitch]"

See `references/examples.md` for worked examples of strong vs weak validations.
