---
name: eino-intro-outline
description: Build a practical outline for introductory Eino content aimed at Go backend engineers, especially when the answer should compare chat, tools, MCP, skills, agents, and orchestration.
---

# Eino Intro Outline Skill

Use this skill when the user wants an introduction, blog outline, talk outline, or framing document for Eino.

## Audience

Default audience: Go backend engineers who already understand APIs, services, and typed interfaces, but are new to Eino.

## Output Shape

Produce:

1. A clear title
2. A one-paragraph thesis
3. A sectioned outline in a practical order
4. A short note on what to demo first

## Framing Rules

- Start with what Eino is in one sentence before discussing advanced features.
- Explain why Eino feels natural for Go teams: explicit interfaces, typed messages, component boundaries, and production-oriented runtime behavior.
- Include at least one section each for tools, agents, orchestration, and developer-facing tradeoffs.
- If the user asks for concrete examples, include basic chat, tool calling, MCP integration, and skills.
- If mentioning MCP, frame it as connecting an MCP server and surfacing its tools inside Eino rather than as a separate agent runtime.
- If mentioning skills, explain that skills are reusable instruction packs loaded via middleware and backed by SKILL.md files.

## Recommended Section Order

1. Why Go teams should care
2. Smallest useful example
3. Tool calling
4. MCP for external systems
5. Skills for reusable guidance
6. Agents and orchestration
7. When Eino is a strong fit
8. When a thinner wrapper is enough

## Tone

- Keep the structure crisp and technical.
- Prefer concrete tradeoffs over marketing language.
- Keep claims grounded in what the framework actually provides.