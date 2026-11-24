package ucp

import "strings"

// documentRules are the checks on the set itself: what has to be in it, how
// many of each, and the particular form articles 17 and 28 require.
func documentRules() []rule {
	return []rule{
		{
			ID:       "R1-MISSING",
			Group:    groupDocuments,
			Title:    "Every document the credit stipulates is in the set",
			Citation: "UCP 600 arts. 14(a), 15",
			Tests:    "That nothing the credit asks for has been left out — the simplest refusal of all and the one a checklist prevents.",
			eval:     ruleMissingDocuments,
		},
		{
			ID:       "R2-ORIGINALS",
			Group:    groupDocuments,
			Title:    "Originals and copies",
			Citation: "UCP 600 arts. 17(a), 17(d), 17(e)",
			Tests:    "That at least one original of each stipulated document is presented, and reads \"in duplicate\" and \"in three copies\" the way article 17(e) does — one original and the rest in copies.",
			eval:     ruleOriginals,
		},
		{
			ID:       "R3-EXTRA",
			Group:    groupDocuments,
			Title:    "Documents the credit does not ask for",
			Citation: "UCP 600 art. 14(g)",
			Tests:    "Flags a document in the set that the credit does not call for: it will be disregarded and may be returned, and anything on it that conflicts with the rest is a risk taken for nothing.",
			eval:     ruleExtraDocuments,
		},
		{
			ID:       "R4-NON-DOCUMENTARY",
			Group:    groupDocuments,
			Title:    "Conditions with no document behind them",
			Citation: "UCP 600 art. 14(h)",
			Tests:    "Lists the credit's conditions that name no document to show compliance: a bank deems such a condition not stated and disregards it.",
			eval:     ruleNonDocumentary,
		},
		{
			ID:       "R5-INSURANCE-FORM",
			Group:    groupDocuments,
			Title:    "Form of the insurance document",
			Citation: "UCP 600 arts. 28(a), 28(b), 28(c), 28(d), 28(h)",
			Tests:    "That the insurance is a policy or certificate and not a broker's cover note, that every original issued is presented, that it was issued by an insurer or underwriter, and how an \"all risks\" requirement is read.",
			eval:     ruleInsuranceForm,
		},
		{
			ID:       "R6-ORIGINALITY",
			Group:    groupDocuments,
			Title:    "What makes a sheet an original",
			Citation: "UCP 600 arts. 17(b), 17(c)",
			Tests:    "Sets out the test a bank applies to decide whether a document is an original, because no form can answer it — it is examined on the face of the paper.",
			eval:     ruleOriginality,
		},
	}
}

