package mockstripe

// v2Accounts returns fully-populated v2.core.account fixtures. Handlers strip
// them down to the requested `include` set, mirroring Stripe's sparse
// responses where an omitted field is null rather than absent.
func v2Accounts() []map[string]any {
	return []map[string]any{
		v2AccountActive(),
		v2AccountRestricted(),
		v2AccountRecipient(),
		v2AccountClosed(),
	}
}

func v2AccountActive() map[string]any {
	return map[string]any{
		"id":                     "acct_mock_v2_active",
		"object":                 "v2.core.account",
		"applied_configurations": []any{"customer", "merchant"},
		"closed":                 false,
		"contact_email":          "owner@furever.example.com",
		"created":                "2026-01-14T10:04:11.000Z",
		"dashboard":              "full",
		"display_name":           "Furever Grooming",
		"livemode":               false,
		"metadata":               map[string]any{"tenant_id": "acme"},
		"configuration": map[string]any{
			"customer": map[string]any{
				"capabilities": map[string]any{
					"automatic_indirect_tax": map[string]any{"status": "active", "status_details": []any{}},
				},
			},
			"merchant": map[string]any{
				"applied": "2026-01-14T10:09:02.000Z",
				"capabilities": map[string]any{
					"card_payments": map[string]any{"status": "active", "status_details": []any{}},
					"stripe_balance": map[string]any{
						"payouts": map[string]any{"status": "active", "status_details": []any{}},
					},
				},
				"statement_descriptor": map[string]any{"descriptor": "FUREVER"},
			},
		},
		"identity": map[string]any{
			"country":     "us",
			"entity_type": "company",
			"business_details": map[string]any{
				"registered_name": "Furever Inc",
				"address":         map[string]any{"country": "us", "postal_code": "94080"},
			},
		},
		"requirements": map[string]any{
			"collector": "stripe",
			"entries":   []any{},
			"summary": map[string]any{
				"minimum_deadline": map[string]any{"status": "eventually_due", "time": nil},
			},
		},
		"future_requirements": map[string]any{
			"collector": "stripe",
			"entries":   []any{},
		},
		"defaults": map[string]any{
			"currency": "usd",
			"locales":  []any{"en-US"},
			"responsibilities": map[string]any{
				"fees_collector":         "stripe",
				"losses_collector":       "stripe",
				"requirements_collector": "stripe",
			},
		},
	}
}

func v2AccountRestricted() map[string]any {
	return map[string]any{
		"id":                     "acct_mock_v2_restricted",
		"object":                 "v2.core.account",
		"applied_configurations": []any{"customer", "merchant"},
		"closed":                 false,
		"contact_email":          "ops@pawsome.example.com",
		"created":                "2026-05-02T08:31:45.000Z",
		"dashboard":              "express",
		"display_name":           "Pawsome Walkers",
		"livemode":               false,
		"metadata":               map[string]any{},
		"configuration": map[string]any{
			"customer": map[string]any{
				"capabilities": map[string]any{
					"automatic_indirect_tax": map[string]any{"status": "active", "status_details": []any{}},
				},
			},
			"merchant": map[string]any{
				"applied": "2026-05-02T08:40:00.000Z",
				"capabilities": map[string]any{
					"card_payments": map[string]any{
						"status": "restricted",
						"status_details": []any{
							map[string]any{"code": "requirements_past_due", "resolution": "provide_info"},
						},
					},
					"stripe_balance": map[string]any{
						"payouts": map[string]any{
							"status": "restricted",
							"status_details": []any{
								map[string]any{"code": "requirements_past_due", "resolution": "provide_info"},
							},
						},
					},
				},
			},
		},
		"identity": map[string]any{
			"country":     "us",
			"entity_type": "company",
			"business_details": map[string]any{
				"address": map[string]any{"country": "us"},
			},
		},
		"requirements": map[string]any{
			"collector": "stripe",
			"entries": []any{
				map[string]any{
					"id":                   "reqent_mock_registered_name",
					"description":          "identity.business_details.registered_name",
					"awaiting_action_from": "user",
					"minimum_deadline":     map[string]any{"status": "past_due"},
					"errors":               []any{},
					"impact": map[string]any{
						"restricts_capabilities": []any{
							map[string]any{
								"capability":    "card_payments",
								"configuration": "merchant",
								"deadline":      map[string]any{"status": "past_due"},
							},
							map[string]any{
								"capability":    "stripe_balance.payouts",
								"configuration": "merchant",
								"deadline":      map[string]any{"status": "past_due"},
							},
						},
					},
					"requested_reasons": []any{map[string]any{"code": "routine_onboarding"}},
				},
				map[string]any{
					"id":                   "reqent_mock_representative",
					"description":          "relationship.representative",
					"awaiting_action_from": "user",
					"minimum_deadline":     map[string]any{"status": "currently_due"},
					"errors":               []any{},
					"impact": map[string]any{
						"restricts_capabilities": []any{
							map[string]any{
								"capability":    "card_payments",
								"configuration": "merchant",
								"deadline":      map[string]any{"status": "currently_due"},
							},
						},
					},
					"reference":         map[string]any{"type": "person", "resource": "person_mock_representative"},
					"requested_reasons": []any{map[string]any{"code": "routine_onboarding"}},
				},
				map[string]any{
					"id":                   "reqent_mock_ein",
					"description":          "identity.business_details.id_numbers.us_ein",
					"awaiting_action_from": "stripe",
					"minimum_deadline":     map[string]any{"status": "currently_due"},
					"errors": []any{
						map[string]any{
							"code":        "verification_document_name_mismatch",
							"description": "The name on the account does not match the name on the document.",
						},
					},
					"impact":            map[string]any{"restricts_capabilities": []any{}},
					"requested_reasons": []any{map[string]any{"code": "routine_verification"}},
				},
			},
			"summary": map[string]any{
				"minimum_deadline": map[string]any{"status": "past_due", "time": "2026-07-30T00:00:00.000Z"},
			},
		},
		"future_requirements": map[string]any{
			"collector": "stripe",
			"entries":   []any{},
		},
		"defaults": map[string]any{
			"currency": "usd",
			"locales":  []any{"en-US"},
			"responsibilities": map[string]any{
				"fees_collector":         "application",
				"losses_collector":       "application",
				"requirements_collector": "stripe",
			},
		},
	}
}

