---
name: incident-triage
description: Guide the first 30 minutes of backend incident response for production issues such as latency spikes, error bursts, failed deploys, or partial outages.
---

# Incident Triage Skill

Use this skill when the user is dealing with a production incident and needs a practical first-response plan rather than a generic explanation.

## Goal

Stabilize the situation, narrow the blast radius, and produce a concrete next-action plan for an on-call engineer or incident commander.

## Inputs To Look For

- Service or API name
- Symptoms such as latency, error rate, saturation, or customer-visible failures
- What changed recently: deploy, config change, traffic spike, dependency issue, or infrastructure event
- Current severity, blast radius, and affected customers or regions
- Known observability signals: logs, traces, dashboards, alerts, and recent rollouts

If details are missing, state the assumptions explicitly and continue with the most likely triage path.

## Response Shape

Produce:

1. A one-sentence incident summary
2. The most likely failure domains to check first
3. A first 30 minute action plan in priority order
4. Containment or rollback options
5. What evidence would confirm or eliminate each hypothesis
6. A short status update suitable for Slack or incident chat

## Triage Priorities

- Start with user impact and current severity.
- Prefer the fastest safe containment step before deep root-cause analysis.
- Check recent changes first because deploys and config changes are common triggers.
- Distinguish between application issues, dependency issues, and infrastructure saturation.
- Call out when rollback is lower risk than continued live debugging.

## Investigation Checklist

- Compare current metrics against the previous stable window.
- Check whether the problem is global or limited to one region, cluster, AZ, tenant, or endpoint.
- Identify whether latency is caused by CPU, memory, lock contention, queueing, downstream calls, or retries.
- Look for correlated spikes in error rate, timeout rate, or dependency saturation.
- Verify whether autoscaling, circuit breakers, caches, or rate limits changed behavior during the incident.

## Decision Rules

- If customer impact is active and a recent deploy matches the timeline, recommend rollback early.
- If only one dependency is degraded, isolate the dependency and reduce pressure on it.
- If the system is saturated, recommend load shedding, scaling, or disabling non-critical work.
- If evidence is weak, prefer reversible mitigations and tighter observation windows.

## Tone

- Be concise, operational, and specific.
- Avoid generic incident-management advice that does not change the next action.
- Treat the output like guidance for an engineer who needs to act now.