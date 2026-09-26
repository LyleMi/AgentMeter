# Pricing Sources

Pricing registry rows are USD per 1M tokens. They are API list-price estimates
for local usage analysis only; subscription usage in Codex, Claude Code, or
other coding agents may not map one-to-one to API billing.

Latest refresh: 2026-09-26 (OpenAI, Anthropic, DeepSeek, Z.AI, xAI, and
new Gemini Flash rows). Other rows retain their previous verification dates.

## Source Of Truth

`internal/pricing/pricing_seed.go` is the seeded registry source of truth. The
seeded `Rate` rows in that file define the actual model aliases, normalized
model keys, rates, source strings, and effective dates inserted into
`pricing_models`. User-saved custom rows are stored in the same table with
`is_custom = 1` and are not overwritten by seeded registry updates.

This document records provider source links and assumptions only. Do not copy a
manual price table here; it will drift from the registry. When pricing changes:

- update `internal/pricing/pricing_seed.go` and, if needed, aliases in
  `internal/pricing/pricing.go`;
- update pricing tests when aliases or expected behavior change;
- update the verification date and assumptions in this file;
- run the validation appropriate for pricing changes from
  [Validation](validation.md).

## Sources

- OpenAI: https://developers.openai.com/api/docs/pricing
- OpenAI GPT-5.6 Sol: https://developers.openai.com/api/docs/models/gpt-5.6-sol
- OpenAI GPT-5: https://developers.openai.com/api/docs/models/gpt-5
- OpenAI GPT-5 mini: https://developers.openai.com/api/docs/models/gpt-5-mini
- OpenAI GPT-5 nano: https://developers.openai.com/api/docs/models/gpt-5-nano
- Anthropic Claude: https://platform.claude.com/docs/en/about-claude/pricing
- Google Gemini API: https://ai.google.dev/gemini-api/docs/pricing
- DeepSeek USD pricing: https://api-docs.deepseek.com/quick_start/pricing-details-usd
- DeepSeek V4 pricing: https://api-docs.deepseek.com/quick_start/pricing
- Z.AI pricing: https://docs.z.ai/guides/overview/pricing
- Kimi API Platform pricing: https://platform.kimi.ai/docs/pricing/chat-k26
- Mistral pricing: https://mistral.ai/pricing/
- Mistral chat endpoint cache note: https://docs.mistral.ai/api/endpoint/chat
- xAI pricing: https://docs.x.ai/developers/pricing
- Cohere pricing: https://cohere.com/pricing
- Alibaba Cloud Model Studio pricing: https://www.alibabacloud.com/help/en/model-studio/model-pricing
- Tencent Hy3 announcement: https://www.tencent.com/en-us/articles/2202320.html

## Assumptions

- The database schema supports input, cached input, and output rates. It does
  not support separate cache write, cache read, reasoning output, batch,
  priority, region, or context-window tiers.
- Anthropic cache input uses cache-hit/read pricing. Cache-creation write
  premiums are approximated as normal input because the parser stores cache
  creation tokens with input tokens.
- Mistral cached input is set to 10% of input, matching the published cached
  token rule in the chat endpoint docs.
- Providers without a published cached-input discount, such as Cohere and Qwen
  rows in this registry, use the normal input price for cached input to avoid
  undercounting.
- OpenAI and Gemini long-context rows are explicit aliases. The default model
  row uses the standard context tier because local session logs do not reliably
  expose the billable prompt-length tier.
- GPT-6 and GPT-5.6 use Standard processing and short-context rates by default.
  Explicit `-long-context` rows represent the published long-context tier.
  Cache input uses cache-read pricing; cache-write premiums are not represented.
  GPT-5.6 Sol uses the promotional rate available at least through 2026-11-21.
- Gemini 3.6/3.7/3.8 Flash use the promotional Standard rates through
  2026-12-31. Review them before 2027-01-01; seeds do not switch automatically.
- DeepSeek Flash (including the retired `deepseek-v4-flash` request name) and
  V4 Pro automatically apply peak/off-peak rates using request start time
  (completion time when start is missing). A request crossing a boundary keeps
  its start-time rate; tokens are not split proportionally by elapsed time.
  Peak windows are Beijing time Monday–Friday 09:00–12:00 and 14:00–18:00,
  excluding Chinese holiday periods; all other times cost half the peak rate.
  The 2026 holiday calendar follows the [State Council notice](https://www.gov.cn/gongbao/2025/issue_12406/202511/content_7048922.html).
  Weekend make-up workdays remain off-peak. Future/unlisted holiday years use
  normal weekday windows until their calendar is verified.
- Session costs and cache savings sum per-call amounts when call counters cover
  the session counters exactly. Incomplete call histories use session start time
  for the aggregate estimate; missing timestamps retain peak pricing. Custom
  rates and explicit `-off-peak` aliases are not automatically discounted.
  Current registry prices are used for historical usage too; this is a current
  list-price estimate, not a reconstruction of past tariff revisions. The local
  call start is a proxy for provider receipt time, which logs do not expose.
- GLM-5 and GLM-5.1 have their own published rates instead of aliases to GLM-5.2.
- New Claude Fable/Mythos 5.1 and Opus 5.5 rows use their model-specific cache
  read discounts, not a uniform percentage of input pricing.
- Gemini 3.1 Pro rows use the standard `<= 200k` tier. Explicit
  `-long-context` aliases use the `> 200k` tier. `gemini-3.1-pro` and
  `gemini-3.1-pro-preview` are treated as equivalent for this local estimate.
- Hy3 preview uses Tencent Cloud TokenHub starting USD rates from Tencent's
  launch announcement. Hosted-provider prices can differ.
- Codex automatic review usage has Codex plan/usage semantics but no separate
  public token API list price. Use a custom pricing row for local estimates
  when sessions contain `codex-auto-review`.
- Qwen prices are regional and tiered. The registry uses common global or
  international standard tiers where available, and the higher thinking-output
  rate where the published table separates thinking and non-thinking output.
- Open-weight models such as Llama are not included unless the model owner
  publishes a first-party token API price. Hosted prices for those models vary
  by inference provider.
