//go:build !windows

package main

// Non-Windows platforms register associations declaratively at packaging
// time: build/darwin/Info.plist (CFBundleDocumentTypes) on macOS and
// build/linux/ (.desktop + shared-mime-info) on Linux. Nothing to do at
// runtime — see docs/backend/FILE-ASSOCIATIONS.md.
func registerFileAssociations() error { return nil }
