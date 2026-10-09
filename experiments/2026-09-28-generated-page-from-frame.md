# Generating a playable page from a frame: white page, and whose fault it was

**Question.** Given a picture of a game plus its rules and an interface contract
(`getState`, `advanceTime`, `render_game_to_text`), can a vision-language model
produce a page that actually plays?

**Method.** `qwen2.5vl:7b` was given the frame and the contract and asked for a
single HTML file using a CDN script tag, inside a container with a browser. The
page was then loaded and inspected: console errors, event listeners, colour count,
and the state after two taps.

**Result.** A white page.

| check | result |
|---|---|
| `<script src>` for the library | **missing** - the word "Phaser" appeared in the title and in `new Phaser.Game`, and nowhere as an actual tag |
| `addEventListener` count | 0 |
| console | `pageerror: Phaser is not defined` |
| distinct colours on the page | 1 |
| state after tap 1 and tap 2 | `marks:[] cats:[]` |

**Before blaming the model, the harness.** The container was checked against the
same CDN URL the pipeline uses: `cdn.jsdelivr.net` and `unpkg` both answered 200
with 1.18 MB. So the network was fine, the browser was fine, and the missing script
tag was the model's omission.

**Verdict.** `watch` on the toolkit that would give a text model image abilities -
the harness is proven, so the next attempt is about the model, not the plumbing.
