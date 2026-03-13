# AI Agent Evaluation: Build First, Decide Later

AI agent evaluation looks deceptively simple. Write a prompt that asks an LLM to score responses. Get a number between 0 and 1. Done.

This is how I started. But my goal wasn't to ship an evaluation framework—it was to understand the problem space deeply enough to evaluate evaluation tools intelligently.

I've always learned by building. Not reading documentation, but writing code until I hit walls I didn't know existed. Two months of building an evaluation system taught me why platforms are complex, what problems they're solving, and which features actually matter.

This is what I learned.

## The Naive Start

The initial approach was embarrassingly simple: write a single prompt that scores agent responses, return a number, call it done.

```
You are an evaluator. Score this response from 0 to 1 based on quality.
Question: {question}
Response: {response}
```
The same response scored 0.7, then 0.5, then 0.8 on consecutive runs. When scores dropped, I couldn't tell why. Was the agent hallucinating? Ignoring instructions? Missing parts of the question?

A single score between 0 and 1 provides no diagnostic signal. You know something is wrong but not what. You can't debug what you can't measure.

**First lesson: single-prompt evaluation fails because it tries to capture multiple quality dimensions simultaneously.**

A response can be highly relevant but factually incorrect. Perfectly coherent but incomplete. Following instructions precisely while ignoring critical context. One prompt cannot evaluate all these dimensions and provide useful feedback.

## Breaking Down Quality

The breakthrough was realizing evaluation needs to be compositional. Instead of one monolithic judge asking "is this good?", break quality into independent dimensions:
- **Relevance**: Does the response address the question?
- **Faithfulness**: Is it grounded in provided context?
- **Coherence**: Is it logically consistent and well-structured?
- **Completeness**: Does it cover all aspects of the request?
- **Instruction**: Does it follow specific instructions?
- **Correctness**: Does it match expected output?

Each dimension gets its own specialized prompt and evaluation strategy. When a response fails, you immediately see which dimension broke. Relevance at 0.9 but faithfulness at 0.2? The agent hallucinated. Coherence at 0.95 but completeness at 0.4? It answered part of the question and ignored the rest.

This diagnostic signal is actionable. You understand exactly what the agent did wrong.

Writing six specialized prompts and iterating until they match human judgment took me three weeks. This is the knowledge that mature evaluation solutions encode—pre-tuned judges that already produce reliable scores across quality dimensions.

## The Cost Explosion

Multi-judge evaluation introduces a new problem: cost.

Six LLM judges running in parallel means six API calls per evaluation. At $0.01 per call, evaluating 10,000 responses costs $600. For continuous evaluation of live traffic, this scales poorly.

I discovered this after burning $400 evaluating test data that was obviously broken. Empty responses, malformed JSON, input regurgitation—all hit expensive judges because I had no pre-filtering. The fix was obvious in retrospect: don't call expensive judges on garbage inputs.

The solution is a two-stage pipeline.

**Stage 1** runs fast heuristic prechecks without LLM calls. Length validation, input-output overlap detection, format checking. If the average precheck score falls below a threshold (say 0.2), exit early. Don't call the expensive LLM judges.

This optimization cuts costs by 80%. Obvious failures get caught immediately with zero API cost. Only quality responses proceed to Stage 2.

**Stage 2** runs the LLM judges in parallel. You have two routing options: configure specific models per judge (faithfulness uses Claude Sonnet, relevance uses GPT-4o-mini), or route all requests through an intelligent model selector that picks optimal models based on cost and latency requirements.

**Second lesson: cost control requires intelligent routing.** Don't hit expensive judges for responses that fail basic heuristics. Building this means implementing decision logic, tuning thresholds, and monitoring costs. Not conceptually hard, but it requires infrastructure.

## The Weight Problem

Fixed judge weights seem reasonable initially. Weight all six judges equally. Simple, explainable.

This breaks down fast.

RAG systems care deeply about faithfulness—did the agent cite correct sources? Code generation cares about correctness—does the output match expected behavior? Customer support agents need strong instruction-following—did they follow company policies?

One-size-fits-all weighting doesn't work. Different use cases need different priorities.

The solution is making everything configurable. Judge weights, aggregation methods (weighted average vs harmonic mean vs median), verdict thresholds—all tunable based on your domain.

But flexibility creates new problems: you can now configure the system into nonsense. Setting all weights to zero breaks aggregation. Using harmonic mean with unbalanced weights makes one judge dominate everything.

