# UI Design

AgentMeter should feel like a professional local diagnostic dashboard: useful
first, visually calm, and fast to scan. The UI is not a landing page, marketing
site, or decorative analytics demo. It should help a developer understand local
coding-agent usage, find anomalies, and drill into source sessions with minimal
friction.

## Product Direction

- Prioritize diagnosis over presentation. Each screen should make the next
  useful question obvious: what changed, what cost money, what took time, what
  failed to parse, and where can I inspect the source.
- Keep the interface local-first. Avoid language or patterns that imply cloud
  sync, team monitoring, telemetry, or hosted observability unless those
  features are explicitly added.
- Prefer clear hierarchy over visual novelty. A screen should read as title,
  scope, primary numbers, filters or controls, then details.
- Use restrained modern styling. Keep the UI quiet, precise, and tool-like,
  with enough contrast and spacing to separate regions without making every
  section look like a promotional card.

## Usefulness Rules

- Put the main answer near the top of each view. For example, Overview should
  expose usage totals and trend context before secondary breakdowns.
- Make drill-down paths visible. Sessions, tools, models, projects, and source
  paths should lead to the most specific useful record when data exists.
- Preserve source traceability. When showing derived values, keep enough context
  to understand which agent, session, file, model, or timestamp produced them.
- Label uncertainty directly. Unknown, missing, estimated, stale, and parse
  error states should not look like zero values.
- Avoid passive dashboards that only show totals. Add sorting, filtering,
  grouping, and inspection affordances where they materially improve diagnosis.

## Layout Hierarchy

- Use stable page structure across dashboard screens: page header, concise
  summary region, controls, primary content, and supporting details.
- Keep controls close to the data they affect. Filters should not be separated
  from the table, chart, or list they change.
- Use width intentionally. Wide layouts can compare summaries, charts, and
  tables, but the most important path must still work on narrow screens.
- Avoid nested cards and decorative containers. Use cards for repeated records,
  modal content, or genuinely framed tools, not as the default page section.
- Keep layout dimensions stable when data updates. Loading text, badges,
  filters, hover states, and action buttons should not resize tables or shift
  summaries unexpectedly.

## Visual Quality

- Use a restrained palette with semantic color for status, warnings, errors,
  and selected states. Do not let decorative gradients or one dominant hue carry
  the product identity.
- Keep typography compact and readable. Reserve large type for page-level
  summary numbers or headings, not dense panels and table cells.
- Align numbers, units, and timestamps consistently. Usage, cost, duration, and
  count values should be easy to compare across rows.
- Use icons for familiar tool actions when the existing UI pattern supports
  them, and pair unfamiliar icons with accessible labels or tooltips.
- Maintain contrast and focus states for keyboard and low-vision use. Visual
  polish should not hide state or reduce readability.

## Components And CSS

- Reuse existing Vue components, layout wrappers, buttons, form controls,
  badges, tables, charts, and CSS variables before adding new patterns.
- Add shared components when a pattern appears on multiple screens or carries a
  product meaning such as parse status, pricing status, source agent, or model.
- Keep component APIs narrow and data-shaped. Presentation components should not
  duplicate pricing, parsing, filtering, or duration logic from backend read
  models.
- Avoid one-off colors, spacing, border radii, shadows, and table styles. If a
  new visual token is needed, name it for its role rather than for a single
  screen.
- New interactive components should define normal, hover, focus, selected,
  disabled, loading, empty, and error behavior when those states apply.

## Analytical Signals And Charts

- Derived signal screens should lead with configurable dynamic charts. Model
  Signals, trend, health, and efficiency views should let users switch metric
  lenses instead of forcing them to compare numeric table columns first.
- Chart controls should stay close to the chart and preserve page filters. When
  relevant, expose metric, dimension, time/project grouping, and current versus
  baseline comparison as explicit controls.
- Charts should label low sample, missing price, missing baseline, and
  unavailable denominator states directly. Do not draw missing values as zero.
- Tables remain the drill-down and traceability layer after the chart. They are
  appropriate for source identity, exact timestamps, exact counts, and links to
  sessions or files.

## Dense Data Tables

- Tables are the default for dense inspectable records such as sessions, tool
  calls, raw model rows, project rows, and pricing data. Aggregated signal
  pages should lead with charts and keep tables as inspectable rows.
- Keep important columns visible: identity, time range, agent or model, usage,
  cost, duration, status, and drill-down action where relevant.
- Right-align numeric columns and keep units consistent. Do not mix raw token
  counts, compact labels, and formatted currency in the same column without a
  clear reason.
- Sort and filter behavior should be explicit and predictable. Defaults should
  match user intent, usually most recent or highest-impact records first.
- Long paths, prompts, model names, and error strings should truncate cleanly
  with a way to inspect the full value.
- Empty tables should explain whether no data exists, filters removed all
  results, indexing has not run, or parsing failed.

## Status, Empty, And Loading States

- Distinguish indexing state, parse state, pricing state, and privacy setting
  state. These are different product facts and should not share a vague
  "warning" treatment.
- Show useful next steps only when the action is safe and relevant. Routine
  smoke checks must not encourage state-changing actions such as rebuilding the
  index or saving settings unless the task is specifically about that flow.
- Loading states should preserve layout shape and avoid hiding existing data
  unless the old data is no longer valid.
- Empty states should be short and specific. Name the likely cause and the next
  read-only inspection path when possible.
- Error states should include enough context to debug locally without requiring
  uploads or external services.

## Validation Expectations

For Web UI changes:

- Run the normal frontend validation for the scope, usually the frontend build,
  unless the change is documentation-only.
- Check at least one wide and one narrow viewport for overflow, clipped text,
  unstable layout, and unreadable dense tables.
- Verify loading, empty, error, and populated states when the changed component
  can show them.
- Compare displayed totals, filters, status labels, and drill-down semantics
  against backend query expectations and the TUI contract when shared behavior
  changes.
- Keep screenshots or PR notes focused on changed behavior and validation
  performed, especially for visible layout or table changes.
