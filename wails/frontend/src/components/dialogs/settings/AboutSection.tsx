// About Section - version, project links, and open-source credits

import { useState } from 'react'
import { BrowserOpenURL } from '../../../../wailsjs/runtime/runtime'
import { WEBSITE_URL, RELEASES_URL } from '../../../services/links'
import { CREDIT_GROUPS } from './credits'
import type { AboutSectionProps } from './types'

export default function AboutSection({ appVersion }: AboutSectionProps) {
  const [openCredit, setOpenCredit] = useState<string | null>(null)

  const open = (url: string) => BrowserOpenURL(url)

  return (
    <>
      <div className="about-header">
        <img className="about-logo" src="/logo.svg" alt="" />
        <div className="about-identity">
          <div className="about-name">Draftline</div>
          <div className="about-version">Version {appVersion || '…'}</div>
          <div className="about-tagline">A writing environment for novelists.</div>
        </div>
      </div>
      <div className="about-links">
        <button className="dialog-btn" onClick={() => open(WEBSITE_URL)}>draftline.ink</button>
        <button className="dialog-btn" onClick={() => open(RELEASES_URL)}>Releases on GitHub</button>
      </div>

      <div className="settings-section-label">Open-Source Credits</div>
      <div className="settings-hint about-credits-intro">
        Draftline is built on these open-source projects. Click one for its
        links and license.
      </div>

      {CREDIT_GROUPS.map(group => (
        <div key={group.label} className="about-credit-group">
          <div className="about-credit-group-label">{group.label}</div>
          <div className="about-credit-list">
            {group.credits.map(credit => {
              const expanded = openCredit === credit.id
              return (
                <div key={credit.id} className={`about-credit${expanded ? ' expanded' : ''}`}>
                  <button
                    className="about-credit-row"
                    onClick={() => setOpenCredit(expanded ? null : credit.id)}
                    aria-expanded={expanded}
                  >
                    <svg className="about-credit-chevron" width="8" height="8" viewBox="0 0 8 8" fill="none" stroke="currentColor" strokeWidth="1.4" strokeLinecap="round" strokeLinejoin="round">
                      <path d="M2.5 1.5 5.5 4 2.5 6.5" />
                    </svg>
                    <span className="about-credit-name">{credit.name}</span>
                    <span className="about-credit-license">{credit.license}</span>
                  </button>
                  {expanded && (
                    <div className="about-credit-detail">
                      <div className="about-credit-role">{credit.role}</div>
                      <div className="about-credit-actions">
                        <button className="dialog-btn about-credit-btn" onClick={() => open(credit.website)}>Website</button>
                        {credit.source && (
                          <button className="dialog-btn about-credit-btn" onClick={() => open(credit.source!)}>Source</button>
                        )}
                        <button className="dialog-btn about-credit-btn" onClick={() => open(credit.licenseUrl)}>License</button>
                      </div>
                    </div>
                  )}
                </div>
              )
            })}
          </div>
        </div>
      ))}
    </>
  )
}
