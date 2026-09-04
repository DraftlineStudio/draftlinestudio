export default function NoAIProviderSetup({ onOpenSettings }: { onOpenSettings: () => void }) {
  return (
    <div className="ai-setup-pane">
      <div className="ai-setup-icon">
        <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
          <path d="M12 2v4m0 12v4M4.93 4.93l2.83 2.83m8.48 8.48l2.83 2.83M2 12h4m12 0h4M4.93 19.07l2.83-2.83m8.48-8.48l2.83-2.83"/>
        </svg>
      </div>
      <div className="ai-setup-title">Set Up an AI Provider</div>
      <p className="ai-setup-desc">
        Choose Codex, Claude Code, an API key, or a local model to power AI Studio.
      </p>
      <button className="ai-run-btn" onClick={onOpenSettings}>
        Open AI Settings →
      </button>
    </div>
  )
}
