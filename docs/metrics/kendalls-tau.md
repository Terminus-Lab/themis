### LLM-as-a-Judge Validation: A Detailed Example

#### The Core Problem

When you use an LLM to automatically evaluate agent responses, you need to validate: **Does the LLM judge agree with human judgment?**

This is critical because if the LLM judge rankings don't correlate with human rankings, your automated evaluation system is unreliable.

#### Validation Process Using Kendall's Tau

**Step 1: Create a Human-Annotated Subset**

From your full evaluation dataset of 10,000 query-response pairs, randomly sample 100-200 examples and have human experts score them.

**Why a subset?**
- Human annotation is expensive
- Annotating 10,000 examples takes weeks
- 100-200 examples provide statistically reliable correlation estimates
- Once validated, use the LLM judge on remaining 9,800 examples

**Step 2: Score the Same Subset with LLM Judge**

Run your automated LLM judge on the exact same 100-200 examples.

**Step 3: Calculate Kendall's Tau**

Compare the rankings to measure agreement.

#### Mathematical Example: Validating LLM Judge

Suppose you have 5 agent responses from your subset:

| Response ID | Human Rank | LLM Judge Rank |
|-------------|-----------|----------------|
| Response 1  | 1 (best)  | 1              |
| Response 2  | 2         | 3              |
| Response 3  | 3         | 2              |
| Response 4  | 4         | 4              |
| Response 5  | 5 (worst) | 5              |

**Total pairs = 5(4)/2 = 10**

**Check concordance:**

Concordant pairs (9):
- (R1,R2): Human 1<2, LLM 1<3 ✓
- (R1,R3): Human 1<3, LLM 1<2 ✓
- (R1,R4): Human 1<4, LLM 1<4 ✓
- (R1,R5): Human 1<5, LLM 1<5 ✓
- (R2,R4): Human 2<4, LLM 3<4 ✓
- (R2,R5): Human 2<5, LLM 3<5 ✓
- (R3,R4): Human 3<4, LLM 2<4 ✓
- (R3,R5): Human 3<5, LLM 2<5 ✓
- (R4,R5): Human 4<5, LLM 4<5 ✓

Discordant pairs (1):
- (R2,R3): Human 2<3, LLM 3>2 ✗

**Calculate:**
```
τ = (9 - 1) / 10 = 0.8
```

**Interpretation:**
- τ = 0.8 shows **strong agreement** (80%) between human and LLM judge
- The LLM judge is **reliable** for automated evaluation
- ✓ Safe to use at scale on the remaining dataset

**Decision thresholds:**
- τ > 0.75: LLM judge is validated, use it at scale
- τ = 0.50-0.75: Moderate agreement, improve judge prompts or rubrics
- τ < 0.50: Poor agreement, LLM judge is unreliable

#### Themis Implementation: Verdict-Based Ranking

Themis uses a three-tier verdict system instead of continuous scores:

**Verdict-to-Rank Conversion** (from `internal/batch/validator.go`):
- `pass` = 2 (highest quality)
- `review` = 1 (medium quality)
- `fail` = 0 (lowest quality)

**Example: Validating Themis Judge**

| Event ID | Human Annotation | LLM Verdict | Human Rank | LLM Rank |
|----------|-----------------|-------------|------------|----------|
| E1       | pass            | pass        | 2          | 2        |
| E2       | pass            | review      | 2          | 1        |
| E3       | review          | review      | 1          | 1        |
| E4       | fail            | fail        | 0          | 0        |
| E5       | fail            | review      | 0          | 1        |

**Step 1: Count all pairs** = 5 × 4 / 2 = 10 pairs

**Step 2: Classify concordant/discordant pairs**

| Pair | Human Ranks | LLM Ranks | humanDiff | llmDiff | Product | Type |
|------|-------------|-----------|-----------|---------|---------|------|
| (E1,E2) | (2,2) | (2,1) | 0 | +1 | 0 | Tie ⚪ |
| (E1,E3) | (2,1) | (2,1) | +1 | +1 | +1 | Concordant ✅ |
| (E1,E4) | (2,0) | (2,0) | +2 | +2 | +4 | Concordant ✅ |
| (E1,E5) | (2,0) | (2,1) | +2 | +1 | +2 | Concordant ✅ |
| (E2,E3) | (2,1) | (1,1) | +1 | 0 | 0 | Tie ⚪ |
| (E2,E4) | (2,0) | (1,0) | +2 | +1 | +2 | Concordant ✅ |
| (E2,E5) | (2,0) | (1,1) | +2 | 0 | 0 | Tie ⚪ |
| (E3,E4) | (1,0) | (1,0) | +1 | +1 | +1 | Concordant ✅ |
| (E3,E5) | (1,0) | (1,1) | +1 | 0 | 0 | Tie ⚪ |
| (E4,E5) | (0,0) | (0,1) | 0 | -1 | 0 | Tie ⚪ |

