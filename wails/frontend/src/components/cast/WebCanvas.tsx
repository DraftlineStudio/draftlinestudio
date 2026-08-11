// WebCanvas — the character relationship web.
// Performance rules: NO SVG filters, and the force layout is computed
// synchronously up front. The simulation only runs while dragging, so the
// canvas costs ~zero CPU at idle.

import { useEffect, useRef, useCallback } from 'react'
import * as d3 from 'd3'
import type { BookData, Character, RelationshipRecord } from '../../types/draftline'
import { characterColor, characterInitials } from '../../utils/characterVisuals'
import { hexToRgba } from '../../utils/accentColor'

interface WebNode extends d3.SimulationNodeDatum {
  id: string
  name: string
  color: string
  radius: number
}

interface WebLink extends d3.SimulationLinkDatum<WebNode> {
  strength: number
}

export function WebCanvas({ book, selectedId, onSelect }: {
  book: BookData
  selectedId: string | null
  onSelect: (id: string | null) => void
}) {
  const svgRef = useRef<SVGSVGElement>(null)
  const containerRef = useRef<HTMLDivElement>(null)

  // The graph only depends on this data. Everything else (selection,
  // unrelated store updates) must NOT rebuild the simulation.
  const characters = book.story_bible?.characters
  const relationships = book.analysis?.relationships?.relationships

  // Keep the latest handler without invalidating the render callback.
  const onSelectRef = useRef(onSelect)
  onSelectRef.current = onSelect

  const render = useCallback(() => {
    const svgEl = svgRef.current
    const container = containerRef.current
    if (!svgEl || !container) return

    const charMap = new Map<string, Character>()
    for (const c of characters ?? []) charMap.set(c.id, c)

    const connected = new Set<string>()
    const links: WebLink[] = []
    for (const rel of (relationships ?? []) as RelationshipRecord[]) {
      if (!charMap.has(rel.character1_id) || !charMap.has(rel.character2_id)) continue
      connected.add(rel.character1_id)
      connected.add(rel.character2_id)
      links.push({ source: rel.character1_id, target: rel.character2_id, strength: rel.strength })
    }
    const nodes: WebNode[] = [...connected].map(id => {
      const c = charMap.get(id)!
      return {
        id,
        name: c.name,
        color: characterColor(c.name),
        radius: Math.max(14, Math.min(36, 10 + Math.sqrt(c.mention_count || 1) * 2.6)),
      }
    })

    const svg = d3.select(svgEl)
    svg.selectAll('*').remove()
    if (!nodes.length) return

    const width = container.clientWidth
    const height = container.clientHeight || 480
    svg.attr('viewBox', [0, 0, width, height]).attr('width', width).attr('height', height)

    const g = svg.append('g')
    const zoom = d3.zoom<SVGSVGElement, unknown>()
      .scaleExtent([0.3, 3.5])
      .on('zoom', e => g.attr('transform', e.transform))
    svg.call(zoom)
    svg.on('click', () => onSelectRef.current(null))

    const simulation = d3.forceSimulation<WebNode>(nodes)
      .force('link', d3.forceLink<WebNode, WebLink>(links)
        .id(d => d.id)
        .distance(d => 90 + 140 * (1 - d.strength))
        .strength(d => 0.2 + d.strength * 0.5))
      .force('charge', d3.forceManyBody().strength(-420))
      .force('center', d3.forceCenter(width / 2, height / 2))
      .force('collision', d3.forceCollide<WebNode>().radius(d => d.radius + 28))
      .alphaDecay(0.05)
      .stop()

    // Compute the layout NOW, synchronously — no animation loop at idle.
    const steps = Math.ceil(Math.log(simulation.alphaMin()) / Math.log(1 - simulation.alphaDecay()))
    simulation.tick(steps)

    const linkSel = g.append('g').selectAll<SVGPathElement, WebLink>('path')
      .data(links)
      .join('path')
      .attr('class', 'web-thread')
      .attr('fill', 'none')
      .attr('stroke', d => hexToRgba((d.source as WebNode).color, 0.25 + d.strength * 0.45))
      .attr('stroke-width', d => 1 + d.strength * 3.5)
      .attr('stroke-linecap', 'round')

    const nodeSel = g.append('g').selectAll<SVGGElement, WebNode>('g')
      .data(nodes)
      .join('g')
      .attr('class', 'web-orb')
      .style('cursor', 'pointer')
      .on('click', (event, d) => {
        event.stopPropagation()
        onSelectRef.current(d.id)
      })
      .call(d3.drag<SVGGElement, WebNode>()
        .on('start', (event, d) => {
          if (!event.active) simulation.alphaTarget(0.12).restart()
          d.fx = d.x; d.fy = d.y
        })
        .on('drag', (event, d) => { d.fx = event.x; d.fy = event.y })
        .on('end', (event, d) => {
          if (!event.active) simulation.alphaTarget(0) // cools, then stops itself
          d.fx = null; d.fy = null
        }))

    // Soft halo WITHOUT filters — a plain translucent ring is cheap.
    nodeSel.append('circle')
      .attr('r', d => d.radius + 5)
      .attr('fill', d => hexToRgba(d.color, 0.12))
    nodeSel.append('circle')
      .attr('class', 'orb-core')
      .attr('r', d => d.radius)
      .attr('fill', d => hexToRgba(d.color, 0.18))
      .attr('stroke', d => d.color)
      .attr('stroke-width', 1.5)
    nodeSel.append('text')
      .attr('class', 'orb-initials')
      .attr('text-anchor', 'middle')
      .attr('dy', '0.36em')
      .attr('fill', d => d.color)
      .style('font-size', d => `${Math.max(11, d.radius * 0.66)}px`)
      .text(d => characterInitials(d.name))
    nodeSel.append('text')
      .attr('class', 'orb-name')
      .attr('text-anchor', 'middle')
      .attr('dy', d => d.radius + 18)
      .text(d => d.name)

    const position = () => {
      linkSel.attr('d', l => {
        const s = l.source as WebNode
        const t = l.target as WebNode
        const dx = t.x! - s.x!
        const dy = t.y! - s.y!
        const dist = Math.hypot(dx, dy) || 1
        const bend = Math.min(36, dist * 0.1)
        const nx = (s.x! + t.x!) / 2 - (dy / dist) * bend
        const ny = (s.y! + t.y!) / 2 + (dx / dist) * bend
        return `M${s.x},${s.y} Q${nx},${ny} ${t.x},${t.y}`
      })
      nodeSel.attr('transform', d => `translate(${d.x},${d.y})`)
    }
    position()
    simulation.on('tick', position) // only fires during drag

    // Hover: emphasize this character's threads.
    const neighbor = new Map<string, Set<string>>()
    for (const l of links) {
      const s = (l.source as WebNode).id
      const t = (l.target as WebNode).id
      if (!neighbor.has(s)) neighbor.set(s, new Set())
      if (!neighbor.has(t)) neighbor.set(t, new Set())
      neighbor.get(s)!.add(t)
      neighbor.get(t)!.add(s)
    }
    nodeSel
      .on('mouseenter', (_e, d) => {
        const near = neighbor.get(d.id) ?? new Set()
        nodeSel.classed('hushed', n => n.id !== d.id && !near.has(n.id))
        linkSel.classed('hushed', l =>
          (l.source as WebNode).id !== d.id && (l.target as WebNode).id !== d.id)
      })
      .on('mouseleave', () => {
        nodeSel.classed('hushed', false)
        linkSel.classed('hushed', false)
      })

    // Start framed on the whole web.
    const xs = nodes.map(n => n.x!)
    const ys = nodes.map(n => n.y!)
    const pad = 60
    const bw = Math.max(1, Math.max(...xs) - Math.min(...xs) + pad * 2)
    const bh = Math.max(1, Math.max(...ys) - Math.min(...ys) + pad * 2)
    const scale = Math.min(1.6, Math.min(width / bw, height / bh))
    const tx = width / 2 - scale * (Math.min(...xs) + Math.max(...xs)) / 2
    const ty = height / 2 - scale * (Math.min(...ys) + Math.max(...ys)) / 2
    svg.call(zoom.transform, d3.zoomIdentity.translate(tx, ty).scale(scale))

    return () => { simulation.stop() }
  }, [characters, relationships])

  useEffect(() => {
    const cleanup = render()
    let timer: ReturnType<typeof setTimeout> | null = null
    const onResize = () => {
      if (timer) clearTimeout(timer)
      timer = setTimeout(render, 200)
    }
    window.addEventListener('resize', onResize)
    return () => {
      cleanup?.()
      if (timer) clearTimeout(timer)
      window.removeEventListener('resize', onResize)
    }
  }, [render])

  // Selection ring without re-layout.
  useEffect(() => {
    if (!svgRef.current) return
    d3.select(svgRef.current)
      .selectAll<SVGGElement, WebNode>('.web-orb')
      .classed('chosen', d => d.id === selectedId)
  }, [selectedId, characters, relationships])

  return (
    <div className="web-canvas" ref={containerRef}>
      <svg ref={svgRef} />
    </div>
  )
}
