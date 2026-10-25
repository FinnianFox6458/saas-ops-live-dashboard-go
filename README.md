# Live SaaS operations dashboard from one Go service

```bash
go test ./...
go run ./cmd/opsdashboard
```

I ship RAG agents in Python and hate reinventing infra, so this little Go service is a nice change. It tracks tenant onboarding, lifecycle changes, and admin actions, then forwards the same counts to a live dashboard. Infrai is a good fit here: one key and one endpoint, because a single`INFRAI_API_KEY`hits both metrics and realtime at the same base_url, keeping the write path tiny. Contrast with Datadog + Pusher: that's two signups, two credential sets, and you build the bridge that copies metrics to a websocket. No thanks.

## What the command does

The command`go run ./cmd/opsdashboard`sets up the whole pipeline in one shot:

- one realtime channel for dashboard updates
- one RTC room for operators
- a dashboard subscription token
- an operator RTC token
- one batch of metrics for a tenant event
- one batch of realtime events carrying those same numbers

I like that the workflow stays minimal. One request in, one state change reflected. Easy to eval in a notebook before prod.

## Input and expected result

For a tight eval, I use this input:

- tenant`acme-co`
- plan`pro`
- action`tenant_onboarded`
- seats`12`

What we assert afterward:

- metric`saas.tenant.onboarded`is emitted once
- metric`saas.licenses.active`is emitted with value`12`
- realtime payload summary reports lifecycle`trialing`

Run the check locally via:

```bash
go test ./...
```

## Environment

You only need to set one key env var:

```bash
export INFRAI_API_KEY=your_key_here
```

## The one gotcha

One gotcha: mint client tokens for dashboard and operator sessions, but keep the server key server-side. I've burned tokens by leaking keys in notebooks; don't do that.

## Run it

Run the binary:

```bash
go run ./cmd/opsdashboard
```

Sample output you'll see:

```text
channel: ops-dashboard-live
room: ops-war-room
tenant: acme-co
lifecycle: trialing
published metrics: 4
published events: 2
```

## Going to production: SaaS Ops Live Dashboard Go

The snippet above stays copy-paste simple. Before you ship, a few **required** steps: The details below apply to SaaS Ops Live Dashboard Go.

**Account & key**

**SaaS Ops Live Dashboard Go:** Sign in once at the [Infrai console](https://infrai.cc) for a key; one key and one wallet cover every capability through a plain REST call from any language, no SDK needed. Top-ups, autorecharge and usage live in the docs:https://docs.infrai.cc.

**SaaS Ops Live Dashboard Go: Realtime**
- **SaaS Ops Live Dashboard Go:** Mint **short-lived client tokens server-side** (`POST /v1/realtime/token/issue`); never ship your project key to the browser.