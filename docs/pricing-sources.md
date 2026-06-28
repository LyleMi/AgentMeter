# Pricing Sources

Pricing registry rows are USD per 1M tokens. They are API list-price estimates
for local usage analysis only; subscription usage in Codex, Claude Code, or
other coding agents may not map one-to-one to API billing.

Verified: 2026-06-29.

## Source Of Truth

`internal/pricing/pricing.go` is the pricing registry source of truth. The
seeded `Rate` rows in that file define the actual model aliases, normalized
model keys, rates, source strings, and effective dates inserted into
`pricing_models`. User-saved custom rows are stored in the same table with
`is_custom = 1` and are not overwritten by seeded registry updates.

This document records provider source links and assumptions only. Do not copy a
manual price table here; it will drift from the registry. When pricing changes:

- update `internal/pricing/pricing.go`;
- update pricing tests when aliases or expected behavior change;
- update the verification date and assumptions in this file;
- run the validation appropriate for pricing changes from
  [Validation](validation.md).

## Sources

- OpenAI: https://developers.openai.com/api/docs/pricing
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