func ruleMissingDocuments(e *exam) []Finding {
	if len(e.C.RequiredDocuments) == 0 {
		return e.skip("the list of documents the credit requires")
	}

	var out []Finding
	var missing []string
	for _, r := range e.C.RequiredDocuments {
		if len(e.P.OfKind(r.Kind)) == 0 {
			name := r.Wording
			if name == "" {
				name = r.Kind.Title()
			}
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		out = append(out, e.find(SeverityDiscrepancy,
			pick(len(missing), "A document the credit requires is missing", "Documents the credit requires are missing"),
			"The credit calls for "+strings.Join(missing, "; ")+", and nothing of that kind is in the presentation. Banks examine the documents to determine whether they constitute a complying presentation, and a document that is not there cannot.",
			"UCP 600 arts. 14(a), 15",
			"Obtain the missing document before you present. Presenting an incomplete set to stop the clock does not work: the bank's five banking days run from the day of presentation either way, and you get the set back refused.",
			missing...))
		return out
	}
	return []Finding{e.find(SeverityCleared, "Every document the credit asks for is in the set",
		plural(len(e.C.RequiredDocuments), "document is required", "documents are required")+" and all of them are present.",
		"UCP 600 art. 14(a)", "")}
}

func ruleOriginals(e *exam) []Finding {
	if len(e.C.RequiredDocuments) == 0 {
		return e.skip("the list of documents the credit requires, with how many originals and copies of each")
	}

	var out []Finding
	tested := 0

	for _, r := range e.C.RequiredDocuments {
		docs := e.P.OfKind(r.Kind)
		if len(docs) == 0 {
			continue // the missing-document rule reports this
		}
		d := docs[0]
		tested++
		name := r.Wording
		if name == "" {
			name = r.Kind.Title()
		}

		if d.Originals == 0 && d.Copies == 0 {
			out = append(out, e.find(SeverityVerify, "How many originals of "+r.Kind.Title()+" were not entered",
				"At least one original of each document stipulated in the credit must be presented, and nothing was entered for "+d.Label()+".",
				"UCP 600 art. 17(a)", "Count what you have against what the credit asks for.", d.Label()))
			continue
		}

		if d.Originals < e.T.MinOriginalsPerDocument {
			out = append(out, e.find(SeverityDiscrepancy, "No original of "+r.Kind.Title()+" is being presented",
				"Only "+plural(d.Copies, "copy", "copies")+" of "+d.Label()+" "+isAre(d.Copies)+" in the set. At least "+
					plural(e.T.MinOriginalsPerDocument, "original", "originals")+" of each document stipulated in the credit must be presented.",
				"UCP 600 art. 17(a)",
				"Add the original. Where the credit asks for copies, article 17(d) lets you present either originals or copies — but where it asks for the document at all, one original is the floor.",
				d.Label()))
			continue
		}

		if r.Originals > 1 && d.Originals < r.Originals {
			// Article 17(e): a credit requiring a document "in duplicate", "in
			// two fold" or "in two copies" is satisfied by one original and the
			// rest in copies. It does not reach a credit that demands
			// originals — "full set 3/3 original bills of lading" wants three
			// originals and nothing less, which is why the relief below turns
			// on how the credit is worded and not on the number alone.
			if r.CountsInCopies && d.Originals+d.Copies >= r.Originals {
				out = append(out, e.find(SeverityCleared, "One original and the rest in copies is enough for "+r.Kind.Title(),
					"The credit asks for "+plural(r.Originals, "original", "originals")+" of "+name+" and you have "+
						plural(d.Originals, "original", "originals")+" and "+plural(d.Copies, "copy", "copies")+
						". Where a credit requires a document \"in duplicate\", \"in two fold\" or \"in two copies\", presenting at least one original and the remaining number in copies satisfies it, unless the document itself says otherwise.",
					"UCP 600 art. 17(e)",
					"Read the credit's exact wording. \"Invoice in triplicate\" is satisfied by one original and two copies; \"three original invoices\" is not.",
					d.Label()))
			} else {
				why := "and article 17(e) does not help, because the credit counts in originals rather than in copies"
				if r.CountsInCopies {
					why = "which is short even allowing copies to stand in for the second and later originals under article 17(e)"
				}
				out = append(out, e.find(SeverityDiscrepancy, "Too few of "+r.Kind.Title()+" presented",
					"The credit asks for "+plural(r.Originals, "original", "originals")+" of "+name+" plus "+
						plural(r.Copies, "copy", "copies")+", and the set has "+plural(d.Originals, "original", "originals")+
						" and "+plural(d.Copies, "copy", "copies")+" — "+why+".",
					"UCP 600 arts. 17(a), 17(e)",
					"Make up the number. Where the credit counts in \"copies\" — \"in duplicate\", \"in triplicate\" — one original and photocopies will do; where it says \"original\" or \"full set\", only originals will.",
					d.Label()))
			}
			continue
		}

		if r.Copies > 0 && d.Originals+d.Copies < r.Originals+r.Copies {
			out = append(out, e.find(SeverityCaution, "Fewer sheets of "+r.Kind.Title()+" than the credit asks for",
				"The credit asks for "+plural(r.Originals, "original", "originals")+" and "+plural(r.Copies, "copy", "copies")+
					" of "+name+"; the set has "+plural(d.Originals, "original", "originals")+" and "+plural(d.Copies, "copy", "copies")+".",
				"UCP 600 art. 17(a)",
				"Photocopy the shortfall. Where the credit asks for copies, originals or copies are both acceptable under article 17(d), so an extra original does no harm.",
				d.Label()))
		}
	}

	if tested == 0 {
		return e.skip("a document in the set matching one the credit requires")
	}
	if len(out) == 0 {
		out = append(out, e.find(SeverityCleared, "Enough originals and copies of everything",
			"Each of the "+plural(tested, "document", "documents")+" the credit stipulates is presented with at least one original and the number the credit calls for.",
			"UCP 600 art. 17(a)", ""))
	}
	return out
}

func ruleExtraDocuments(e *exam) []Finding {
	if len(e.C.RequiredDocuments) == 0 {
		return e.skip("the list of documents the credit requires")
	}
	required := map[Kind]bool{}
	for _, r := range e.C.RequiredDocuments {
		required[r.Kind] = true
	}
	var extra []string
	for _, d := range e.P.Documents {
		if !required[d.Kind] {
			extra = append(extra, d.Label())
		}
	}
	if len(extra) == 0 {
		return e.skip("a document in the set that the credit does not call for (nothing to report)")
	}
	return []Finding{e.find(SeverityCaution,
		pick(len(extra), "A document is in the set that the credit does not ask for", "Documents are in the set that the credit does not ask for"),
		strings.Join(extra, ", ")+" "+isAre(len(extra))+" in the presentation, and nothing entered shows the credit calling for "+itThem(len(extra))+
			". A document presented but not required by the credit will be disregarded and may be returned to the presenter.",
		"UCP 600 art. 14(g)",
		"Take "+itThem(len(extra))+" out. It cannot help you — a bank disregards it — and while it is in the set, anything on it that contradicts the rest of the documents is still data a human examiner has read.",
		extra...)}
}

func ruleNonDocumentary(e *exam) []Finding {
	if len(e.C.NonDocumentaryConditions) == 0 {
		return e.skip("a condition in the credit with no document named to show it has been met")
	}
	return []Finding{e.find(SeverityCleared,
		pick(len(e.C.NonDocumentaryConditions), "A condition of the credit will be disregarded", "Conditions of the credit will be disregarded"),
		"The credit states: \""+strings.Join(e.C.NonDocumentaryConditions, "\"; \"")+
			"\" — without naming a document to show compliance. If a credit contains a condition without stipulating the document to indicate compliance with it, banks will deem such condition as not stated and will disregard it.",
		"UCP 600 art. 14(h)",
		"You still have to perform your contract with the buyer — article 4 keeps the credit and the contract separate, and the buyer can sue on the contract. But do not manufacture a certificate to evidence a condition the bank will not look at: a document nobody asked for is a document that can conflict (art. 14(g)).",
		e.C.NonDocumentaryConditions...)}
}

func ruleInsuranceForm(e *exam) []Finding {
	ins, ok := e.P.First(KindInsurance)
	if !ok || ins.Insurance == nil {
		return e.skip("an insurance document in the presentation")
	}
	in := ins.Insurance
	var out []Finding

	if in.IsCoverNote {
		out = append(out, e.find(SeverityDiscrepancy, "A cover note is not acceptable",
			ins.Label()+" is a broker's cover note. Cover notes will not be accepted — the sub-article says so without qualification, and no amount of wording on the note changes it.",
			"UCP 600 art. 28(c)",
			"Get the policy or the certificate itself. An insurance policy is acceptable in place of a certificate or a declaration under an open cover (art. 28(d)), so the policy is always safe; the cover note never is.",
			ins.Label()))
	}

	if in.OriginalsIssued > 0 && ins.Originals > 0 && ins.Originals < in.OriginalsIssued {
		out = append(out, e.find(SeverityDiscrepancy, "Not every original of the insurance document is being presented",
			ins.Label()+" says it was issued in "+plural(in.OriginalsIssued, "original", "originals")+" and only "+
				plural(ins.Originals, "is", "are")+" in the set. Where the insurance document indicates that it has been issued in more than one original, all originals must be presented.",
			"UCP 600 art. 28(b)",
			"Find the other original. This is a favourite of document checkers because the count is printed on the face of the certificate — \"issued in duplicate\" means both halves travel together.",
			ins.Label()))
	}

	if in.IssuedBy != "" {
		independent := true
		if e.C.Beneficiary.Name != "" {
			switch Compare(e.C.Beneficiary.Name, in.IssuedBy, e.T) {
			case MatchExact, MatchEquivalent, MatchTypo:
				independent = false
			}
		}
		if !independent {
			out = append(out, e.find(SeverityDiscrepancy, "The insurance document appears to be issued by you",
				ins.Label()+" shows "+in.IssuedBy+" as its issuer, which is the beneficiary. An insurance document must appear to be issued and signed by an insurance company, an underwriter or their agents or their proxies.",
				"UCP 600 art. 28(a)",
				"Present the insurer's own document. Your own declaration under an open policy is not one, unless it is issued on the insurer's form and signed for the insurer.",
				ins.Label()))
		}
	} else {
		out = append(out, e.find(SeverityVerify, "Who issued the insurance document was not entered",
			"An insurance document must appear to be issued and signed by an insurance company, an underwriter or their agents or their proxies.",
			"UCP 600 art. 28(a)",
			"Check the document carries the insurer's name and a signature for the insurer, and that an agent or proxy signing it says for whom.",
			ins.Label()))
	}

	if ContainsAny(e.C.InsuranceRisks, "all risks") {
		out = append(out, e.find(SeverityCleared, "An \"all risks\" requirement is read generously",
			"The credit asks for \""+Shorten(e.C.InsuranceRisks, 80)+"\". Where a credit requires insurance against \"all risks\" and the document presented carries any \"all risks\" notation or clause — whether or not it is headed \"all risks\" — the document is accepted without regard to any risks stated to be excluded.",
			"UCP 600 art. 28(h)",
			"Do not argue with the insurer about the exclusions printed on the back. What matters is that an \"all risks\" clause or notation appears at all. The same generosity applies to \"usual risks\" or \"customary risks\" (art. 28(g)).",
			ins.Label()))
	}

	if e.insuranceEndorsementWanted() && !in.Endorsed {
		out = append(out, e.find(SeverityCaution, "The credit appears to want the insurance endorsed",
			"The credit's wording for the insurance document mentions an endorsement or that it be issued to order, and nothing entered shows "+ins.Label()+
				" endorsed. UCP 600 has no article requiring an endorsement — the requirement comes from the credit's own wording, and the way banks read it is standard banking practice rather than a rule in the text.",
			"UCP 600 art. 14(a), on the credit's own term",
			"Have the insurer or the assured endorse it exactly as the credit words it — in blank where it says \"blank endorsed\". Ask your bank how they read the wording before the document is issued, not after.",
			ins.Label()))
	}

	if len(out) == 0 {
		out = append(out, e.find(SeverityCleared, "The insurance document is in an acceptable form",
			ins.Label()+" is a policy or certificate rather than a cover note, and nothing entered puts its form in doubt.",
			"UCP 600 arts. 28(a), 28(c)", "", ins.Label()))
	}
	return out
}

func ruleOriginality(e *exam) []Finding {
	if len(e.P.Documents) == 0 {
		return e.skip("at least one document in the presentation")
	}
	return []Finding{e.find(SeverityVerify, "Whether each sheet counts as an original is read off the paper",
		"This tool took your word for how many originals you hold, because originality is examined on the face of the document and no form can show it. A bank treats as an original any document bearing an apparently original signature, mark, stamp or label of the issuer, unless the document itself says it is not one (art. 17(b)). A document also counts as an original if it appears to be written, typed, perforated or stamped by the issuer's hand, or to be on the issuer's original stationery, or states that it is original (art. 17(c)).",
		"UCP 600 arts. 17(b), 17(c)",
		"Look at every sheet before it goes in. A photocopy of a signature is not an apparently original signature; a document marked \"copy\" is not an original whatever else it bears; and a laser-printed certificate on plain paper with no stamp is the one most often sent back.")}
}

// insuranceEndorsementWanted reads the credit's own wording for the insurance
// document to see whether it asks for an endorsement.
func (e *exam) insuranceEndorsementWanted() bool {
	for _, r := range e.C.RequiredDocuments {
		if r.Kind != KindInsurance {
			continue
		}
		if ContainsAny(r.Wording, "endorsed", "endorsement", "blank endorsed", "to order") {
			return true
		}
	}
	return false
}

func itThem(n int) string {
	if n == 1 {
		return "it"
	}
	return "them"
}
