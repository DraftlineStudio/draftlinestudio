// Public project links. Kept in one place so the app menu, settings, and any
// future help surface all point at the same places.

export const REPO_URL = 'https://github.com/DraftlineStudio/draftlinestudio'

// The help site, written for writers. Deliberately not the repository README,
// which is written for whoever is building Draftline rather than using it --
// somebody looking for how the export flow works should not land in a Go
// project. Its own address too, so where the code lives can change without
// stranding the menu item.
export const DOCS_URL = 'https://docs.draftline.ink'
export const RELEASES_URL = `${REPO_URL}/releases`

// Pre-labelled new-issue links. The version goes into the body so a report
// arrives with the one fact every triage starts with.
export function newIssueUrl(kind: 'bug' | 'feature', appVersion: string): string {
  const version = appVersion ? `Draftline version: ${appVersion}\n\n` : ''
  const params = kind === 'bug'
    ? { labels: 'bug', title: 'Bug: ', body: `${version}What happened:\n\nWhat I expected:\n\nSteps to reproduce:\n` }
    : { labels: 'enhancement', title: 'Feature request: ', body: `${version}What I'd like:\n\nWhy it would help:\n` }
  const query = new URLSearchParams(params).toString()
  return `${REPO_URL}/issues/new?${query}`
}
