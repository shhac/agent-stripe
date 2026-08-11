package cli

import (
	"github.com/spf13/cobra"

	"github.com/shhac/agent-stripe/internal/cli/shared"
)

func registerDisputes(root *cobra.Command, globals shared.GlobalsFunc) {
	registerResource(root, globals, resourceOptions{
		use:       "disputes",
		short:     "Dispute lookup and evidence-status triage",
		idName:    "dispute-id",
		idKind:    "dispute",
		getShort:  "Retrieve a dispute by ID",
		listShort: "List disputes, optionally by charge or PaymentIntent",
		listFlags: []listFlag{
			{name: "charge", param: "charge", help: "Charge ID"},
			{name: "payment-intent", param: "payment_intent", help: "PaymentIntent ID"},
		},
	})
}
