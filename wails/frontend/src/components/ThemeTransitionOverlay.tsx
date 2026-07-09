import { useEffect, useState, useMemo } from 'react'

interface ThemeTransitionOverlayProps {
  isTransitioning: boolean
  targetTheme: 'light' | 'dark'
}

interface Star {
  id: number
  x: number
  y: number
  size: number
  delay: number
  duration: number
}

interface Cloud {
  id: number
  x: number
  y: number
  scale: number
  delay: number
  speed: number
}

export default function ThemeTransitionOverlay({ isTransitioning, targetTheme }: ThemeTransitionOverlayProps) {
  const [visible, setVisible] = useState(false)
  const [phase, setPhase] = useState<'enter' | 'hold' | 'exit'>('enter')

  // Generate random stars for night sky
  const stars = useMemo<Star[]>(() => {
    return Array.from({ length: 50 }, (_, i) => ({
      id: i,
      x: Math.random() * 100,
      y: Math.random() * 100,
      size: Math.random() * 2 + 1,
      delay: Math.random() * 0.5,
      duration: Math.random() + 0.5
    }))
  }, [])

  // Generate random clouds for day sky
  const clouds = useMemo<Cloud[]>(() => {
    return Array.from({ length: 6 }, (_, i) => ({
      id: i,
      x: Math.random() * 120 - 10,
      y: Math.random() * 60 + 20,
      scale: Math.random() * 0.5 + 0.5,
      delay: Math.random() * 0.3,
      speed: Math.random() * 10 + 15
    }))
  }, [])

  useEffect(() => {
    if (isTransitioning) {
      setVisible(true)
      setPhase('enter')

      // Hold phase after sky fades in
      const holdTimer = setTimeout(() => setPhase('hold'), 500)

      // Exit phase - let CSS animations handle the fade out
      const exitTimer = setTimeout(() => setPhase('exit'), 2400)

      // Hide completely after animations finish
      const hideTimer = setTimeout(() => {
        setVisible(false)
        setPhase('enter')
      }, 3200)

      // Only clear timers if component unmounts, not when isTransitioning changes
      // This prevents the race condition where isTransitioning=false clears hideTimer
      return () => {
        clearTimeout(holdTimer)
        clearTimeout(exitTimer)
        clearTimeout(hideTimer)
      }
    } else {
      // When isTransitioning becomes false, ensure we clean up
      // Use a small delay to let any in-flight timers complete
      const cleanupTimer = setTimeout(() => {
        setVisible(false)
        setPhase('enter')
      }, 300)
      return () => clearTimeout(cleanupTimer)
    }
  }, [isTransitioning])

  if (!visible) return null

  return (
    <div className={`theme-transition-overlay ${phase}`} data-theme-target={targetTheme}>
      {targetTheme === 'dark' ? (
        // Night sky with stars and moon
        <div className="theme-sky theme-sky-night">
          <div className="stars-container">
            {stars.map(star => (
              <div
                key={star.id}
                className="star"
                style={{
                  left: `${star.x}%`,
                  top: `${star.y}%`,
                  width: `${star.size}px`,
                  height: `${star.size}px`,
                  animationDelay: `${star.delay}s`,
                  animationDuration: `${star.duration}s`
                }}
              />
            ))}
          </div>
          <div className="moon">
            <div className="moon-glow" />
            <div className="moon-body" />
            <div className="moon-shadow" />
          </div>
        </div>
      ) : (
        // Day sky with sun and clouds
        <div className="theme-sky theme-sky-day">
          <div className="sun">
            <div className="sun-glow" />
            <div className="sun-body" />
            <div className="sun-rays">
              {Array.from({ length: 12 }, (_, i) => (
                <div key={i} className="sun-ray" style={{ transform: `rotate(${i * 30}deg)` }} />
              ))}
            </div>
          </div>
          <div className="clouds-container">
            {clouds.map(cloud => (
              <div
                key={cloud.id}
                className="cloud"
                style={{
                  left: `${cloud.x}%`,
                  top: `${cloud.y}%`,
                  transform: `scale(${cloud.scale})`,
                  animationDelay: `${cloud.delay}s`,
                  animationDuration: `${cloud.speed}s`
                }}
              >
                <div className="cloud-puff cloud-puff-1" />
                <div className="cloud-puff cloud-puff-2" />
                <div className="cloud-puff cloud-puff-3" />
                <div className="cloud-puff cloud-puff-4" />
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}
