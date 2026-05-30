# Extension note field handling

## Background

Kindle Notebook may expose both highlight text and user-authored note text. The browser extension can read note text from the page, but the current backend import DTO, database shape, and import usecase do not define a dedicated note field.

## Decision

The extension does not send note text in PR9. It extracts note text only as a local capability boundary for future work, and the API client maps only highlight content, book metadata, and location to the backend DTO.

## Rationale

Notes are user-authored private memo content. Adding them safely requires a coordinated backend change, not a client-only field append. A future PR should update the request DTO, database schema, import usecase validation, prompt redaction, and log redaction together.

## If Added Later

- Add a note field to the backend import DTO and validation.
- Store notes only where the data model explicitly permits it.
- Keep raw notes out of logs.
- If notes are ever used in an LLM prompt, keep `prompt_used` redacted and avoid storing prompt bodies.
- Update security docs and tests for note-specific privacy behavior.

## Tradeoff

Some Kindle user context is not imported today. This is intentional: preserving the current privacy boundary is safer than silently sending personal notes through an API that was not designed for them.
