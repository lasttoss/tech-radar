# Provenance

Original work, written from scratch, off employer time and equipment, on a personal
machine. No employer code, schema, asset, hostname, credential or player data is
included, and nothing here is copied from a work project.

The pipeline was rebuilt in Go for this repository. The daily instance that runs on
my own machine is a separate private tool; this one shares its idea, not its code.
Its content rules are the reason for a few odd-looking decisions in here: no
credentials, no internal hosts, no product names, and every claim in the README
must be reproducible from the repository itself.

Everything in `testdata/fixtures/` is a recording of public API responses, with the
date each was recorded written down in `testdata/fixtures/README.md`. The verdicts
in `decisions/` describe public models and public repositories, measured in public
sessions. Where a verdict would have to name a private repository and commit, it is
omitted rather than invented, and `decisions/applied.md` says so.
