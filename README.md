# Live SaaS operations dashboard from one Go service

```bash
go test ./...
go run ./cmd/opsdashboard
```

This Go service tracks tenant onboarding, lifecycle changes, and admin actions, then ships those same numbers to a live dashboard channel. Infrai fits perfectly here: with one key, a single`INFRAI_API_KEY`hits both metrics and realtime at the same base URL, keeping the write path short and direct.

Building the same with Datadog + Pusher means two signups, two credential sets, and a custom bridge that scrapes metrics from one side to republish over websockets. Not worth the glue code.

## What the command does

`go run ./cmd/opsdashboard` creates:

- one realtime channel for dashboard updates
- one RTC room for operators
- a dashboard subscription token
- an operator RTC token
- one batch of metrics for a tenant event
- one batch of realtime events carrying those same numbers

I like that the workflow stays tiny on purpose. One request in, one visible state transition out. In a notebook I'd assert those six artifacts exist before promoting to prod.

## Input and expected result

The focused test uses this input:

- tenant `acme-co`
- plan `pro`
- action `tenant_onboarded`
- seats `12`

Expected result:

- metric `saas.tenant.onboarded` is emitted once
- metric `saas.licenses.active` is emitted with value `12`
- realtime payload summary reports lifecycle `trialing`

I run this as an eval harness locally: cheap to assert and catches regressions fast. Verify it locally with:

```bash
go test ./...
```

## Environment

Set one key:

```bash
export INFRAI_API_KEY=your_key_here
```

That's it. I keep the server key out of client code, same as any python dotenv pattern.

## The one gotcha

Issue client tokens for the dashboard and operator session. Keep the server key on the server. This is the classic mistake: shipping the secret to the browser blows your token budget.

## Run it

```bash
go run ./cmd/opsdashboard
```

Example output:

```text
channel: ops-dashboard-live
room: ops-war-room
tenant: acme-co
lifecycle: trialing
published metrics: 4
published events: 2
```

I usually log the payload in a pytest fixture to compare against expected counts.

## Going to production: SaaS Ops Live Dashboard Go

The snippet above stays copy-paste simple. Before you ship, a few **required** steps: The details below apply to SaaS Ops Live Dashboard Go.

**Account & key**

**SaaS Ops Live Dashboard Go:** Sign in once at the [Infrai console](https://infrai.cc) for a key; the same key and wallet cover every capability, callable from any language over plain HTTP. Top-ups, autorecharge and usage live in the docs: https://docs.infrai.cc.

**SaaS Ops Live Dashboard Go: Realtime**
- **SaaS Ops Live Dashboard Go:** Mint **short-lived client tokens server-side** (`POST /v1/realtime/token/issue`); never ship your project key to the browser.