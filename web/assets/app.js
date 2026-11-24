/* Letter of credit discrepancy pre-checker — the page's own behaviour.

   Every figure on this page comes from the tool running on this machine; no
   other host is contacted. The page builds its tables and its findings as text
   nodes and never as markup, so a goods description copied off a credit can
   never be read as anything but text. */
(function () {
  "use strict";

  var MONTHS = ["January", "February", "March", "April", "May", "June",
    "July", "August", "September", "October", "November", "December"];

  var state = null;     // what GET /api/state last said
  var current = null;   // the case being edited, as last returned by the server
  var lastResult = null;

  /* ---------- small helpers ---------- */

  function $(id) { return document.getElementById(id); }

  function clear(node) {
    while (node && node.firstChild) { node.removeChild(node.firstChild); }
  }

  function show(node, on) {
    if (!node) { return; }
    if (on) { node.removeAttribute("hidden"); } else { node.setAttribute("hidden", ""); }
  }

  function el(tag, className, text) {
    var node = document.createElement(tag);
    if (className) { node.className = className; }
    if (text !== undefined && text !== null && text !== "") {
      node.appendChild(document.createTextNode(String(text)));
    }
    return node;
  }

  function dateText(iso) {
    if (!iso) { return "—"; }
    var parts = String(iso).split("-");
    if (parts.length !== 3) { return String(iso); }
    var month = MONTHS[Number(parts[1]) - 1] || parts[1];
    return Number(parts[2]) + " " + month + " " + parts[0];
  }

  function banner(text, kind) {
    var node = $("banner");
    clear(node);
    if (!text) { show(node, false); return; }
    node.appendChild(document.createTextNode(text));
    node.className = kind === "bad" ? "note bad" : "note";
    show(node, true);
  }

  function num(value) {
    var n = Number(value);
    return isFinite(n) ? n : 0;
  }

  function lines(text) {
    return String(text || "").split("\n").map(function (s) { return s.trim(); })
      .filter(function (s) { return s !== ""; });
  }

  /* ---------- talking to the tool ---------- */

  function call(method, path, body) {
    var options = { method: method, headers: {} };
    if (body !== undefined) {
      options.headers["Content-Type"] = "application/json";
      options.body = JSON.stringify(body);
    }
    return fetch(path, options).then(function (response) {
      var type = response.headers.get("Content-Type") || "";
      if (type.indexOf("application/json") < 0) {
        return response.text().then(function (text) {
          if (!response.ok) { throw new Error(text || ("the tool answered " + response.status)); }
          return text;
        });
      }
      return response.json().then(function (data) {
        if (!response.ok) { throw new Error((data && data.problem) || ("the tool answered " + response.status)); }
        return data;
      });
    });
  }

  function failed(err) {
    banner((err && err.message) || "something went wrong talking to the tool", "bad");
  }

  /* ---------- reading and writing the form ---------- */

  /* Each entry is [element id, path into the case, kind of value]. Keeping the
     mapping in one table means the form and the JSON cannot drift apart. */
  var FIELDS = [
    ["c-reference", "credit.reference", "text"],
    ["c-issuing-bank", "credit.issuingBank", "text"],
    ["c-available-with", "credit.availableWith", "text"],
    ["c-availability", "credit.availability", "text"],
    ["c-tenor-days", "credit.draftTenorDays", "int"],
    ["c-tenor-basis", "credit.draftTenorBasis", "text"],

    ["c-currency", "credit.currency", "text"],
    ["c-amount", "credit.amount", "float"],
    ["c-amount-qualifier", "credit.amountQualifier", "text"],
    ["c-tol-plus", "credit.tolerancePlusPercent", "float"],
    ["c-tol-minus", "credit.toleranceMinusPercent", "float"],

    ["c-applicant-name", "credit.applicant.name", "text"],
    ["c-applicant-address", "credit.applicant.address", "text"],
    ["c-applicant-country", "credit.applicant.country", "text"],
    ["c-beneficiary-name", "credit.beneficiary.name", "text"],
    ["c-beneficiary-address", "credit.beneficiary.address", "text"],
    ["c-beneficiary-country", "credit.beneficiary.country", "text"],

    ["c-issue-date", "credit.issueDate", "text"],
    ["c-expiry-date", "credit.expiryDate", "text"],
    ["c-expiry-place", "credit.expiryPlace", "text"],
    ["c-latest-shipment", "credit.latestShipmentDate", "text"],
    ["c-on-or-about", "credit.shipmentOnOrAbout", "text"],
    ["c-presentation-days", "credit.presentationPeriodDays", "int"],

    ["c-goods", "credit.goodsDescription", "text"],
    ["c-quantity", "credit.quantity", "float"],
    ["c-quantity-unit", "credit.quantityUnit", "text"],
    ["c-quantity-qualifier", "credit.quantityQualifier", "text"],
    ["c-packing-units", "credit.quantityInPackingUnits", "bool"],
    ["c-qty-tol-plus", "credit.quantityTolerancePlusPercent", "float"],
    ["c-qty-tol-minus", "credit.quantityToleranceMinusPercent", "float"],
    ["c-unit-price", "credit.unitPrice", "float"],

    ["c-trade-term", "credit.tradeTerm", "text"],
    ["c-insurance-percent", "credit.insurancePercent", "float"],
    ["c-insurance-risks", "credit.insuranceRisks", "text"],
    ["c-place-receipt", "credit.placeOfReceipt", "text"],
    ["c-port-loading", "credit.portOfLoading", "text"],
    ["c-port-discharge", "credit.portOfDischarge", "text"],
    ["c-final-destination", "credit.finalDestination", "text"],
    ["c-partial", "credit.partialShipments", "text"],
    ["c-transhipment", "credit.transhipment", "text"],

    ["p-date", "presentation.date", "text"],
    ["p-place", "presentation.place", "text"],
    ["p-drawn-currency", "presentation.drawnCurrency", "text"],
    ["p-drawn-amount", "presentation.drawnAmount", "float"],

    ["case-name", "name", "text"]
  ];

  function setPath(object, path, value) {
    var parts = path.split(".");
    var node = object;
    for (var i = 0; i < parts.length - 1; i++) {
      if (!node[parts[i]] || typeof node[parts[i]] !== "object") { node[parts[i]] = {}; }
      node = node[parts[i]];
    }
    node[parts[parts.length - 1]] = value;
  }

  function getPath(object, path) {
    var parts = path.split(".");
    var node = object;
    for (var i = 0; i < parts.length; i++) {
      if (node === null || node === undefined) { return undefined; }
      node = node[parts[i]];
    }
    return node;
  }

  function readValue(input, kind) {
    switch (kind) {
      case "bool": return !!input.checked;
      case "int": return input.value === "" ? 0 : Math.round(num(input.value));
      case "float": return input.value === "" ? 0 : num(input.value);
      default: return input.value.trim();
    }
  }

  function writeValue(input, kind, value) {
    if (kind === "bool") { input.checked = !!value; return; }
    if (kind === "int" || kind === "float") {
      input.value = (value === 0 || value === undefined || value === null) ? "" : String(value);
      return;
    }
    input.value = (value === undefined || value === null) ? "" : String(value);
  }

  /* readCase turns the whole form into the object the tool examines. */
  function readCase() {
    var c = { id: current && current.id ? current.id : "", credit: {}, presentation: {} };

    FIELDS.forEach(function (entry) {
      var input = $(entry[0]);
      if (input) { setPath(c, entry[1], readValue(input, entry[2])); }
    });

    c.credit.bankClosedDays = lines($("c-closed-days").value);
    c.credit.nonDocumentaryConditions = lines($("c-non-documentary").value);

    c.credit.requiredDocuments = [];
    Array.prototype.forEach.call($("requirements").children, function (row) {
      var kind = row.querySelector('[data-field="kind"]').value;
      var wording = row.querySelector('[data-field="wording"]').value.trim();
      c.credit.requiredDocuments.push({
        kind: kind,
        wording: wording,
        originals: Math.round(num(row.querySelector('[data-field="originals"]').value)),
        copies: Math.round(num(row.querySelector('[data-field="copies"]').value)),
        mustBeSigned: false,
        requireOnBoard: false,
        requireFullSet: false
      });
    });

    c.presentation.documents = [];
    Array.prototype.forEach.call($("documents").children, function (row) {
      c.presentation.documents.push(readDocument(row));
    });

    return c;
  }

  /* The fields every document row carries, and the ones each kind adds. */
  var DOC_FIELDS = [
    ["title", "text"], ["issuer", "text"], ["date", "text"],
    ["originals", "int"], ["copies", "int"], ["signed", "bool"], ["issuedByBeneficiary", "bool"],
    ["goodsDescription", "text"], ["invoiceNumberShown", "text"], ["lcReferenceShown", "text"],
    ["quantity", "float"], ["quantityUnit", "text"], ["packages", "float"],
    ["grossWeightKg", "float"], ["shippingMarks", "text"]
  ];

  var KIND_FIELDS = {
    "commercial-invoice": ["invoice.currency", "invoice.amount", "invoice.unitPrice", "invoice.tradeTerm",
      "invoice.madeOutTo.name", "invoice.madeOutTo.address", "invoice.madeOutTo.country"],
    "transport": ["transport.mode", "transport.onBoardDate", "transport.preprintedOnBoard",
      "transport.carrier", "transport.signedBy", "transport.vessel", "transport.voyage",
      "transport.portOfLoading", "transport.portOfDischarge", "transport.placeOfReceipt",
      "transport.finalDestination", "transport.shipper", "transport.consignee", "transport.notifyParty",
      "transport.originalsIssued", "transport.transhipment", "transport.containerised",
      "transport.onDeck", "transport.mayBeOnDeck", "transport.subjectToCharterParty",
      "transport.adverseClause"],
    "insurance": ["insurance.currency", "insurance.amount", "insurance.issuedBy",
      "insurance.originalsIssued", "insurance.coverEffectiveFrom", "insurance.risks",
      "insurance.isCoverNote", "insurance.endorsed"],
    "draft": ["draft.currency", "draft.amount", "draft.tenorDays", "draft.drawnOn"]
  };

  /* kindOf returns the kind of value a document field holds, from the input
     itself, so the two lists above need not repeat it. */
  function kindOfInput(input) {
    if (input.type === "checkbox") { return "bool"; }
    if (input.type === "number") { return input.step && input.step !== "1" ? "float" : "int"; }
    return "text";
  }

  function readDocument(row) {
    var kind = row.querySelector('[data-field="kind"]').value;
    var doc = { kind: kind };

    DOC_FIELDS.forEach(function (entry) {
      var input = row.querySelector('[data-field="' + entry[0] + '"]');
      if (input) { doc[entry[0]] = readValue(input, entry[1]); }
    });

    (KIND_FIELDS[kind] || []).forEach(function (path) {
      var input = row.querySelector('[data-field="' + path + '"]');
      if (input) { setPath(doc, path, readValue(input, kindOfInput(input))); }
    });

    return doc;
  }

  function writeCase(c) {
    current = c;
    FIELDS.forEach(function (entry) {
      var input = $(entry[0]);
      if (input) { writeValue(input, entry[2], getPath(c, entry[1])); }
    });

    $("c-closed-days").value = (c.credit.bankClosedDays || []).join("\n");
    $("c-non-documentary").value = (c.credit.nonDocumentaryConditions || []).join("\n");

    clear($("requirements"));
    (c.credit.requiredDocuments || []).forEach(addRequirement);
    clear($("documents"));
    (c.presentation.documents || []).forEach(addDocument);
  }

  /* ---------- the repeating rows ---------- */

  function addRequirement(req) {
    var row = $("requirement-template").content.firstElementChild.cloneNode(true);
    if (req) {
      row.querySelector('[data-field="kind"]').value = req.kind || "other";
      row.querySelector('[data-field="wording"]').value = req.wording || "";
      row.querySelector('[data-field="originals"]').value = String(req.originals || 0);
      row.querySelector('[data-field="copies"]').value = String(req.copies || 0);
    }
    row.querySelector("[data-remove]").addEventListener("click", function () {
      row.parentNode.removeChild(row);
    });
    $("requirements").appendChild(row);
    return row;
  }

  function addDocument(doc) {
    var row = $("document-template").content.firstElementChild.cloneNode(true);
    var kindSelect = row.querySelector('[data-field="kind"]');

    function applyKind() {
      row.setAttribute("data-kind", kindSelect.value);
    }
    kindSelect.addEventListener("change", applyKind);

    if (doc) {
      kindSelect.value = doc.kind || "other";
      DOC_FIELDS.forEach(function (entry) {
        var input = row.querySelector('[data-field="' + entry[0] + '"]');
        if (input) { writeValue(input, entry[1], doc[entry[0]]); }
      });
      (KIND_FIELDS[doc.kind] || []).forEach(function (path) {
        var input = row.querySelector('[data-field="' + path + '"]');
        if (input) { writeValue(input, kindOfInput(input), getPath(doc, path)); }
      });
    }
    applyKind();

    row.querySelector("[data-remove]").addEventListener("click", function () {
      row.parentNode.removeChild(row);
    });
    $("documents").appendChild(row);
    return row;
  }

  /* ---------- showing the result ---------- */

  var SEVERITY_CLASS = {
    "discrepancy": "discrepancy",
    "caution": "caution",
    "verify-yourself": "verify",
    "not-a-discrepancy": "cleared"
  };

  var SEVERITY_LABEL = {
    "discrepancy": "Discrepancies to fix",
    "caution": "Worth a second look",
    "verify-yourself": "Check these yourself",
    "not-a-discrepancy": "Not discrepancies, though they look like it"
  };

  var SEVERITY_ORDER = ["discrepancy", "caution", "verify-yourself", "not-a-discrepancy"];

  function renderResult(result) {
    lastResult = result;
    var summary = result.summary || {};

    $("verdict-count").textContent = String(summary.discrepancies || 0);
    $("verdict-text").textContent = result.verdict || "";

    var table = $("headline");
    clear(table);
    function line(label, value) {
      if (!value) { return; }
      var tr = document.createElement("tr");
      tr.appendChild(el("th", null, label));
      tr.appendChild(el("td", null, value));
      table.appendChild(tr);
    }
    line("Date of shipment", result.shipmentDate ? dateText(result.shipmentDate) : "");
    line("…taken from", result.shipmentDateReason);
    line("Last day to present", result.lastPresentationDay ? dateText(result.lastPresentationDay) : "");
    line("…because", result.lastPresentationWhy);
    line("A refusal must reach you by", result.answerExpectedBy ? dateText(result.answerExpectedBy) : "");
    line("Checks run", String(summary.rulesRun || 0) + " of " + String((summary.rulesRun || 0) + (summary.rulesSkipped || 0)));
    line("Cautions", summary.cautions ? String(summary.cautions) : "");
    line("Cleared", summary.cleared ? String(summary.cleared) : "");

    var box = $("findings");
    clear(box);
    SEVERITY_ORDER.forEach(function (severity) {
      var group = (result.findings || []).filter(function (f) { return f.severity === severity; });
      if (!group.length) { return; }
      box.appendChild(el("div", "severity-head " + SEVERITY_CLASS[severity],
        SEVERITY_LABEL[severity] + " (" + group.length + ")"));
      group.forEach(function (f) {
        var card = el("div", "finding " + SEVERITY_CLASS[severity]);
        card.appendChild(el("h3", null, f.title));
        card.appendChild(el("p", null, f.detail));
        if (f.documents && f.documents.length) {
          card.appendChild(el("p", "docs", f.documents.join(" · ")));
        }
        if (f.fix) { card.appendChild(el("p", "fix", f.fix)); }
        card.appendChild(el("p", "meta", f.rule + " · " + f.citation));
        box.appendChild(card);
      });
    });

    var gaps = result.notChecked || [];
    clear($("gaps"));
    gaps.forEach(function (g) {
      $("gaps").appendChild(el("li", null, g.rule + " — needs " + g.need));
    });
    show($("gaps-wrap"), gaps.length > 0);
  }

  /* ---------- the four tabs ---------- */

  function showPanel(name) {
    ["check", "cases", "checks", "figures"].forEach(function (panel) {
      show($("panel-" + panel), panel === name);
    });
    Array.prototype.forEach.call($("tabs").children, function (button) {
      button.setAttribute("aria-pressed", button.getAttribute("data-panel") === name ? "true" : "false");
    });
  }

  /* ---------- saved cases ---------- */

  function renderCases() {
    var table = $("cases-table");
    clear(table);
    var cases = (state && state.cases) || [];
    show($("cases-empty"), cases.length === 0);

    if (cases.length) {
      var head = document.createElement("tr");
      ["Case", "Credit", "Expires", "Saved", "Where it stands", ""].forEach(function (label) {
        head.appendChild(el("th", null, label));
      });
      table.appendChild(head);
    }

    cases.forEach(function (c) {
      var tr = document.createElement("tr");
      tr.appendChild(el("td", null, c.name));
      tr.appendChild(el("td", null, c.credit || "—"));
      tr.appendChild(el("td", null, c.expiry ? dateText(c.expiry) : "—"));
      tr.appendChild(el("td", null, c.savedOn ? dateText(c.savedOn) : "—"));

      var stand = el("td", (c.summary && c.summary.discrepancies) ? "bad" : "ok", c.verdict);
      tr.appendChild(stand);

      var actions = el("td", "actions");
      var open = el("button", "themer", "Open");
      open.type = "button";
      open.addEventListener("click", function () { loadCase(c.id); });
      actions.appendChild(open);

      var remove = el("button", "themer", "Remove");
      remove.type = "button";
      remove.addEventListener("click", function () { removeCase(c.id, c.name); });
      actions.appendChild(remove);

      tr.appendChild(actions);
      table.appendChild(tr);
    });

    $("data-file").textContent = state && state.dataFile
      ? "Saved in " + state.dataFile + " on this machine." : "";
  }

  function loadCase(id) {
    call("GET", "/api/cases/" + encodeURIComponent(id)).then(function (data) {
      writeCase(data.case);
      renderResult(data.result);
      showPanel("check");
      banner("Opened " + data.case.name + ".");
      window.scrollTo(0, 0);
    }).catch(failed);
  }

  function removeCase(id, name) {
    call("DELETE", "/api/cases/" + encodeURIComponent(id)).then(function () {
      banner("Removed " + name + ".");
      if (current && current.id === id) { current = null; }
      return refreshState();
    }).catch(failed);
  }

  /* ---------- the catalogue and the figures ---------- */

  function renderCatalogue() {
    var table = $("catalogue");
    clear(table);
    var head = document.createElement("tr");
    ["Check", "What it tests", "Rule"].forEach(function (label) {
      head.appendChild(el("th", null, label));
    });
    table.appendChild(head);

    var group = "";
    ((state && state.catalogue) || []).forEach(function (entry) {
      if (entry.group !== group) {
        group = entry.group;
        var groupRow = document.createElement("tr");
        var cell = el("td", null, group);
        cell.colSpan = 3;
        cell.className = "severity-head";
        groupRow.appendChild(cell);
        table.appendChild(groupRow);
      }
      var tr = document.createElement("tr");
      tr.appendChild(el("td", null, entry.id));
      var what = el("td");
      what.appendChild(el("strong", null, entry.title));
      what.appendChild(document.createElement("br"));
      what.appendChild(document.createTextNode(entry.tests));
      tr.appendChild(what);
      tr.appendChild(el("td", "cite", entry.citation));
      table.appendChild(tr);
    });
  }

  var FIGURE_FIELDS = [
    ["t-presentation-days", "presentationPeriodDays", "int"],
    ["t-examination-days", "examinationBankingDays", "int"],
    ["t-on-or-about", "onOrAboutDays", "int"],
    ["t-about-tolerance", "aboutTolerancePercent", "float"],
    ["t-quantity-tolerance", "quantityTolerancePercent", "float"],
    ["t-short-draw", "shortDrawTolerancePercent", "float"],
    ["t-insurance", "minInsurancePercent", "float"],
    ["t-originals", "minOriginalsPerDocument", "int"],
    ["t-typo", "typoEditDistance", "int"],
    ["t-decimals", "amountDecimals", "int"]
  ];

  function renderFigures() {
    var table = $("citations");
    clear(table);
    var head = document.createElement("tr");
    ["Figure", "In force", "Where it comes from"].forEach(function (label) {
      head.appendChild(el("th", null, label));
    });
    table.appendChild(head);

    ((state && state.citations) || []).forEach(function (cite) {
      var tr = document.createElement("tr");
      tr.appendChild(el("td", null, cite.key));
      var value = el("td");
      value.appendChild(el("strong", null, cite.figure));
      value.appendChild(document.createElement("br"));
      value.appendChild(document.createTextNode(cite.comment));
      tr.appendChild(value);
      tr.appendChild(el("td", cite.source.indexOf("Not UCP") === 0 ? "cite convention" : "cite", cite.source));
      table.appendChild(tr);
    });

    var th = (state && state.thresholds) || {};
    FIGURE_FIELDS.forEach(function (entry) {
      var input = $(entry[0]);
      if (input) { input.value = String(th[entry[1]] === undefined ? "" : th[entry[1]]); }
    });
    $("t-weekend").value = (th.weekendDays || []).join(", ");
  }

  function saveFigures() {
    var next = { amountDecimalsSet: true, weekendDays: [] };
    FIGURE_FIELDS.forEach(function (entry) {
      var input = $(entry[0]);
      if (!input) { return; }
      next[entry[1]] = entry[2] === "int" ? Math.round(num(input.value)) : num(input.value);
    });
    $("t-weekend").value.split(",").forEach(function (part) {
      part = part.trim();
      if (part !== "") { next.weekendDays.push(Math.round(num(part))); }
    });

    call("PUT", "/api/thresholds", next).then(function () {
      banner("Figures saved. Every check from now on uses them, and the report says so.");
      return refreshState();
    }).then(runCheck).catch(failed);
  }

  function resetFigures() {
    call("PUT", "/api/thresholds", { amountDecimalsSet: false }).then(function () {
      banner("Back to the figures UCP 600 fixes.");
      return refreshState();
    }).then(runCheck).catch(failed);
  }

  /* ---------- the buttons ---------- */

  function runCheck() {
    return call("POST", "/api/check", { case: readCase(), save: false }).then(function (data) {
      current = data.case;
      renderResult(data.result);
      banner("");
    }).catch(failed);
  }

  function saveCase() {
    call("POST", "/api/check", { case: readCase(), save: true }).then(function (data) {
      writeCase(data.case);
      renderResult(data.result);
      banner("Saved as " + data.case.name + ". Open it again from Saved cases after you have fixed the documents.");
      return refreshState();
    }).catch(failed);
  }

  function loadSample() {
    call("GET", "/api/sample").then(function (data) {
      writeCase(data.case);
      renderResult(data.result);
      banner("This is a worked example with faults planted in it on purpose, including three things that look wrong and are not.");
      window.scrollTo(0, 0);
    }).catch(failed);
  }

  function clearForm() {
    current = null;
    writeCase({ id: "", name: "", credit: {}, presentation: {} });
    addRequirement({ kind: "commercial-invoice", wording: "commercial invoice", originals: 1, copies: 0 });
    addDocument({ kind: "commercial-invoice", originals: 1 });
    renderResult({ summary: {}, verdict: "Enter the credit's terms and your documents.", findings: [], notChecked: [] });
    banner("");
  }

  function downloadReport() {
    var c = readCase();
    fetch("/api/report.txt", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ case: c, save: false })
    }).then(function (response) {
      if (!response.ok) {
        return response.text().then(function (text) { throw new Error(text); });
      }
      return response.text();
    }).then(function (text) {
      var blob = new Blob([text], { type: "text/plain;charset=utf-8" });
      var url = URL.createObjectURL(blob);
      var link = document.createElement("a");
      link.href = url;
      link.download = "lc-pre-check.txt";
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
      URL.revokeObjectURL(url);
      banner("The report has been saved to your downloads.");
    }).catch(failed);
  }

  /* ---------- the theme toggle ---------- */

  function setUpTheme() {
    var root = document.documentElement;
    var button = document.querySelector("[data-themer]");
    if (!button) { return; }

    function stored() {
      try { return localStorage.getItem("omsme-theme"); } catch (e) { return null; }
    }
    function currentTheme() {
      var t = root.getAttribute("data-theme");
      if (t) { return t; }
      return (window.matchMedia && window.matchMedia("(prefers-color-scheme: dark)").matches) ? "dark" : "light";
    }
    function paint() {
      var dark = currentTheme() === "dark";
      button.textContent = dark ? "☀ Light" : "☾ Dark";
      button.setAttribute("aria-label", dark ? "Switch to light theme" : "Switch to dark theme");
    }

    paint();
    button.addEventListener("click", function () {
      var next = currentTheme() === "dark" ? "light" : "dark";
      root.setAttribute("data-theme", next);
      try { localStorage.setItem("omsme-theme", next); } catch (e) { /* a private window */ }
      paint();
    });
    if (window.matchMedia) {
      window.matchMedia("(prefers-color-scheme: dark)").addEventListener("change", function () {
        if (!stored()) { paint(); }
      });
    }
  }

  /* ---------- start ---------- */

  function refreshState() {
    return call("GET", "/api/state").then(function (data) {
      state = data;
      renderCases();
      renderCatalogue();
      renderFigures();
    });
  }

  function start() {
    setUpTheme();

    Array.prototype.forEach.call($("tabs").children, function (button) {
      button.addEventListener("click", function () {
        showPanel(button.getAttribute("data-panel"));
      });
    });

    $("add-requirement").addEventListener("click", function () { addRequirement(null); });
    $("add-document").addEventListener("click", function () { addDocument(null); });
    $("run-check").addEventListener("click", runCheck);
    $("save-case").addEventListener("click", saveCase);
    $("load-sample").addEventListener("click", loadSample);
    $("clear-form").addEventListener("click", clearForm);
    $("download-report").addEventListener("click", downloadReport);
    $("save-figures").addEventListener("click", saveFigures);
    $("reset-figures").addEventListener("click", resetFigures);

    clearForm();
    refreshState().catch(failed);
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", start);
  } else {
    start();
  }
})();
