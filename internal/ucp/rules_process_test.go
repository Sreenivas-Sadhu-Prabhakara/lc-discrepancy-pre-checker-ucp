package ucp

import "testing"

// processCredit is a credit with the places and the bank a presentation rule
// needs.
func processCredit() Credit {
	c := minimalCredit()
	c.Amount = 10000
	c.ExpiryDate = MustDate("2026-04-30")
	return c
}

func TestRulePlaceOfPresentation(t *testing.T) {
	th := StatutoryThresholds()

	t.Run("no place entered", func(t *testing.T) {
		r := Check(processCredit(), Presentation{Date: MustDate("2026-03-20")}, th)
		wantSkipped(t, r, "M1-PLACE")
	})

	t.Run("nothing in the credit to compare it with", func(t *testing.T) {
		credit := Credit{Currency: "USD", ExpiryDate: MustDate("2026-04-30")}
		r := Check(credit, Presentation{Date: MustDate("2026-03-20"), Place: "State Bank of India, Coimbatore"}, th)
		wantSkipped(t, r, "M1-PLACE")
	})

	t.Run("presenting to the nominated bank", func(t *testing.T) {
		credit := processCredit()
		credit.AvailableWith = "State Bank of India, Coimbatore"
		r := Check(credit, Presentation{Date: MustDate("2026-03-20"), Place: "State Bank of India, Coimbatore"}, th)
		wantFinding(t, r, "M1-PLACE", SeverityCleared, "art. 6(d)(ii)", "in addition to the place of the issuing bank")
	})

	t.Run("presenting to the issuing bank is always open to you", func(t *testing.T) {
		credit := processCredit()
		credit.AvailableWith = "State Bank of India, Coimbatore"
		credit.IssuingBank = "Banco de Manila"
		r := Check(credit, Presentation{Date: MustDate("2026-03-20"), Place: "Banco de Manila"}, th)
		wantFinding(t, r, "M1-PLACE", SeverityCleared)
	})

	t.Run("a credit available with any bank", func(t *testing.T) {
		credit := processCredit()
		credit.AvailableWith = "Any bank in India by negotiation"
		r := Check(credit, Presentation{Date: MustDate("2026-03-20"), Place: "Canara Bank, Erode"}, th)
		wantFinding(t, r, "M1-PLACE", SeverityCleared, "any bank")
	})

	t.Run("presenting somewhere the credit does not name", func(t *testing.T) {
		credit := processCredit()
		credit.AvailableWith = "State Bank of India, Coimbatore"
		credit.ExpiryPlace = "Coimbatore"
		credit.IssuingBank = "Banco de Manila"
		r := Check(credit, Presentation{Date: MustDate("2026-03-20"), Place: "Punjab National Bank, Ludhiana"}, th)
		wantFinding(t, r, "M1-PLACE", SeverityCaution, "arts. 6(a), 6(d)(ii)", "days it spends in transit")
	})
}

func TestRuleExamination(t *testing.T) {
	th := StatutoryThresholds()

	t.Run("no presentation date", func(t *testing.T) {
		r := Check(processCredit(), Presentation{}, th)
		wantSkipped(t, r, "M2-EXAMINATION")
	})

	t.Run("the fifth banking day, counted over a weekend", func(t *testing.T) {
		// Presented Monday 23 March 2026; the fifth banking day following is
		// Monday 30 March.
		r := Check(processCredit(), Presentation{Date: MustDate("2026-03-23")}, th)
		wantFinding(t, r, "M2-EXAMINATION", SeverityVerify,
			"30 March 2026", "arts. 14(b), 16(c), 16(d)", "art. 16(f)", "precluded")
	})

	t.Run("closed days are skipped and stated", func(t *testing.T) {
		credit := processCredit()
		credit.BankClosedDays = []Date{MustDate("2026-03-25")}
		r := Check(credit, Presentation{Date: MustDate("2026-03-23")}, th)
		wantFinding(t, r, "M2-EXAMINATION", SeverityVerify, "31 March 2026", "1 closed day", "Saturday, Sunday")
	})

	t.Run("a bank open on no day at all cannot be counted", func(t *testing.T) {
		never := StatutoryThresholds()
		never.WeekendDays = []int{0, 1, 2, 3, 4, 5, 6}
		r := Check(processCredit(), Presentation{Date: MustDate("2026-03-23")}, never)
		wantSkipped(t, r, "M2-EXAMINATION")
	})
}
