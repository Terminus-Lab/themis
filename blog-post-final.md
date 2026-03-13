# AI Agent Evaluation: Build First, Decide Later

AI agent evaluation looks deceptively simple. Write a prompt, get a score, done.

I spent one month building an evaluation system to understand why platforms are complex and which features actually matter. This is what I learned.

## Why Single Metrics Fail

The naive approach: write one prompt that scores responses from 0 to 1.

The same response scored 0.7, then 0.5, then 0.8 on consecutive runs. When scores dropped, I couldn't tell why. Hallucinating? Ignoring instructions? Incomplete?

A single metric provides no diagnostic signal. A response can be highly relevant but factually incorrect. Perfectly coherent but incomplete. One prompt cannot capture these distinct failure modes.

**Evaluation must be compositional.** Break quality into independent dimensions:

- **Relevance**: Does it address the question?
- **Faithfulness**: Is it grounded in provided context?
- **Coherence**: Is it logically consistent?
- **Completeness**: Does it cover all aspects?
- **Instruction**: Does it follow specific instructions?
- **Correctness**: Does it match expected output?

When a response fails, you immediately see which dimension broke. Relevance 0.9 but faithfulness 0.2? Hallucination. Coherence 0.95 but completeness 0.4? Partial answer.

This diagnostic signal is what makes multi-judge evaluation valuable.

## The Scariest Failure Mode: Evaluation Drift

After tuning judge prompts, scores improved across the board. Every response scored higher. Success?

Not quite. Spot-checking revealed high-scoring responses had obvious problems. The judges had drifted from human judgment.

**This is the scariest failure mode: the evaluator confidently scores garbage as excellent because you optimized the wrong thing.**

The fix is correlation validation using Kendall's τ. Take a sample (I used 25%), get human annotations, compute correlation between judge scores and human scores. If τ < 0.3, judges don't agree with humans.

Why Kendall's τ? It measures rank correlation—do judges rank responses in the same order as humans? A judge that consistently ranks good responses higher is useful even if its absolute 0-1 scale is miscalibrated. Pearson correlation assumes linear relationships—often not true for evaluation scores.

But validation metrics alone aren't enough. You need systematic improvement.

This is where configurable prompts become critical. When τ < 0.3:
1. Write improved prompt
2. Swap into configuration
3. Run on same dataset
4. Compare τ scores
5. Keep better prompt

This is A/B testing for evaluation infrastructure. Without easy prompt swapping, you can't iterate systematically.

Example: My faithfulness judge scored τ = 0.24 initially. I tested five prompts on 100 annotated samples. Prompt v3 achieved τ = 0.34.

**Calibration requires validation metrics AND configuration flexibility.** Evaluation solutions encode months of annotation work and systematic experimentation to achieve reliable correlation.

## Conversation Context Matters More Than Individual Turns

Early versions evaluated single events. One request, one response, one score.

This breaks when most agent interactions are multi-turn conversations:
- Turn 1: Agent asks clarifying question (scores well)
- Turn 2: User provides clarification
- Turn 3: Agent answers correctly (scores well)

Each turn scores well individually, but the conversation is inefficient. The agent should have answered in turn 1.

Conversation tracking revealed patterns invisible at turn level:
- Some agents are accurate but inefficient (five turns for simple questions)
- Others are fast but repetitive (same clarifying questions every conversation)
- Some drift off-topic in long conversations

Implementing this meant designing database schemas for turn grouping, aggregation logic, and visualizations. This took longer than building the judges themselves.

## Cost Control Through Intelligent Routing

Multi-judge evaluation scales costs quickly. Six judges running in parallel means six API calls per evaluation. Evaluating 10k responses can easily cost hundreds of dollars depending on model choice.

Running all judges on obviously broken responses—empty outputs, malformed JSON, input regurgitation—is expensive and wasteful. Pre-filtering is essential.

The solution: two-stage pipeline.

**Stage 1** runs fast rule-based checks without LLM calls. Length validation, overlap detection, format checking. If multiple checks fail, exit early. Don't call expensive judges.

This optimization significantly reduced costs. Only responses passing basic sanity checks proceed to Stage 2.

**Stage 2** runs LLM judges in parallel. You can configure specific models per judge (faithfulness uses Claude Sonnet, relevance uses GPT-4o-mini), or route through an intelligent selector that picks models based on cost and latency.

Cost control requires intelligent routing—but the decision logic, thresholds, and monitoring add infrastructure complexity.

## Integration: Agent Self-Evaluation

None of this matters if agents can't use it during development.

The system needs multiple interfaces: CLI for batch evaluation, API with dashboard for debugging, and—critically—Model Context Protocol (MCP) for agent self-evaluation.

MCP lets agents call external tools during execution. Add an evaluation server to Claude Code's MCP config, and the agent can:

1. Generate code
2. Call `evaluate_response` via MCP
3. See scores per judge (relevance: 0.9, correctness: 0.6)
4. Regenerate with fixes
5. Evaluate again
6. Repeat until passing

This closes the feedback loop: agents generate → self-evaluate → self-correct → improve. The bottleneck shifts from human review to evaluation infrastructure.

**However, there's a caveat:** agents can learn to game evaluation metrics. This is reward hacking—agents optimize for high scores rather than actual quality. You need to continuously validate that judges still correlate with human judgment, especially when agents are training on evaluation feedback.

Building MCP integration requires understanding the protocol, implementing tool handlers, managing Docker deployments, and handling stdio communication—non-trivial infrastructure work.

## What Building Taught Me

**Judge Development**: Writing specialized prompts and validating them against human judgment. This is the knowledge evaluation solutions encode.

**Calibration Systems**: Validation metrics plus configurable prompts enable A/B testing. Testing five faithfulness prompts to find one passing correlation threshold required annotation infrastructure and statistical pipelines.

**Conversation Analytics**: Database schemas, aggregation logic, visualization dashboards. Often overlooked but critical for understanding agent behavior.

**Cost Optimization**: Rule-based filtering before expensive LLM calls. Requires decision logic, threshold tuning, and monitoring.

**Integration Points**: CLI, API, dashboards, MCP servers. Each interface requires engineering effort.

None of these are conceptually hard. Together they represent significant engineering work.

## Why Go

I built the system in Go for two pragmatic reasons: single-binary CLI tools and trivial parallel execution via goroutines.

## Conclusion

Evaluation tools are complex because evaluation itself is complex.

Single prompts fail because quality is multi-dimensional. Judges drift from human judgment without continuous validation. Single-turn evaluation misses conversation context. And none of it matters without integration into agent workflows.

Building an evaluation system teaches you where complexity hides. The scariest lesson: evaluators can confidently score garbage as excellent when they drift from human judgment. This failure mode is silent and corrupts every downstream decision.

**If you're considering evaluation platforms, start here:**

Build two judges (relevance and faithfulness). Collect 50 human annotations. Compute Kendall's τ.

You'll understand what features actually matter—or decide you need custom infrastructure. Either way, you'll know what you're choosing and why.

Build first. Decide later.

---

*The system described here is open sourced as Themis at github.com/Terminus-Lab/themis*
