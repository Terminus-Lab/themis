# AI Agent Evaluation: Build First, Decide Later

AI agent evaluation looks deceptively simple. Write a prompt that asks an LLM to score responses. Get a number between 0 and 1. Done.

This is how I started. But my goal wasn't to ship an evaluation framework—it was to understand the problem space deeply enough to evaluate evaluation tools intelligently.

I've always learned by building. Not reading documentation, but writing code until I hit walls I didn't know existed. One month taught me which features actually matter and why evaluation platforms are complex.

This is what I learned.

## The Naive Start

The initial approach: write a single prompt that scores responses, return a number, done.

The same response scored 0.7, then 0.5, then 0.8 on consecutive runs. When scores dropped, I couldn't tell why. Hallucinating? Ignoring instructions? Missing context?

**First lesson: single-prompt evaluation fails because it tries to capture multiple quality dimensions simultaneously.**

A response can be highly relevant but factually incorrect. Perfectly coherent but incomplete. One prompt cannot evaluate all these dimensions.

## Breaking Down Quality

The breakthrough was realizing evaluation needs to be compositional. Break quality into independent dimensions:

- **Relevance**: Does it address the question?
- **Faithfulness**: Is it grounded in provided context?
- **Coherence**: Is it logically consistent?
- **Completeness**: Does it cover all aspects?
- **Instruction**: Does it follow specific instructions?
- **Correctness**: Does it match expected output?

When a response fails, you immediately see which dimension broke. Relevance 0.9 but faithfulness 0.2? Hallucination. Coherence 0.95 but completeness 0.4? Partial answer.

Writing six specialized prompts and iterating until they match human judgment took three weeks. This is the knowledge mature evaluation solutions encode.

## The Cost Explosion

Multi-judge evaluation introduces a new problem: cost.

Six LLM judges in parallel means six API calls per evaluation. At $0.01 per call, evaluating 10,000 responses costs $600.

I burned $400 evaluating obviously broken test data—empty responses, malformed JSON, input regurgitation—because I had no pre-filtering.

The solution: two-stage pipeline.

**Stage 1** runs fast heuristic prechecks without LLM calls. Length validation, overlap detection, format checking. If average score < 0.2, exit early.

This cuts costs by 80%. Only quality responses proceed to Stage 2.

**Stage 2** runs LLM judges in parallel. Configure specific models per judge (faithfulness uses Claude Sonnet, relevance uses GPT-4o-mini), or route through an intelligent model selector that picks optimal models based on cost and latency.

**Second lesson: cost control requires intelligent routing.** Building this means implementing decision logic, tuning thresholds, and monitoring costs.

## The Weight Problem

Fixed judge weights seem reasonable initially. Weight all six equally. Simple, explainable.

This breaks fast. RAG systems care about faithfulness. Code generation cares about correctness. Customer support needs instruction-following.

The solution: configurable weights, aggregation methods (weighted average vs harmonic mean vs median), verdict thresholds—all tunable per domain.

But flexibility creates problems. Setting all weights to zero breaks aggregation. I spent time debugging scores before realizing my config gave correctness 90% weight—essentially single-judge evaluation again.

**Third lesson: flexibility requires validation.** Every configuration change needs validation.

## The Validation Problem

After tuning judge prompts, scores improved across the board. Success?

Not quite. Spot-checking revealed high-scoring responses had obvious problems. The judges had drifted from human judgment.

This is the scariest failure mode: the evaluator confidently scores garbage as excellent.

The fix: Kendall's τ correlation validation. Take a sample (I used 25%), get human annotations, compute correlation. If τ < 0.3, judges don't agree with humans.

Why Kendall's τ? It measures rank correlation—do judges rank responses in the same order as humans? A judge that consistently ranks good responses higher is useful even if its 0-1 scale is miscalibrated. Pearson correlation assumes linear relationships—often not true.

But validation metrics alone aren't enough. You need systematic improvement.

This is where configurable prompts become critical. When τ < 0.3:
1. Write improved prompt
2. Swap into configuration
3. Run on same dataset
4. Compare τ scores
5. Keep better prompt

This is A/B testing for evaluation infrastructure. Without easy prompt swapping, you can't iterate systematically.

Example: My faithfulness judge scored τ = 0.24 initially. I tested five prompts over two weeks on the same 100 annotated samples. Prompt v3 achieved τ = 0.34 and became production.