func v2AccountRecipient() map[string]any {
	return map[string]any{
		"id":                     "acct_mock_v2_recipient",
		"object":                 "v2.core.account",
		"applied_configurations": []any{"recipient"},
		"closed":                 false,
		"contact_email":          "payouts@sitters.example.com",
		"created":                "2026-06-19T16:02:00.000Z",
		"dashboard":              "none",
		"display_name":           "Sitters Collective",
		"livemode":               false,
		"metadata":               map[string]any{},
		"configuration": map[string]any{
			"recipient": map[string]any{
				"capabilities": map[string]any{
					"bank_accounts": map[string]any{
						"local": map[string]any{
							"status": "restricted",
							"status_details": []any{
								map[string]any{"code": "determining_status", "resolution": "provide_info"},
							},
						},
					},
					"stripe_balance": map[string]any{
						"stripe_transfers": map[string]any{"status": "active", "status_details": []any{}},
					},
				},
			},
		},
		"identity": map[string]any{
			"country":     "gb",
			"entity_type": "individual",
		},
		"requirements": map[string]any{
			"collector": "application",
			"entries": []any{
				map[string]any{
					"id":                   "reqent_mock_bank_account",
					"description":          "configuration.recipient.default_outbound_destination",
					"awaiting_action_from": "user",
					"minimum_deadline":     map[string]any{"status": "eventually_due"},
					"errors":               []any{},
					"impact": map[string]any{
						"restricts_capabilities": []any{
							map[string]any{
								"capability":    "bank_accounts.local",
								"configuration": "recipient",
								"deadline":      map[string]any{"status": "eventually_due"},
							},
						},
					},
					"requested_reasons": []any{map[string]any{"code": "routine_onboarding"}},
				},
			},
			"summary": map[string]any{
				"minimum_deadline": map[string]any{"status": "eventually_due", "time": nil},
			},
		},
		"future_requirements": map[string]any{"collector": "application", "entries": []any{}},
		"defaults": map[string]any{
			"currency": "gbp",
			"responsibilities": map[string]any{
				"fees_collector":   "application",
				"losses_collector": "application",
			},
		},
	}
}

