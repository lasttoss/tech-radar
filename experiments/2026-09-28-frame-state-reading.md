# Reading a game frame: which vision model can be trusted with state

**Question.** Can a vision-language model read structured state out of a game frame
reliably enough to drive an automation pipeline, on a local machine, with no cost
per call?

**Method.** One 980x768 frame of a running puzzle game, ground truth read by hand:
a 4x4 board, one cat, zero marks, HUD showing 0/4 and three hearts, and four
coloured regions. The same prompt was given to two local models, asking for the
same fields. Frame and prompt were identical in both runs.

| run | model | result | time | tokens |
|---|---|---|---|---|
| T1a | `qwen3-vl:8b` | **empty content** - all 3165 characters went into the thinking block, and the tag always thinks, so thinking could not be turned off | 900 eval tokens | - |
| T1b | `qwen2.5vl:7b` | **6 of 7 fields correct**; wrong on region colours (said 2, there are 4) | 9.1s | 62 |

**Verdict.** T1a `reject`: a model that answers with nothing on a prompt this
concrete cannot be part of a state-reading step, and this was not a prompt problem -
it is the tag's behaviour. T1b `watch`: accurate enough to be a baseline, not
accurate enough to replace the existing pipeline. Kept on the bench, at zero cost.

**Why this is worth writing down.** The empty answer looked like a broken harness.
It was not: the failure was inside the model's output structure. That distinction is
the expensive one to make by hand at 2am, and it is now a file.
