# AviaSurveil360 Harness Coverage

This inventory covers the 31 canonical capabilities and the local semantic
boundary. `verified` is limited to the named local routes. `N/A` records a
deliberate non-adoption; candidate runtime and certification blockers remain
explicit in [certification.md](certification.md), not silently promoted.

| Source principle or capability | Repository implementation | Required evidence | Status and reason |
| --- | --- | --- | --- |
| Humans set intent; agents execute within authority | [operating-loop.md](operating-loop.md) and `AGENTS.md` | Scoped authority route | verified — native smoke test exercises the map. |
| Break large goals into reusable design, code, review, test, and verification steps | [planning](../PLANS.md) and active plans | Restartable milestones | verified — plan index and lifecycle remain native. |
| Agents can self-review and respond to feedback | [operating loop](operating-loop.md) and [output](output-contract.md) | Correction route | verified — harness maintenance is repeatable. |
| Application behavior is directly readable | Static demo and candidate source | Focused behavior proof | verified — verification matrix names direct local tests. |
| Logs, metrics, and traces are queryable when relevant | Candidate operations docs | Query/correlation proof | N/A — this harness does not own a telemetry stack. |
| Repository knowledge is the durable record | [docs index](../index.md) | Resolving canonical routes | verified — harness smoke validates links and facts. |
| Repository tools and authorized work context are directly invocable | [registry](registry.md) | Command discovery | verified — direct Node, Make, and repository commands are routed. |
| Dependencies and abstractions remain agent-legible | `ARCHITECTURE.md` and registry | Discoverable contracts | verified — legacy, candidate, Auth, and Data boundaries are explicit. |
| `AGENTS.md` is a concise map, not an encyclopedia | `../../AGENTS.md` | Scannable routing | verified — canonical routes are link-tested. |
| Plans are versioned living artifacts | [planning](../PLANS.md) and plan index | Current plan lifecycle | verified — readiness plan/sign-off lifecycle is retained. |
| Architecture and critical taste boundaries are mechanical | `ARCHITECTURE.md` and smoke tests | Actionable boundary check | verified — semantic smoke protects legacy/candidate distinction. |
| Local autonomy exists inside enforced central boundaries | [operating loop](operating-loop.md) | Escalation/recovery path | verified — external, commit, HMAC, and Data stops are literal. |
| Verification proves working behavior, not only code changes | [verification matrix](verification-matrix.md) | Exact local commands | verified — checks are selected by changed risk surface. |
| Failures and review judgment feed back into the harness | [operating loop](operating-loop.md) | Durable correction route | verified — recurring gaps route to plan, test, or debt. |
| Entropy and technical debt are continuously controlled | [entropy checklist](entropy-cleanup-checklist.md) | Dated/owned follow-up | verified — drift and retired Data boundary are tracked. |
| Autonomy increases only after test, review, recovery, and escalation loops exist | [operating loop](operating-loop.md) | Higher authority unavailable | verified — certification stays blocked rather than inferred. |
| Merge throughput policy matches project risk | Repository-local review policy | Risk rationale | N/A — this harness owns no merge automation. |
| Release, deployment, and production actions require repository-local authority | [certification](certification.md) | Denial/escalation rule | N/A — no deployment or release action is authorized here. |
| Repository-specific OpenAI examples are treated as options, not universal mandates | Decision ledger below | Independent local dispositions | verified — all case-study choices are assessed individually. |

## Case-study decision ledger

| OpenAI case-study choice | Local decision or implementation | Required evidence | Status and reason |
| --- | --- | --- | --- |
| Zero human-authored code as an operating constraint | Rejected | Responsibility model | N/A — no zero-human-code constraint applies. |
| Reported repository size, pull-request throughput, elapsed-time speedup, and long agent-run duration as targets | Context only | Quality outcome target | N/A — vanity and duration metrics are not adopted. |
| Local and cloud agent review loops continue until reviewers are satisfied while human review is optional | Local review only | Review stopping condition | N/A — no cloud review-loop authority is configured. |
| Per-worktree application isolation | Local candidate/worktree isolation | Collision-free proof | N/A — no independent worktree runtime is owned by this harness. |
| Per-worktree observability stack | Not adopted | Isolated query proof | N/A — no observability stack is owned here. |
| Chrome DevTools Protocol for UI control | Optional local browser tooling | Browser-flow proof | N/A — direct browser QA is sufficient when UI work requires it. |
| Victoria Logs, Metrics, and Traces with LogQL/PromQL/TraceQL | Not adopted | Actual telemetry query | N/A — no Victoria stack exists in this scope. |
| OpenAI's fixed layered domain architecture | Project-native boundaries | Executable boundary proof | N/A — external fixed layer names are not copied. |
| Reimplementing upstream dependency behavior locally | Avoided | Tradeoff record | N/A — Auth and platform dependencies retain their own contracts. |
| Minimally blocking merge gates and short-lived pull requests | Not owned | Risk rationale | N/A — merge policy is outside repository harness authority. |
| Scheduled Codex documentation gardening and quality-scoring agents open targeted repair pull requests | Not adopted | Cadence/write authority | N/A — schedulers and GitHub writes are prohibited. |
| Automated merge and agent-authored release tooling | Not adopted | Automation-gate rationale | N/A — release automation is outside authority. |
| Federated certification handoff | [certification](certification.md) | Exact child/fixture/Data boundaries | N/A — candidate-only local work is blocked from certification without authority. |
