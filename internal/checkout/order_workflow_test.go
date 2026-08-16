package checkout

import (
	"context"
	"testing"
)

type fixedFlag bool

func (f fixedFlag) IsEnabled(context.Context, string) (bool, error) { return bool(f), nil }

func TestWorkflowRoutesOrderByFlag(t *testing.T) {
	tests := []struct {
		name        string
		enabled     bool
		processor   string
		fulfillment string
	}{
		{name: "flag off keeps incumbent", processor: "incumbent"},
		{name: "flag on completes new order flow", enabled: true, processor: "infrai-flagged", fulfillment: "queued"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workflow := Workflow{Flags: fixedFlag(tt.enabled), FlagKey: "commerce-checkout-v2"}
			got, err := workflow.Process(context.Background(), Order{ID: "ord_1042", CustomerID: "cus_77", AmountCents: 4299})
			if err != nil {
				t.Fatal(err)
			}
			if got.Processor != tt.processor || got.Fulfillment != tt.fulfillment {
				t.Fatalf("processor=%q fulfillment=%q", got.Processor, got.Fulfillment)
			}
		})
	}
}
