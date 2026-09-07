package fingerprint

import (
	"testing"

	"draftline/internal/types"
)

func ledgerFor(ledgers []types.StateLedger, entity, aspect string) *types.StateLedger {
	for index := range ledgers {
		if ledgers[index].EntityName == entity && ledgers[index].Aspect == aspect {
			return &ledgers[index]
		}
	}
	return nil
}

func TestTransferProducesBothSidesOfThePossessionLedger(t *testing.T) {
	book := frameBook([]types.EvidenceRecord{
		evidence("e1", 0, "Avery picked up the brass key from the desk.", "event", "interaction", "picked"),
		evidence("e2", 1, "Avery handed the brass key to Mira.", "event", "interaction", "handed"),
	})
	model := Build(&book, nil)
	avery := ledgerFor(model.Ledgers, "Avery Cole", "possession")
	mira := ledgerFor(model.Ledgers, "Mira Voss", "possession")
	if avery == nil || mira == nil {
		t.Fatalf("expected possession ledgers for both sides: %#v", model.Ledgers)
	}
	if len(avery.Entries) != 2 || avery.Entries[0].Operation != "set" || avery.Entries[1].Operation != "transfer_out" {
		t.Fatalf("avery possession history: %#v", avery.Entries)
	}
	if len(mira.Entries) != 1 || mira.Entries[0].Operation != "transfer_in" {
		t.Fatalf("mira possession history: %#v", mira.Entries)
	}
	if avery.Qualifier != mira.Qualifier || avery.Qualifier == "" {
		t.Fatalf("both sides must share the item qualifier, got %q vs %q", avery.Qualifier, mira.Qualifier)
	}
}

func TestConflictingLifeStatusAccountsArePreservedNotAveraged(t *testing.T) {
	book := frameBook([]types.EvidenceRecord{
		evidence("e1", 0, "Mira Voss was killed in the warehouse fire.", "event", "state", "killed"),
		evidence("e2", 3, "Mira was alive and waiting at the dock.", "fact", "state", "was"),
	})
	model := Build(&book, nil)
	ledger := ledgerFor(model.Ledgers, "Mira Voss", "life_status")
	if ledger == nil || len(ledger.Entries) != 2 {
		t.Fatalf("expected two preserved life-status accounts: %#v", ledger)
	}
	if ledger.Entries[0].Value != "dead" || ledger.Entries[1].Value != "alive" {
		t.Fatalf("expected dead then alive, got %#v", ledger.Entries)
	}
}

func TestLocationHistoryOrdersByNarrativeOrder(t *testing.T) {
	book := frameBook([]types.EvidenceRecord{
		evidence("e1", 0, "Avery was at Northgate Station before sunrise.", "fact", "state", "was"),
		evidence("e2", 2, "Avery entered the customs office at noon.", "event", "transition", "entered"),
		evidence("e3", 4, "Avery left the customs office in a hurry.", "event", "transition", "left"),
	})
	book.Analysis.Evidence.Records[0].NamedEntities = []types.EvidenceTerm{{Text: "Northgate Station", Label: "FAC"}}
	model := Build(&book, nil)
	ledger := ledgerFor(model.Ledgers, "Avery Cole", "location")
	if ledger == nil || len(ledger.Entries) != 3 {
		t.Fatalf("location ledger: %#v", ledger)
	}
	if ledger.Entries[0].Value != "Northgate Station" || ledger.Entries[2].Operation != "clear" {
		t.Fatalf("location history: %#v", ledger.Entries)
	}
}

func TestNegatedKnowledgeClearsTheLedger(t *testing.T) {
	book := frameBook([]types.EvidenceRecord{
		evidence("e1", 0, "Avery never learned that the safe had a second combination.", "fact", "state", "learned"),
	})
	model := Build(&book, nil)
	ledger := ledgerFor(model.Ledgers, "Avery Cole", "knowledge")
	if ledger == nil || len(ledger.Entries) != 1 || ledger.Entries[0].Operation != "clear" {
		t.Fatalf("negated knowledge must clear, got %#v", ledger)
	}
}
