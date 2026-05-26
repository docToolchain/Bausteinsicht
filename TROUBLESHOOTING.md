# Troubleshooting Guide

Quick troubleshooting for common issues with Bausteinsicht. Use this guide to diagnose and resolve problems.

## Table of Contents

1. [Sync Issues](#sync-issues)
2. [Export Issues](#export-issues)
3. [File & Permission Issues](#file--permission-issues)
4. [Performance Issues](#performance-issues)
5. [Installation Issues](#installation-issues)
6. [Debug Techniques](#debug-techniques)
7. [Emergency Recovery](#emergency-recovery)
8. [FAQ](#faq)

---

## Sync Issues

### Sync Hangs or Never Completes

**Symptoms:** `bausteinsicht sync` blocks indefinitely without completing.

**Solutions (in order):**

1. **Check sync state file exists**
   ```bash
   ls -la .bausteinsicht-sync
   ```
   If missing, it will be created on next sync.

2. **Validate model syntax**
   ```bash
   bausteinsicht validate
   ```
   Fix any validation errors (unknown kinds, missing titles, invalid relationships).

3. **Check model file permissions**
   ```bash
   ls -la architecture.jsonc
   chmod 644 architecture.jsonc  # if needed
   ```

4. **Check draw.io file**
   - Open `architecture.drawio` in draw.io visually
   - Look for corrupted cells or extreme coordinates (e.g., x = 999999)
   - Try saving the file in draw.io to validate

5. **Reset sync state (nuclear option)**
   ```bash
   rm .bausteinsicht-sync
   bausteinsicht sync
   ```
   This re-syncs from scratch, overwriting draw.io.

### Sync Creates Unwanted Changes

**Symptoms:** Sync modifies elements you didn't change in the model.

**Cause:** Conflict detection mismatch; one side was edited without proper sync.

**Solutions:**

1. **Review changes before committing**
   ```bash
   git diff architecture.drawio
   ```

2. **Understand conflict resolution** — model always wins. If draw.io has manual edits not in the model, sync will overwrite them.

3. **Recover draw.io state**
   ```bash
   git checkout HEAD -- architecture.drawio  # restore to last git version
   bausteinsicht sync
   ```

### Duplicate Elements After Sync

**Symptoms:** Elements appear twice in draw.io after sync.

**Cause:** `.bausteinsicht-sync` state is stale; sync doesn't recognize existing elements.

**Solution:**
```bash
rm .bausteinsicht-sync
bausteinsicht sync  # full re-sync
```

---

## Export Issues

### Export Output is Empty or Invalid

**Symptoms:** `bausteinsicht export-diagram` produces blank files or invalid syntax.

**Check:**

1. **View has elements**
   ```bash
   bausteinsicht show context  # or your view key
   ```
   If no output, the view matches no elements.

2. **View filter is correct**
   ```bash
   bausteinsicht validate  # check view definitions
   ```

3. **Export for specific format**
   ```bash
   bausteinsicht export-diagram --format mermaid --view context
   ```

### PNG/SVG Export Fails

**Symptoms:** `Export failed` or permission errors when exporting to images.

**Requirements:**

- Draw.io CLI must be installed
- `dbus` daemon running (Linux/containers)
- Sufficient disk space
- Valid input diagram

**Debug:**

```bash
# Check draw.io binary
which drawio || which draw.io

# For containers, ensure dbus is running
ps aux | grep dbus  # should see dbus-daemon

# Try export with verbose
bausteinsicht export --verbose
```

---

## File & Permission Issues

### Permission Denied: Cannot Read architecture.jsonc

**Solution:**
```bash
chmod 644 architecture.jsonc
```

### Permission Denied: Cannot Write .bausteinsicht-sync

**Cause:** Directory is read-only or filesystem is full.

**Solution:**
```bash
# Check directory permissions
ls -ld .  # should have write permission (w bit set)

# Check disk space
df -h .  # ensure sufficient space

# Fix permissions
chmod 755 .  # or just 644 for files
```

### Invalid JSONC Syntax Error

**Symptoms:** `loading model: invalid JSONC` error.

**Validation:**

1. **Use a JSON schema-aware editor** (VS Code with JSON Schema extension)
2. **Validate manually**
   ```bash
   # Check for syntax errors (offline)
   bausteinsicht validate
   ```
3. **Common mistakes:**
   - Trailing commas in arrays/objects
   - Unquoted keys
   - Comments outside strings
   - Missing colons in key-value pairs

---

## Performance Issues

### Sync is Slow (>10 seconds)

**Cause:** Large model or slow I/O.

**Check model size:**
```bash
bausteinsicht validate --verbose
# Look for element count and nesting depth
```

**Optimize:**

- Split large models into multiple files (v2 feature: multi-model workspace)
- Reduce element count by removing unused elements
- Move to faster storage (SSD instead of network drive)

### High Memory Usage During Sync

**Cause:** Deep nesting or large relationship count.

**Check:**
```bash
bausteinsicht validate  # see element/relationship count
```

**Note:** v1 loads entire model into memory; this is expected for models with >1000 elements.

---

## Installation Issues

### Go Tool Not Found

**Symptoms:** `command not found: bausteinsicht`

**Solution:**
```bash
# Ensure $GOPATH/bin is in PATH
export PATH="$PATH:$(go env GOPATH)/bin"

# Add to ~/.bashrc or ~/.zshrc permanently
echo 'export PATH="$PATH:$(go env GOPATH)/bin"' >> ~/.bashrc
source ~/.bashrc

# Now reinstall
go install github.com/docToolchain/Bausteinsicht/cmd/bausteinsicht@latest
```

### Draw.io Binary Not Found (Export Fails)

**Linux:**
```bash
# Install draw.io CLI
curl -s https://get.draw.io/draw.io.tar.gz | tar -xz
# Add to PATH or use full path
```

**macOS:**
```bash
brew install draw-io
```

**Docker/Container:**
```bash
# Ensure dbus-daemon is running
sudo service dbus start  # or systemctl start dbus
```

---

## Debug Techniques

### Enable Verbose Output

```bash
bausteinsicht sync --verbose
bausteinsicht validate --verbose
```

Verbose mode prints detailed diagnostic info.

### Inspect Files Directly

**JSONC Model:**
```bash
# Validate syntax (offline)
cat architecture.jsonc | jq empty

# Pretty-print with comments preserved
cat architecture.jsonc
```

**Draw.io XML:**
```bash
# Extract structure (requires xmllint)
xmllint --format architecture.drawio | head -100

# Count elements
grep -c 'id="' architecture.drawio
```

**Sync State:**
```bash
# Inspect state file (it's JSON)
cat .bausteinsicht-sync | jq '.checksums | keys'
```

### Check Git History

```bash
# See what changed in last sync
git log -p -- architecture.jsonc | head -50

# Compare to last good version
git show HEAD:architecture.jsonc | diff - architecture.jsonc
```

---

## Emergency Recovery

### Restore to Last Git Commit

```bash
# Revert all changes
git checkout HEAD -- architecture.jsonc architecture.drawio

# Resync from model
bausteinsicht sync
```

### Start from Scratch

```bash
# Backup current files
cp architecture.jsonc architecture.jsonc.bak
cp architecture.drawio architecture.drawio.bak

# Reset
rm .bausteinsicht-sync
bausteinsicht sync
```

### Recover from Corrupted Draw.io

If `architecture.drawio` is corrupted (XML parse error):

```bash
# Use git version
git checkout HEAD -- architecture.drawio

# Or recreate from model
rm architecture.drawio
bausteinsicht sync  # creates fresh diagram
```

---

## FAQ

**Q: How do I undo the last sync?**
A: Sync changes are written to both files, so use git: `git checkout HEAD -- architecture.jsonc architecture.drawio`

**Q: Can I manually edit architecture.jsonc?**
A: Yes, edit freely. Sync will merge your changes with draw.io edits (model wins on conflict).

**Q: What if I accidentally deleted an element?**
A: Restore from git: `git checkout HEAD -- architecture.jsonc`, then `bausteinsicht sync`

**Q: How big can my model be?**
A: No hard limit, but expect slowdowns with >1000 elements. Monitor with `bausteinsicht validate --verbose`.

**Q: Can I have multiple models?**
A: Not in v1. v2 will support multi-model workspaces (link architectures).

**Q: How do I debug sync conflicts?**
A: Use `--verbose` flag and check git diff to see which side won.

**Q: Where are logs stored?**
A: Logs are printed to stdout/stderr, not persisted. Redirect to file: `bausteinsicht sync > sync.log 2>&1`

**Q: How do I reset everything?**
A: Delete `.bausteinsicht-sync` and resync from the model: `rm .bausteinsicht-sync && bausteinsicht sync`

**Q: Is my data safe?**
A: Yes. All changes are persisted to `architecture.jsonc` and `architecture.drawio`. Sync state (`.bausteinsicht-sync`) is just metadata and can be regenerated.

**Q: What if the draw.io file is huge?**
A: This is normal if the model has many elements. Sync performance is linear with element count.

---

## Still Stuck?

1. **Check GitHub Issues:** https://github.com/docToolchain/Bausteinsicht/issues
2. **Enable Verbose Mode:** `bausteinsicht sync --verbose`
3. **Check File Integrity:** `bausteinsicht validate`
4. **Report a Bug:** Include output from `bausteinsicht validate --verbose` and git status

Remember: All data is stored in `architecture.jsonc`, so you can safely delete everything else and recreate it from the model.
