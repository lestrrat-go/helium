# `.claude/docs`

Purpose:

- Act as stable index into `helium` codebase.
- Cache rough package structure, key files, feature status, behavior notes, and review hotspots so agents do not waste tokens rediscovering them.
- Hold repository-specific instructions that are repeatedly needed in this repo.

Rules:

- Keep files here stable. Do NOT store transient work product here.
- Put transient data, command output, fetched data, scratch notes, generated review findings, and other temporary artifacts in `.tmp`.
- Prefer updating existing topic docs over creating overlapping docs.
- Keep content optimized for agent consumption: terse, factual, repository-specific.
