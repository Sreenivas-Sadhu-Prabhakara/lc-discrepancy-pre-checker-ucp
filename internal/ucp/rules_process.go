package ucp

// processRules are the checks on the presentation itself rather than on the
// documents: where it goes and what happens after it is made. Neither of them
// is a discrepancy, and both change what an exporter does next.
func processRules() []rule {
	return []rule{
		{
			ID:       "M1-PLACE",
			Group:    groupProcess,
			Title:    "Where the presentation is made",
			Citation: "UCP 600 arts. 6(a), 6(d)(ii)",
			Tests:    "That the documents go to a place for presentation the credit actually has — the place of the bank with which the credit is available, with the issuing bank's own counters always available in addition.",
			eval:     rulePlace,
		},
		{
			ID:       "M2-EXAMINATION",
			Group:    groupProcess,
			Title:    "When you should hear back",
			Citation: "UCP 600 arts. 14(b), 16(c), 16(d), 16(f)",
			Tests:    "Works out the day by which a refusal has to reach you, and what the notice must contain — a bank that misses it is precluded from claiming the documents do not comply.",
			eval:     ruleExamination,
		},
	}
}

func rulePlace(e *exam) []Finding {
	if e.P.Place == "" {
		return e.skip("where you will present the documents")
	}
	if e.C.AvailableWith == "" && e.C.ExpiryPlace == "" && e.C.IssuingBank == "" {
		return e.skip("the bank the credit is available with, or the place of expiry")
	}

	// The place of the bank with which the credit is available is the place for
	// presentation. A credit available with any bank may be presented at any
	// bank. The issuing bank's place is always a place for presentation in
	// addition to any other the credit names.
	anyBank := ContainsAny(e.C.AvailableWith, "any bank")
	candidates := []struct{ what, where string }{
		{"the bank the credit is available with", e.C.AvailableWith},
		{"the place of expiry stated in the credit", e.C.ExpiryPlace},
		{"the issuing bank", e.C.IssuingBank},
	}

	for _, c := range candidates {
		if c.where == "" {
			continue
		}
		switch Compare(c.where, e.P.Place, e.T) {
		case MatchExact, MatchEquivalent, MatchGeneral, MatchTypo:
			return []Finding{e.find(SeverityCleared, "Presenting at a place the credit allows",
				"You are presenting to \""+e.P.Place+"\", which matches "+c.what+
					". A place for presentation other than that of the issuing bank is in addition to the place of the issuing bank, so the issuing bank's counters are always open to you as well.",
				"UCP 600 art. 6(d)(ii)", "")}
		}
	}

	if anyBank {
		return []Finding{e.find(SeverityCleared, "The credit is available with any bank",
			"The credit is available with any bank, so the place for presentation is that of any bank — including \""+e.P.Place+"\".",
			"UCP 600 art. 6(d)(ii)",
			"Presenting through your own bank is usually quicker and gives you someone to argue with on your side of the transaction.")}
	}

	return []Finding{e.find(SeverityCaution, "Presenting somewhere the credit may not name",
		"You are presenting to \""+e.P.Place+"\". The credit is available with \""+orDash(e.C.AvailableWith)+
			"\" and expires at \""+orDash(e.C.ExpiryPlace)+"\". The place of the bank with which the credit is available is the place for presentation; a nominated bank's counters are not open to a presentation under a credit that is not available with it.",
		"UCP 600 arts. 6(a), 6(d)(ii)",
		"Present to the nominated bank the credit names, or direct to the issuing bank — whose place is a place for presentation in addition to any other. Where your own bank only forwards the set, the days it spends in transit are yours, not the bank's.")}
}

func ruleExamination(e *exam) []Finding {
	if e.P.Date.Zero() {
		return e.skip("the date you will present the documents")
	}
	by, ok := e.T.AddBankingDays(e.P.Date, e.T.ExaminationBankingDays, e.C.BankClosedDays)
	if !ok {
		return e.skip("a workable list of the bank's closed days")
	}

	closed := ""
	if len(e.C.BankClosedDays) > 0 {
		closed = " The count skips the " + plural(len(e.C.BankClosedDays), "closed day", "closed days") + " you listed, and the weekend (" + weekdayNames(e.T.WeekendDays) + ")."
	}

	return []Finding{e.find(SeverityVerify, "A refusal must reach you by "+by.Human(),
		"Presenting on "+e.P.Date.Human()+", each bank has a maximum of "+plural(e.T.ExaminationBankingDays, "banking day", "banking days")+
			" following the day of presentation to determine whether the presentation complies, and notice of refusal must be given no later than the close of the fifth banking day following the day of presentation."+closed+
			" A notice of refusal must say that the bank is refusing to honour or negotiate, state each discrepancy on which it refuses, and say what it is doing with the documents.",
		"UCP 600 arts. 14(b), 16(c), 16(d)",
		"Diarise "+by.Human()+". If a refusal arrives after it, or arrives without listing each discrepancy, say so in writing at once: an issuing or confirming bank that fails to act in accordance with article 16 is precluded from claiming that the documents do not constitute a complying presentation (art. 16(f)). That preclusion is one of the few real protections the rules give a beneficiary, and it is lost by saying nothing.")}
}
