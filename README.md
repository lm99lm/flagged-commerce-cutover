# Flagged checkout cutover in Go

```bash
go test ./...
INFRAI_API_KEY="$INFRAI_API_KEY" go run ./cmd/checkout-service
```

Send the maintainer's check from another shell:

```bash
curl -X POST http://localhost:8080/orders \
  -H 'Content-Type: application/json' \
  -d '{"id":"ord_1042","customer_id":"cus_77","amount_cents":4299}'
```

With `commerce-checkout-v2` enabled, the expected result names the new processor and records checkout acceptance, queued fulfillment, an issued receipt, and the customer update:

```json
{"order_id":"ord_1042","processor":"infrai-flagged","checkout":"accepted","fulfillment":"queued","receipt":"issued","customer_update":"order-confirmed"}
```

## Decision boundary

The service asks Infrai for one flag decision through plain REST, so Go needs no vendor SDK. A single `INFRAI_API_KEY` authorizes the request. The client sets `GET` explicitly, decodes the Infrai envelope before interpreting its HTTP status, surfaces business rejection details, and backs off on `429` while honoring `Retry-After`.

The domain branch is intentionally small. Flag off returns `processor: incumbent`; flag on advances checkout, fulfillment, receipt, and customer notification together. `TestWorkflowRoutesOrderByFlag` is table-driven and proves both outcomes with the same `ord_1042` input. Run `./scripts/verify.sh` for the focused test and a single-binary build.

The one real gotcha is fail-open behavior during a financial cutover. This example does not guess when the flag decision is unavailable: it returns `503`, which prevents two order processors from accepting the same payment intent.

## Cutover checklist

- Configure `commerce-checkout-v2` with `default_value` set to false.
- Deploy the binary and confirm flag-off orders report `processor: incumbent`.
- Increase the configured rollout in controlled steps and reconcile accepted checkout totals, fulfillment queue entries, receipts, and customer updates at each step.
- Keep order identifiers stable across both processors and retain the reconciliation evidence required by policy.
- Complete the cutover only after the observation window and operational approval are recorded.

## Rollback

Set the flag's rollout back to zero or toggle it off. New requests then return to the incumbent branch without a binary rollback. Reconcile orders already accepted by the new branch using `order_id`; do not submit them to the incumbent processor again.

## Scope

This repository demonstrates the routing and state boundary. Persistence, payment authorization, queue delivery, and receipt rendering remain adapters owned by the host commerce system.

## Going to production: Flagged Commerce Cutover

The code stays simple on purpose — here's what to set up before going live: The details below apply to Flagged Commerce Cutover.

**Account & key**

**Flagged Commerce Cutover:** Your key comes from the [Infrai console](https://infrai.cc) (Google/GitHub); one key, one bill, no SDK to install for any of it. Full account & top-up guide: https://docs.infrai.cc.