# The Calibration Loop

*This is the final part of a three-part series. [Part 1](https://vladpovarna.substack.com/p/ai-agent-evaluation-build-first-decide) covered why single metrics fail and how evaluation drift silently corrupts downstream decisions. [Part 2](https://vladpovarna.substack.com/p/build-first-measure-comprehensively) covered validation metrics and ended with an honest admission: every monitoring path leads back to human annotation, and I didn't have a full solution.*

*This is what I tried to close it.*

---

## The Debt from Part 2

Part 2 ended here:

> *"I don't have a fully automated solution to offer here, and I'm skeptical of anyone who claims they do."*

That was accurate at the time. Human annotation was the bottleneck — necessary to distinguish judge drift from agent regression, necessary to tune prompts, necessary to validate improvements. The tooling existed to measure the gap between judge scores and human judgment. What didn't exist was a systematic way to close it.

## DSPy and the Idea Behind It

Reading [Dropbox's post on optimizing their Dash relevance judge](https://dropbox.tech/machine-learning/optimizing-dropbox-dash-relevance-judge-with-dspy) introduced me to [DSPy](https://github.com/stanfordnlp/dspy), a framework from Stanford that changed how I thought about the problem.

The philosophy behind DSPy is a direct challenge to how most people work with LLMs. When a prompt doesn't perform well, the default response is to rewrite it — manually, iteratively, by intuition. DSPy argues this is the wrong abstraction entirely. Prompts are parameters, not programs. You wouldn't hand-tune the weights of a neural network; you'd define a loss function and let the optimizer find them. DSPy applies the same idea to prompts: define what you want (a metric), provide labeled examples, and let the system search for the prompt that maximizes your metric.

The practical implication is significant. Instead of asking *"how should I phrase this prompt?"*, you ask *"what does success look like, and do I have examples of it?"* The optimizer handles the rest.

I didn't adopt DSPy directly — the architecture already had everything needed without adding a new framework. But the philosophy shaped what I built: stop editing prompts by intuition, start using disagreements between judge scores and human judgment as the optimization signal.

## The Rubric Change

Before closing the loop, I changed the scoring scale from 0–1 to 1–5.

This matters because the entire approach depends on annotation quality. If humans score inconsistently, the disagreements are noise, and any suggested prompt revision is built on bad signal. A continuous 0–1 scale sounds precise but produces inconsistency in practice — annotators interpret the same response differently, and the boundary between good and borderline is invisible.

A 1–5 rubric forces discrimination. Each point has explicit criteria. Humans annotate more consistently against discrete levels, and the disagreements that surface become genuinely diagnostic. Kendall's τ and Cohen's Kappa handle ordinal data naturally, so the metrics required no changes.

Better annotations feed better disagreements. Better disagreements produce better prompt suggestions.

## What I Created and Tried

The feedback loop works in five steps:

**1. Annotate with reasoning.** I created a TUI terminal app that presents each sampled conversation and collects three things: a verdict (`pass` / `review` / `fail`), a score (1–5), and a free-text reasoning field. The reasoning is what makes the next step meaningful.

**2. Identify the worst judge.** After evaluation, the correlation report shows per-judge Cohen's Kappa. The judge with the lowest Kappa is the one most misaligned with human judgment.

**3. Collect disagreements.** The evaluation CLI filters to conversations where `themis_verdict != human_label` and returns them directly in the output — printed to the log and included in the summary file alongside the correlation report. Each record contains the judge's score, the human's score, and the human's reasoning for why they judged differently.

**4. Ask an LLM to suggest a revision.** The disagreements are assembled into a meta-prompt: *"Here is the current judge prompt. In these N examples the judge scored X but the human said Y with this reasoning. Suggest a revised prompt."* This is the DSPy philosophy applied without the framework — use failure cases and human signal to search for a better prompt, rather than editing by intuition.

**5. Human approval before anything changes.** The suggested revision is shown as a YAML diff. Nothing changes in `judges.yaml` until a human approves it. Then re-evaluate on the same sample and confirm Kappa improved.

The free-text reasoning from step 1 is what makes step 4 non-trivial. Without it, the optimizer sees score disagreements but not the reasoning gap — it knows the judge was wrong, not why. With reasoning, the meta-prompt contains specific failure cases and explicit human expectations. The resulting suggestions are targeted, not generic.

## What I Didn't Try

At one point I considered feeding the holistic conversation score back into the turn judges as additional context — the idea being that turn judges might score more accurately knowing how the overall conversation was rated.

After thinking it through: no. Turn judges are intentionally scoped to a single turn. Feeding holistic context into them introduces a circular dependency — if Phase A needs Phase B output and Phase B needs Phase A output, you need to run the holistic judge twice. The conversation-level context is already the job of the holistic judge. Adding conversation history to the turn judge prompt achieves the real goal without the dependency inversion.

## Testing Locally with Ollama

The loop makes a lot of LLM calls. Each candidate prompt gets evaluated on the full annotated sample — 100 annotations, three prompt variants, that's 300 judge calls before committing to anything.

I used [Ollama](https://ollama.com/) with a 7B model locally to iterate without burning API quota: verifying that YAML diffs render correctly, that the meta-prompt produces coherent output, that disagreements surface as expected. Local models respond in seconds and cost nothing for that kind of exploratory work.

The broader issue is real though. Evaluation systems are expensive to iterate on because the number of LLM calls compounds quickly — per turn, per judge, per candidate prompt. Even small models hit rate limits fast once the annotation set grows. It's a constraint worth keeping in mind when designing how frequently to run recalibration.

## Results

On a sample of 120 annotated conversations, the relevance judge had the lowest initial Kappa at 0.31. After one optimization cycle — three candidate prompts evaluated, one selected and approved — Kappa improved to 0.48. The confusion matrix showed the primary failure mode was borderline `review` cases being classified as `pass`. The human reasoning made that pattern visible in the meta-prompt; the revised rubric added explicit criteria for when a response is review rather than pass.

One cycle. One meaningful improvement. The loop held.

## Closing the Trilogy

Part 1 identified the failure modes: single metrics, evaluation drift, missing conversation context.

Part 2 built the measurement infrastructure: Kendall's τ, Cohen's Kappa, confusion matrix. It ended with the annotation bottleneck unsolved.

This part describes what I tried to close it. The loop is: annotate with reasoning, identify the worst judge, collect disagreements, suggest a revised prompt, validate the improvement. Human judgment stays in at two points — annotation and approval — but the work between those points is no longer manual.

The bottleneck didn't disappear. It became something you can act on.

---

*Themis is open source at [github.com/Terminus-Lab/themis](https://github.com/Terminus-Lab/themis). The annotation TUI is at [github.com/Terminus-Lab/stamper](https://github.com/Terminus-Lab/stamper).*
