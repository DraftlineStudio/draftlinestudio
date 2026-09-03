// Seekable Read Aloud progress bar: fraction fill, per-speaker dialogue tick
// marks, and a diamond playhead. Two size variants share one implementation —
// the compact bar's 4px strip and the expanded panel's 6px version.

export interface SeekTick {
  fraction: number // 0–1 position along the track
  color: string
}

interface SeekBarProps {
  fraction: number // 0–1 playhead position
  ticks: SeekTick[]
  variant: 'bar' | 'panel'
  onSeek: (fraction: number) => void
  disabled?: boolean
}

export default function SeekBar({ fraction, ticks, variant, onSeek, disabled }: SeekBarProps) {
  const pct = `${(Math.max(0, Math.min(1, fraction)) * 100).toFixed(2)}%`

  const handlePointerDown = (e: React.PointerEvent<HTMLDivElement>) => {
    if (disabled) return
    const rect = e.currentTarget.getBoundingClientRect()
    if (rect.width <= 0) return
    onSeek(Math.max(0, Math.min(1, (e.clientX - rect.left) / rect.width)))
  }

  return (
    <div
      className={`rap-seek rap-seek--${variant}${disabled ? ' rap-seek--disabled' : ''}`}
      onPointerDown={handlePointerDown}
      title={disabled ? undefined : 'Seek'}
      role="slider"
      aria-label="Playback position"
      aria-valuemin={0}
      aria-valuemax={100}
      aria-valuenow={Math.round(Math.max(0, Math.min(1, fraction)) * 100)}
      aria-disabled={disabled || undefined}
    >
      <span className="rap-seek-fill" style={{ width: pct }} />
      {ticks.map((tick, i) => (
        <span
          key={i}
          className="rap-seek-tick"
          style={{ left: `${(tick.fraction * 100).toFixed(2)}%`, background: tick.color }}
        />
      ))}
      <span className="rap-seek-thumb" style={{ left: pct }} />
    </div>
  )
}
