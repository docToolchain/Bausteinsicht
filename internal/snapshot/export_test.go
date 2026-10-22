package snapshot

// SnapshotDirName exposes the unexported snapshotDir constant to the external
// snapshot_test package. Test files are not compiled into the production
// binary, so this keeps the constant internal while letting tests reference
// the real production value instead of a hand-maintained copy that could drift.
const SnapshotDirName = snapshotDir