func v2AccountClosed() map[string]any {
	return map[string]any{
		"id":                     "acct_mock_v2_closed",
		"object":                 "v2.core.account",
		"applied_configurations": []any{"customer"},
		"closed":                 true,
		"contact_email":          "closed@example.com",
		"created":                "2025-11-01T09:00:00.000Z",
		"dashboard":              "none",
		"display_name":           "Wound Down Ltd",
		"livemode":               false,
		"metadata":               map[string]any{},
		"configuration": map[string]any{
			"customer": map[string]any{
				"capabilities": map[string]any{
					"automatic_indirect_tax": map[string]any{"status": "inactive", "status_details": []any{}},
				},
			},
		},
		"identity":            map[string]any{"country": "us", "entity_type": "company"},
		"requirements":        map[string]any{"collector": "stripe", "entries": []any{}},
		"future_requirements": map[string]any{"collector": "stripe", "entries": []any{}},
		"defaults":            map[string]any{"currency": "usd"},
	}
}

// v2AccountPersons keys persons by the account they belong to. Only the
// restricted account has any, matching its outstanding representative
// requirement.
func v2AccountPersons(accountID string) []map[string]any {
	if accountID != "acct_mock_v2_restricted" {
		return nil
	}
	return []map[string]any{
		{
			"id":         "person_mock_representative",
			"object":     "v2.core.account_person",
			"account":    accountID,
			"created":    "2026-05-02T08:35:12.000Z",
			"updated":    "2026-07-14T11:20:04.000Z",
			"given_name": "Jenny",
			"surname":    "Rosen",
			"email":      "jenny.rosen@example.com",
			"address": map[string]any{
				"city":        "Brothers",
				"country":     "us",
				"line1":       "27 Fredrick Ave",
				"postal_code": "97712",
				"state":       "OR",
			},
			"date_of_birth":        map[string]any{"day": 28, "month": 1, "year": 1988},
			"id_numbers":           []any{map[string]any{"type": "us_ssn_last_4"}},
			"additional_addresses": []any{},
			"additional_names":     []any{},
			"nationalities":        []any{"us"},
			"relationship": map[string]any{
				"owner":             true,
				"representative":    true,
				"percent_ownership": "0.8",
				"title":             "CEO",
			},
			"metadata": map[string]any{},
		},
		{
			"id":         "person_mock_owner",
			"object":     "v2.core.account_person",
			"account":    accountID,
			"created":    "2026-05-02T08:36:40.000Z",
			"updated":    "2026-05-02T08:36:40.000Z",
			"given_name": "Sam",
			"surname":    "Okonkwo",
			"email":      "sam.okonkwo@example.com",
			"address":    map[string]any{"country": "us", "state": "NY"},
			"id_numbers": []any{},
			"relationship": map[string]any{
				"owner":             true,
				"representative":    false,
				"percent_ownership": "0.2",
			},
			"metadata": map[string]any{},
		},
	}
}

func v2Events() []map[string]any {
	return []map[string]any{
		{
			"id":       "evt_test_mock_capability_restricted",
			"object":   "v2.core.event",
			"type":     "v2.core.account[configuration.merchant].capability_status_updated",
			"created":  "2026-07-30T09:14:02.000Z",
			"livemode": false,
			"context":  nil,
			"reason":   nil,
			"related_object": map[string]any{
				"id":   "acct_mock_v2_restricted",
				"type": "v2.core.account",
				"url":  "/v2/core/accounts/acct_mock_v2_restricted?include=configuration.merchant",
			},
		},
		{
			"id":       "evt_test_mock_requirements_updated",
			"object":   "v2.core.event",
			"type":     "v2.core.account[requirements].updated",
			"created":  "2026-07-30T09:13:58.000Z",
			"livemode": false,
			"context":  nil,
			"reason":   nil,
			"related_object": map[string]any{
				"id":   "acct_mock_v2_restricted",
				"type": "v2.core.account",
				"url":  "/v2/core/accounts/acct_mock_v2_restricted?include=requirements",
			},
		},
		{
			"id":       "evt_test_mock_identity_updated",
			"object":   "v2.core.event",
			"type":     "v2.core.account[identity].updated",
			"created":  "2026-07-14T11:20:04.000Z",
			"livemode": false,
			"context":  nil,
			"reason":   nil,
			"related_object": map[string]any{
				"id":   "acct_mock_v2_restricted",
				"type": "v2.core.account",
				"url":  "/v2/core/accounts/acct_mock_v2_restricted?include=identity",
			},
		},
		{
			"id":       "evt_test_mock_account_created",
			"object":   "v2.core.event",
			"type":     "v2.core.account.created",
			"created":  "2026-01-14T10:04:11.000Z",
			"livemode": false,
			"context":  nil,
			"reason":   nil,
			"related_object": map[string]any{
				"id":   "acct_mock_v2_active",
				"type": "v2.core.account",
				"url":  "/v2/core/accounts/acct_mock_v2_active",
			},
		},
	}
}