**Third lesson: flexibility requires validation.** A good evaluation solution provides sensible defaults tuned across thousands of evaluations. When you build your own, every configuration change needs validation. I spent two days debugging why scores looked wrong before realizing my weighted average configuration gave the correctness judge 90% weight—essentially turning it back into single-judge evaluation.

## The Validation Problem

After tuning judge prompts, evaluation scores improved across the board. Every response scored higher. Success?

Not quite.

When spot-checking results, they seemed worse. High-scoring responses had obvious problems. The judges had drifted from human judgment.

This is the scariest failure mode: the evaluator confidently scores garbage as excellent because you optimized the wrong thing.

The fix is correlation validation using Kendall's τ. Take a sample of responses (I used 25% of my dataset), get human annotations, compute correlation between judge scores and human scores. If τ < 0.3, your judges don't agree with human judgment and need retuning.

Why Kendall's τ specifically? It measures rank correlation—do judges rank responses in the same order as humans? This matters more than absolute score agreement. A judge that consistently ranks good responses higher than bad ones is useful even if its 0-1 scale is miscalibrated. Pearson correlation measures linear relationships, which assumes your judge scores are linearly related to human scores—often not true.

But validation metrics alone aren't enough. You need a systematic way to improve judges when correlation is low.

This is where configurable prompts become critical. When a judge's τ score is below 0.3, you need to:
1. Write an improved prompt
2. Swap it into the judge configuration
3. Run evaluation on the same dataset
4. Compare τ scores
5. Keep the better prompt

This is A/B testing for evaluation infrastructure. Without easy prompt swapping, you can't iterate systematically. You're stuck rewriting code every time you want to test a new prompt.

Example: My faithfulness judge initially scored τ = 0.24 against human annotations. I tested five different prompts over two weeks, running each against the same 100 annotated samples. Prompt v3 achieved τ = 0.34 and became the production version. Without configuration flexibility and validation metrics together, this experimentation loop would be impossible.

Building a calibration system requires:
- Human annotation infrastructure (who labels? how much? what format?)
- Statistical validation pipeline (compute correlation, flag regressions)
- Iterative tuning process (adjust prompts until correlation improves)

Human annotation is expensive and slow. But shipping broken evaluators is worse. They silently corrupt every decision downstream—model selection, prompt tuning, deployment decisions—all based on bogus scores.

**Fourth lesson: calibration requires validation metrics AND configuration flexibility.** Evaluation solutions encode months of annotation work and systematic prompt experimentation to achieve reliable correlation. When you build your own, you need both the validation infrastructure and the ability to swap prompts easily for A/B testing. Budget time and money for human labeling and systematic experimentation.

## Conversations, Not Events

Early versions evaluated single events. One request, one response, one score. Done.

This breaks down when you realize most agent interactions are multi-turn conversations. The agent asks clarifying questions. The user provides more context. The agent refines its answer across three turns.

Evaluating each turn in isolation misses context-dependent failures:
- Turn 1: Agent asks clarifying question (valid)
- Turn 2: User provides clarification
- Turn 3: Agent answers correctly (valid)

Each turn scores well individually, but the conversation is inefficient. The agent should have answered correctly in turn 1 instead of asking obvious questions.

The system needs to track conversations with conversation IDs and group turns together. You can evaluate individual turns for granular debugging, but also need conversation-level metrics for holistic assessment.

This revealed patterns invisible at the turn level:
- Some agents are highly accurate but inefficient (five turns for simple questions)
- Others are fast but repetitive (asking the same clarifying questions every conversation)
- Some drift off-topic across long conversations

**Fifth lesson: conversation context matters more than individual turn quality.** Implementing conversation tracking meant designing database schemas for turn grouping, building aggregation logic for conversation-level metrics, and creating visualizations that let you drill down from conversation to turn. This took longer than building the judges themselves.

## Integration: The Hidden Complexity

None of this matters if you can't use it where you work.

The system needs multiple interfaces:

**CLI** for batch evaluation. Process datasets with concurrent workers. Validate configurations against human annotations. Essential for offline experimentation.

**API** with dashboard for real-time inspection. See evaluation results as they happen. Filter by agent or verdict. Drill into individual turns. Debug issues visually.

**Model Context Protocol (MCP)** for agent self-evaluation. This is the critical one.