**Results:**
- Concordant pairs: **5** ✅
- Discordant pairs: **0** ❌
- Ties: **5** ⚪ (ignored in calculation)

**Step 3: Calculate τ**
```
τ = (concordant - discordant) / total_pairs
τ = (5 - 0) / 10 = 0.5
```

**Interpretation:**
- τ = 0.5 → "Moderate to strong agreement"
- Themis default threshold: τ ≥ 0.3
- ✓ This judge **passes validation**

**Why not τ = 1.0?**

Even with zero discordant pairs, τ = 0.5 because:
- E2: Human "pass" vs LLM "review" → Close but not perfect
- E5: Human "fail" vs LLM "review" → Close but not perfect
- 5 ties reduce the maximum possible concordance

For perfect τ = 1.0, every verdict must match exactly.

**Themis-Specific Validation Workflow:**

```bash
# Run validation with Themis CLI
themis validate \
  --input human_annotated.jsonl \
  --threshold 0.3

# Output includes:
# - kendall_tau: 0.5
# - interpretation: "Moderate to strong agreement"
# - passed: true
# - agreement_rate: 0.6 (3/5 exact matches)
# - confusion_matrix: pass_review:1, fail_review:1, ...
```

**Common τ ranges for Themis:**
- τ ≥ 0.6: Excellent judge, production-ready
- τ = 0.3-0.6: Acceptable, monitor closely
- τ < 0.3: Poor correlation, revise judge prompts

#### Evaluation Dataset Scenarios

**Scenario 1: Standard One-Query-One-Response (Most Common)**

Your evaluation dataset contains different queries, each with one response:

```
Row 1: Query 1 → Agent Response 1 → Human: 4.5/5 → LLM Judge: 4.2/5
Row 2: Query 2 → Agent Response 2 → Human: 3.8/5 → LLM Judge: 3.9/5
Row 3: Query 3 → Agent Response 3 → Human: 5.0/5 → LLM Judge: 4.8/5
...
Row 150: Query 150 → Agent Response 150 → Human: 4.1/5 → LLM Judge: 4.0/5
```

**What you're validating:** Does the LLM judge's scoring correlate with human scoring across diverse queries?

**Example calculation:**

| Query | Human Score (1-5) | LLM Judge Score (1-5) |
|-------|-------------------|----------------------|
| Q1    | 5                 | 5                    |
| Q2    | 4                 | 3                    |
| Q3    | 3                 | 4                    |
| Q4    | 2                 | 2                    |
| Q5    | 1                 | 1                    |

Convert to ranks and calculate τ to measure scoring consistency.

**Scenario 2: Multiple Models Answering Same Query**

Used for **model comparison** - testing different LLMs on identical queries:

```
Query: "Explain photosynthesis"
├─ GPT-4 answer     → Human rank: 1 (best) → LLM Judge rank: 1
├─ Claude answer    → Human rank: 2        → LLM Judge rank: 3
├─ Llama answer     → Human rank: 3        → LLM Judge rank: 2
└─ Gemini answer    → Human rank: 4        → LLM Judge rank: 4
```

**What you're validating:** Does the LLM judge rank model quality the same way humans do?

**Scenario 3: Prompt Strategy Testing**

Comparing different **prompting approaches** on the same query:

```
Query: "Summarize this 10-page document"
├─ Zero-shot prompting     → Human rank: 3 → LLM Judge rank: 3
├─ Few-shot prompting      → Human rank: 1 → LLM Judge rank: 1
├─ Chain-of-thought        → Human rank: 2 → LLM Judge rank: 2
└─ With examples           → Human rank: 4 → LLM Judge rank: 4
```

**What you're validating:** Can the LLM judge identify which prompting strategy produces better outputs?

#### Why Kendall's Tau is Ideal for This

1. **Ordinal data**: Ratings and rankings are naturally ordinal (1st, 2nd, 3rd)
2. **Scale invariant**: Human uses 1-5, LLM uses 1-10? Kendall's Tau handles it
3. **Handles ties**: Multiple responses can receive the same score
4. **Robust**: Not affected if LLM judge consistently rates 0.3 points higher
5. **Interpretable**: "75% agreement" makes sense to stakeholders
6. **Conservative**: More reliable than Spearman for validation samples of 100-200

#### Complete Validation Workflow

```
Full Dataset (10,000 examples)
        ↓
Sample 150 examples randomly
        ↓
     ┌─────┴─────┐
     ↓           ↓
Human Score   LLM Judge Score
(expensive)   (cheap, fast)
     └─────┬─────┘
           ↓
Calculate Kendall's Tau
           ↓
    ┌──────┴──────┐
    ↓             ↓
τ > 0.75      τ < 0.75
    ↓             ↓
✓ Validated   ✗ Needs work
    ↓             ↓
Use LLM       Improve:
on 9,850      - Rubrics
remaining     - Prompts
examples      - Model
              - Examples
```