**Fourth lesson: calibration requires validation metrics AND configuration flexibility.** Budget time and money for human labeling and systematic experimentation.

## Conversations, Not Events

Early versions evaluated single events. One request, one response, one score.

This breaks when most agent interactions are multi-turn conversations:
- Turn 1: Agent asks clarifying question (valid)
- Turn 2: User provides clarification
- Turn 3: Agent answers correctly (valid)

Each turn scores well individually, but the conversation is inefficient. The agent should have answered in turn 1.

Conversation tracking revealed patterns invisible at turn level:
- Some agents are accurate but inefficient (five turns for simple questions)
- Others are fast but repetitive (asking same questions every conversation)
- Some drift off-topic in long conversations

**Fifth lesson: conversation context matters more than individual turn quality.** This meant designing database schemas for turn grouping, aggregation logic, and visualizations—took longer than building the judges.

## Integration: The Hidden Complexity

None of this matters if you can't use it where you work.

**CLI** for batch evaluation with concurrent workers. Essential for offline experimentation.

**API with dashboard** for real-time inspection. Filter by agent or verdict. Drill into individual turns.

**Model Context Protocol (MCP)** for agent self-evaluation. This is critical.

MCP lets agents call external tools during execution. Add an evaluation server to Claude Code's MCP config, and the agent can:

1. Generate code
2. Call `evaluate_response` via MCP
3. See scores per judge (relevance: 0.9, correctness: 0.6)
4. Regenerate with fixes
5. Evaluate again
6. Repeat until passing

All without human intervention.

This closes the feedback loop. Without evaluation integration: agents generate → humans review → agents learn nothing. With MCP: agents generate → self-evaluate → self-correct → improve. The bottleneck shifts from human review to evaluation infrastructure.

**Sixth lesson: evaluation tools must integrate with agent workflows.** Building MCP integration requires understanding the protocol, implementing tool handlers, managing Docker deployments, and handling stdio communication—non-trivial infrastructure work.

## What These Challenges Teach You

Building an evaluation system reveals hidden complexity:

**Judge Development**: Three weeks writing prompts. Another week collecting annotations and validating correlation.

**Cost Optimization**: Decision logic, threshold tuning, monitoring. One misconfigured threshold cost $400.

**Configuration Complexity**: Flexible weighting creates combinatorial explosion. Easy to accidentally configure single-judge evaluation with complex UI.

**Calibration Systems**: Validation metrics plus configurable prompts enable A/B testing. Tested five faithfulness prompts over two weeks to find one passing correlation threshold.

**Conversation Analytics**: Database schemas, aggregation logic, visualization dashboards. Took longer than building judges.

**Integration Points**: CLI, API, dashboards, MCP servers. Each requires significant engineering effort.

None of these are conceptually hard. Together they represent two to three months of focused work. This is the accumulated knowledge evaluation solutions encode.

## Why Go for Implementation

When I built this (open sourced as Themis at github.com/Terminus-Lab/themis), I chose Go for two reasons:

**CLI Excellence**: Go produces single-binary executables with fantastic CLI libraries like Cobra. No runtime dependencies. No virtual environments. Copy the binary, it works.

**Concurrency Model**: Go's goroutines and channels make parallel judge execution trivial. Spin up six goroutines for six judges, collect results via channels, aggregate when complete. Clean code, excellent performance.

These directly address the hardest integration problems: making evaluation accessible to developers (CLI) and fast enough for continuous use (concurrency).

## Conclusion

AI agent evaluation looks simple until you build it.

Single prompts fail because quality is multi-dimensional. Multi-judge evaluation is expensive without intelligent routing. Flexible configuration requires validation infrastructure. Single-turn evaluation misses conversation context. And none of it matters without integration into agent workflows.

Building an evaluation system teaches you what problems existing solutions solve and where complexity hides. More importantly, it teaches you what to look for when evaluating platforms. Not features—everyone has multi-judge evaluation. But rather: how well are judges validated? What cost optimization strategies? Do they support conversation-level analytics? Can agents call evaluation tools via MCP?

**If you're considering evaluation platforms, start here:**

Build two judges (relevance + faithfulness). Get 50 human annotations on sample data. Compute Kendall's τ. This takes two weeks and costs under $50 in API calls.

You'll understand the problem space well enough to evaluate platforms intelligently—or decide you need custom infrastructure.

Build first. Decide later. Understanding both sides helps you build better AI systems.

---

*The system described here is open sourced as Themis at github.com/Terminus-Lab/themis*
