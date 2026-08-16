package checkout

import "context"

type FlagReader interface {
	IsEnabled(context.Context, string) (bool, error)
}

type Order struct {
	ID          string `json:"id"`
	CustomerID  string `json:"customer_id"`
	AmountCents int64  `json:"amount_cents"`
}

type OrderResult struct {
	OrderID        string `json:"order_id"`
	Processor      string `json:"processor"`
	Checkout       string `json:"checkout"`
	Fulfillment    string `json:"fulfillment"`
	Receipt        string `json:"receipt"`
	CustomerUpdate string `json:"customer_update"`
}

type Workflow struct {
	Flags   FlagReader
	FlagKey string
}

func (w Workflow) Process(ctx context.Context, order Order) (OrderResult, error) {
	enabled, err := w.Flags.IsEnabled(ctx, w.FlagKey)
	if err != nil {
		return OrderResult{}, err
	}
	if !enabled {
		return OrderResult{OrderID: order.ID, Processor: "incumbent", Checkout: "accepted"}, nil
	}
	return OrderResult{
		OrderID: order.ID, Processor: "infrai-flagged", Checkout: "accepted",
		Fulfillment: "queued", Receipt: "issued", CustomerUpdate: "order-confirmed",
	}, nil
}
