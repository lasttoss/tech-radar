# Measurements

Two kinds of number appear in this repository: what the pipeline does, and what a model did.

## Environment

| | |
|---|---|
| Machine | 12-core x86-64 (AMD64), Ubuntu 24.04, Docker 29.1.3 |
| Toolchain | Go 1.27.2 / Node 24.19.0 / JDK 17 in `maven:3.9-eclipse-temurin-17` |
| Date | 2026-10-13 (UTC) |


## The pipeline

```
$ make demo            # the ordinary path, over the committed fixtures, with no model provider
$ make coverage
total:  (statements)  73.1%
```

**Result.** 73.1% of statements, with no provider configured, which is the repository's design: the
ranking is code, and a run that needs a network and a key would be a run nobody repeats.

## 6 of 7 frame fields correct in 9.1 s / 62 tokens

That is a **model** result, and it is not reproduced by anything on this page: it needs a local vision
model. It is quoted in the README from
[`experiments/2026-09-28-frame-state-reading.md`](../experiments/2026-09-28-frame-state-reading.md),
which records the run - model, result, wall clock, tokens - and says which field was wrong:

| run | model | result | time | tokens |
|---|---|---|---|---|
| T1a | `qwen2.5vl:7b` | rejected: answered with nothing on a prompt this explicit | - | - |
| T1b | `qwen2.5vl:7b` | **6 of 7 fields correct**; wrong on region colours | 9.1 s | 62 |

**To reproduce it:** a machine with Ollama and `qwen2.5vl:7b` pulled, then the prompt in that file.
The verdict recorded there - `reject` - is the reason the number is in the README at all: a model that
gets six of seven structured fields right is not a model to build state on, and saying so is cheaper
than finding out in production.