MCP lets agents call external tools during execution. Add an evaluation server to Claude Code's MCP config, and suddenly the agent can evaluate its own generated code:

1. Claude Code generates a function
2. Calls `evaluate_response` tool via MCP
3. Gets back scores per judge (relevance: 0.9, correctness: 0.6, faithfulness: 0.8)
4. Sees correctness is low
5. Regenerates the function with fixes
6. Evaluates again
7. Repeats until all scores pass thresholds

All without human intervention.

This closes the feedback loop. Without evaluation integration, agents generate → humans review → agents learn nothing. With MCP, agents generate → self-evaluate → self-correct → improve. The bottleneck shifts from human review to evaluation infrastructure.

The alternative is humans manually reviewing every agent-generated artifact. That doesn't scale. Agent self-evaluation through structured feedback is what separates demo agents from production-ready ones.

**Sixth lesson: evaluation tools must integrate with agent workflows.** Most solutions provide API endpoints, but few offer MCP integration. For coding agents that generate code during development, this matters enormously. Building MCP integration yourself requires understanding the protocol, implementing tool handlers, managing Docker deployments, and handling stdio-based communication—non-trivial infrastructure work.

## What These Challenges Teach You

Building an evaluation system reveals hidden complexity:

**Judge Development**: Three weeks writing and tuning prompts. Another two weeks collecting human annotations and validating correlation.

**Cost Optimization**: Decision logic, threshold tuning, monitoring infrastructure. One misconfigured threshold cost me $400.

**Configuration Complexity**: Flexible weighting creates combinatorial explosion. Spent two days debugging why scores looked wrong—had accidentally configured single-judge evaluation with fancy UI.

**Calibration Systems**: Validation metrics (Kendall's τ) plus configurable prompts enable systematic A/B testing of judge improvements. Tested five faithfulness prompts over two weeks to find one that passed correlation threshold. Requires annotation pipelines, statistical analysis, and easy prompt swapping.

**Conversation Analytics**: Database schemas for turn grouping, aggregation logic, visualization dashboards. Took longer than building the judges.

**Integration Points**: CLI, API, dashboards, MCP servers. Each interface requires significant engineering effort.

None of these problems are conceptually hard. But together they represent two to three months of focused work. This is the accumulated knowledge that evaluation solutions encode—not just the technology, but what works, what doesn't, and how to configure it for different domains.

## Why Go for Implementation

When I built this (open sourced as Themis at github.com/Terminus-Lab/themis), I chose Go for two specific reasons:

**CLI Excellence**: Go produces single-binary executables with fantastic CLI libraries like Cobra. No runtime dependencies. No virtual environments. Copy the binary, it works. This matters enormously for tooling that developers use daily.

**Concurrency Model**: Go's goroutines and channels make parallel judge execution trivial. Spin up six goroutines for six judges, collect results via channels, aggregate when all complete. The code is clean and the performance is excellent. This makes Go ideal for high-throughput evaluation workloads.

These aren't theoretical benefits—they directly address the two hardest integration problems: making evaluation accessible to developers (CLI) and making it fast enough for continuous use (concurrency).

## Conclusion

AI agent evaluation looks simple until you build it.

Single prompts fail because quality is multi-dimensional. Multi-judge evaluation is expensive without intelligent cost controls. Flexible configuration requires validation infrastructure. Single-turn evaluation misses conversation context. And none of it matters without integration into agent workflows.

These challenges aren't obvious until you encounter them. Building an evaluation system—even a simple one—teaches you what problems existing solutions are solving and where the complexity hides.

More importantly, it teaches you what to look for when evaluating platforms. Not features—everyone has multi-judge evaluation. But rather: how well are judges validated? What cost optimization strategies do they use? Do they support conversation-level analytics? Can agents call evaluation tools directly via MCP?

**If you're considering evaluation platforms, start here:**

Build two judges (relevance + faithfulness). Get 50 human annotations on sample data. Compute Kendall's τ correlation. This takes two weeks and costs under $50 in API calls.

You'll understand the problem space well enough to evaluate platforms intelligently—or decide you need to build custom infrastructure. Either way, you'll know what you're choosing and why.

The evaluation infrastructure is catching up to the agents. Build first. Decide later. Understanding both sides helps you build better AI systems.

---

*The system described here is open sourced as Themis at github.com/Terminus-Lab/themis*
