# Letter of credit discrepancy pre-checker (UCP 600)

Check your draft export documents against a letter of credit **before** you present
them to a bank. Enter the credit's key terms and describe the documents you are
about to hand in; the tool runs 38 checks drawn from UCP 600 and lists what to
fix — while fixing it is still free.

It is a single binary with no dependencies. It serves a small web interface on
the machine it runs on, keeps everything in one JSON file beside itself, and
makes no outbound network calls of any kind.

**Explainer site:** <https://sreenivas-sadhu-prabhakara.github.io/lc-discrepancy-pre-checker-ucp/>

## What it does

- **Deadlines** — presentation within the expiry date, shipment within the
  latest shipment date, and the documents in within the 21 calendar days of
  article 14(c). It works out the date of shipment the way the transport
  articles define it, which is not always the date printed on the document.
- **Amounts and quantities** — the drawing against the credit amount and the
  tolerance that actually applies; the quantity against the tolerance in
  article 30(b) and its two conditions; the sum insured against the percentage
  required.
- **Descriptions and parties** — whether the invoice describes the goods as the
  credit does, whether it is issued by you and made out to the applicant, and
  whether a difference in wording is a typing error or a change of data.
- **The documents themselves** — anything the credit asks for that is missing,
  whether you hold an original of each, and whether the insurance is a policy
  rather than a broker's cover note.
- **The transport document** — clean, on board, at the credit's ports, in a
  full set, and not a charter party bill when the credit never asked for one.
- **Cross-document consistency** — the invoice number, package count, gross
  weight and shipping marks that your documents repeat from one another, which
  is the test in article 14(d) and where most refusals actually come from.

Fifteen of the 38 checks exist to **clear** something that looks wrong and is
not: a packing list describing the goods in general terms (article 14(e)), a
different branch address on the invoice (article 14(j)), a shipper who is not
you (article 14(k)), a transhipment on a container shipment under a credit that
forbids transhipment (article 20(c)(ii)). Each of those gets documents
reprinted for nothing every week.

## Who it's for

Small and medium exporters in India and the Philippines who ship under
documentary credits and have no in-house trade-finance desk — garment and
textile units, machining shops, food processors, furniture and handicraft
exporters. You need no accounting system and no internet connection.

## The rule it implements

**UCP 600** — the Uniform Customs and Practice for Documentary Credits, 2007
revision, International Chamber of Commerce publication 600. It applies to a
credit because the credit says it does (article 1).

Every finding names the sub-article it rests on. The principal ones are
article 3 (interpretations), articles 6 and 29 (expiry, place of presentation
and the extension when the bank is closed), article 14 (the standard for
examination), articles 16 and 17 (refusal, originals), article 18 (the
commercial invoice), articles 19 to 27 (transport documents), article 28
(insurance) and articles 30 to 32 (tolerances, partial shipments, instalments).

Every figure those articles fix is a **named constant in one place** —
`internal/ucp/thresholds.go` — cited with its sub-article, listed in the
interface, printed in every report, and editable, because several of them a
credit may vary and the rules themselves are revised from time to time.

| Figure | In force | Source |
| --- | --- | --- |
| Presentation period after shipment | 21 calendar days | art. 14(c) |
| Examination period | 5 banking days | arts. 14(b), 16(d) |
| "On or about" | 5 days either side, inclusive | art. 3 |
| "About" / "approximately" | 10% either way | art. 30(a) |
| Quantity tolerance | 5% either way, conditional | art. 30(b) |
| Short-drawing tolerance | 5% **less only**, conditional | art. 30(c) |
| Minimum insurance | 110% of the CIF or CIP value | art. 28(f)(ii) |
| Originals of each stipulated document | at least 1 | art. 17(a) |
| Typing-error allowance | 2 character edits | **not UCP 600** — a convention of this tool |
| Decimal places on amounts | 2 | **not UCP 600** — a rounding convention |
| Weekend | Saturday, Sunday | **not UCP 600** — a local convention |

The last three are conventions of this tool and are labelled as such wherever
they appear. The typing-error allowance stands in for the ICC's standard
banking practice that a misspelling which does not affect meaning is no
discrepancy — stated in ISBP 745, the ICC's International Standard Banking
Practice companion to UCP 600, in its general principles on misspellings and
typing errors. No paragraph number of ISBP 745 is cited, because this build has
not verified one and an invented citation would be worse than none.

Confirm every figure and article against the current text of UCP 600, or with
your bank, before you rely on it.

## Install and run

```sh
git clone https://github.com/sreenivas-sadhu-prabhakara/lc-discrepancy-pre-checker-ucp.git
cd lc-discrepancy-pre-checker-ucp
go build -o lc-discrepancy-pre-checker-ucp .
./lc-discrepancy-pre-checker-ucp          # then open http://127.0.0.1:8080
```

Or run it straight from the source with `go run .`. Flags:

```
-addr   address to serve the interface on   (default 127.0.0.1:8080)
-data   file to keep your saved cases in    (default ./lc-discrepancy-pre-checker-ucp-data.json)
```

It binds to `127.0.0.1` — the machine it is running on, not the network. Stop
it with Ctrl-C. There is a worked example built in: press **Load the worked
example** to see a report before typing anything of your own.

## Where your data is stored

In the one file `-data` names, on your own machine. It holds your credits, your
documents and the figures in force, as plain indented JSON you can read. Every
save writes a temporary file beside it and renames it over the old one, so an
interrupted save leaves the previous file intact rather than half of one, and
the file is written readable only by you (mode 600).

Nothing leaves the machine. There are no outbound network calls, no telemetry,
no analytics, no external fonts and no bank or portal integration.

## Scope and accuracy

**This is a pre-submission self-check, not a bank's examination.** A bank
examines the documents themselves, on their face, and determines whether the
presentation constitutes a complying presentation (articles 14(a) and 15).
**Only the issuing bank's determination decides whether you are paid.** This
tool reads a description of your documents that you typed in, so a clean result
is not a complying presentation — and the tool never says it is. Its best
verdict is "nothing found that would stop the presentation — the issuing bank
still decides".

It cannot see a signature, a stamp or whether a sheet is an original: that is
decided on the face of the paper under articles 17(b) and 17(c), and the tool
says so instead of pretending otherwise. It does not read or scan your
documents — every field is entered by hand — and a field you leave blank is
reported as a check that could not be run, never as a check that passed.

Not implemented: transferable credits (article 38), instalment drawings and
shipments (article 32), bank-to-bank reimbursement (article 13), assignment of
proceeds (article 39), standby letters of credit, demand guarantees, and
credits issued under eUCP for electronic records. No national law is applied;
UCP 600 applies because the credit says it does, and the law of the place of
presentation sits above it.

It is a checklist with citations, not advice. Confirm every article and figure
against the current text of UCP 600, and take a banker's or a lawyer's view on
anything that carries real money.

MIT licensed.
