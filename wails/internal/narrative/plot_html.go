package narrative

import (
	"fmt"
	"html"
	"strings"
)

// RenderPlotHTML is a standalone local diagnostic, with no network resources,
// scripts, production frontend changes, or author-maintained planning data.
func RenderPlotHTML(d Document, p PlotAnalysis) (string, error) {
	if err := d.Validate(); err != nil {
		return "", err
	}
	if p.Graph.Revision != d.Revision {
		return "", fmt.Errorf("stale plot projection")
	}
	if _, err := BuildGraph(d, p.Graph.Observations, p.Graph.Nodes, p.Graph.Relations); err != nil {
		return "", err
	}
	for _, a := range p.Attributes {
		for _, source := range []Anchor{a.Source, a.SubjectSource, a.ValueSource, a.ContextSource} {
			if _, err := d.Position(source); err != nil {
				return "", err
			}
		}
	}
	var out strings.Builder
	esc := html.EscapeString
	out.WriteString(`<!doctype html><html lang="en"><meta charset="utf-8"><title>Draftline — plot evidence prototype</title><style>body{font:16px system-ui;margin:2rem;background:#111827;color:#e5e7eb}a{color:#93c5fd}.scroll{overflow:auto}svg{background:#172033}article{padding:1rem;border:1px solid #475569;margin:1rem 0}small{color:#cbd5e1}blockquote{white-space:pre-wrap}table{border-collapse:collapse}td,th{padding:.6rem;border:1px solid #475569}svg text{fill:#e5e7eb}h1{font-size:1.5rem}</style><h1>Plot evidence — experimental candidate braid</h1><p>Horizontal axis: manuscript disclosure position. Dashed connections are candidate references, not proven causality. Unlinked developments remain visible.</p><ul>`)
	for _, s := range p.Limitations {
		fmt.Fprintf(&out, "<li>%s</li>", esc(s))
	}
	out.WriteString("</ul>")
	obs := map[string]Observation{}
	for _, o := range p.Graph.Observations {
		if _, err := d.Position(o.Source); err != nil {
			return "", err
		}
		obs[o.ID] = o
	}
	nodes := map[string]Development{}
	for _, n := range p.Graph.Nodes {
		nodes[n.ID] = n
	}
	lanes := map[string]int{}
	claims := map[string]PlotClaim{}
	for _, c := range p.Claims {
		claims[c.Observation.ID] = c
	}
	for i, t := range p.Graph.Threads {
		for _, n := range t.NodeIDs {
			lanes[n] = i + 1
		}
	}
	width := max(900, len(d.Blocks)*14+260)
	height := (len(p.Graph.Threads) + 2) * 80
	point := func(n Development) (int, int) {
		o := obs[n.EstablishedBy]
		bi, _ := d.Position(o.Source)
		fraction := 0
		if len(d.Blocks[bi].Text) > 0 {
			fraction = 14 * o.Source.Start / len(d.Blocks[bi].Text)
		}
		return 240 + bi*14 + fraction, 50 + lanes[n.ID]*80
	}
	fmt.Fprintf(&out, `<div class="scroll"><svg width="%d" height="%d" role="img" aria-label="Candidate activity strands in manuscript order">`, width, height)
	for i := 0; i <= len(p.Graph.Threads); i++ {
		label := "Unlinked disclosures"
		if i > 0 {
			thread := p.Graph.Threads[i-1]
			first := nodes[thread.NodeIDs[0]]
			label = fmt.Sprintf("%d: %s", i, claims[first.EstablishedBy].Target)
		}
		fmt.Fprintf(&out, `<text x="8" y="%d" font-size="11">%s</text><path d="M230 %d H%d" stroke="#334155"/>`, 55+i*80, esc(label), 50+i*80, width)
	}
	for _, edge := range p.Graph.Relations {
		a, b := nodes[edge.From], nodes[edge.To]
		x1, y1 := point(a)
		x2, y2 := point(b)
		fmt.Fprintf(&out, `<a href="#node-%s"><path d="M%d %d L%d %d" stroke="#fbbf24" stroke-width="2" stroke-dasharray="5 4"><title>%s</title></path></a>`, esc(b.ID), x1, y1, x2, y2, esc(edge.Rule))
	}
	for i, n := range p.Graph.Nodes {
		x, y := point(n)
		fmt.Fprintf(&out, `<a href="#node-%s"><circle cx="%d" cy="%d" r="7" fill="#60a5fa"><title>%s</title></circle><text x="%d" y="%d" font-size="11">%d</text></a>`, esc(n.ID), x, y, esc(n.Label), x, y-12, i+1)
	}
	out.WriteString("</svg></div><h2>Developments and evidence</h2>")
	for i, n := range p.Graph.Nodes {
		o := obs[n.EstablishedBy]
		fmt.Fprintf(&out, `<article id="node-%s"><h3>%d. %s</h3><small>%s · %s · bytes %d–%d</small><blockquote>%s</blockquote>`, esc(n.ID), i+1, esc(o.Kind), esc(o.Mode), esc(o.Source.BlockID), o.Source.Start, o.Source.End, esc(o.Source.Quote))
		for _, e := range p.Graph.Relations {
			if e.To == n.ID {
				fmt.Fprintf(&out, `<p>Candidate connection: <a href="#node-%s">earlier development</a> — %s</p>`, esc(e.From), esc(e.Rule))
				for _, w := range e.Witnesses {
					fmt.Fprintf(&out, "<blockquote>%s</blockquote>", esc(obs[w].Source.Quote))
				}
			}
		}
		out.WriteString("</article>")
	}
	out.WriteString("<h2>Attribute mentions — not verified facts or contradiction verdicts</h2><p>Object identity, assertion scope, and reference time remain unresolved. Different colors may describe different lights or a state change.</p><table><tr><th>Surface subject</th><th>Property</th><th>Value</th><th>Context</th><th>Passage</th></tr>")
	for _, a := range p.Attributes {
		value := a.Value
		if a.PreviousValue != "" {
			value = a.PreviousValue + " → " + value
		}
		fmt.Fprintf(&out, "<tr><td>%s</td><td>%s</td><td>%s</td><td>%s / %s</td><td>%s<details><summary>Full context</summary>%s</details><small>%s</small></td></tr>", esc(a.SubjectSurface), esc(a.Property), esc(value), esc(a.Mode), esc(a.TemporalStatus), esc(a.Source.Quote), esc(a.ContextSource.Quote), esc(a.Source.BlockID))
	}
	out.WriteString("</table></html>")
	return out.String(), nil
}
