# Kandev Design Task Template — Pre-Flight Checklist

> **Status**: Proposal — pending Kandev workflow admin approval
> **Author**: M1W2 design session retrospective
> **Date**: 2026-10-23

---

## Proposal

Add the following section to the Kandev design task template so it is automatically included in every new design task's prompt.

---

## Template Text

Add this block to the design task template, after the goal/inputs sections and before the constraints:

```markdown
## Pre-Flight Checklist (complete before designing)

Before designing ANY screen, complete these steps:

### 1. Read the Penpot file's Read me page
- Every Penpot file has a "Read me" page with file-specific conventions, color mappings, and known issues
- Read it first — it contains lessons from previous design sessions

### 2. Audit existing screens for component patterns
- Open 2-3 existing screens that are similar to what you're designing
- Note which DS components they use (Input, Select, Badge, Button variants, etc.)
- Note which local library components they use (Shell, Sidebar, Topbar, etc.)
- Note which elements are raw shapes (no DS equivalent)

### 3. Inspect DS component internals before first use
- Before using any DS component for the first time, inspect its internal structure:
  ```
  const comp = dsLib.components.find(c => c.mainInstance()?.name === 'COMPONENT_NAME');
  const struct = penpotUtils.shapeStructure(comp.mainInstance(), 5);
  ```
- Check for: hidden children, text fill colors, flex properties, clipping risks
- The 7K DS file's Read me page documents known issues per component

### 4. Write helper functions BEFORE starting
- `replaceInBoard(boardId, mappings)` — batch replace elements
- `fixTextCentering(shape)` — adjust flex padding after resize
- `setPosition(shape, x, y)` — wrapper for penpotUtils.setParentXY
- Define these once, reuse across all boards

### 5. Plan batch operations
- One tool call per board, not per element type
- Group all replacements for a single board into one execute_code call
- Use `penpotUtils.setParentXY()` for ALL positioning (never `shape.x = value`)

### 6. Know the variant access pattern
```javascript
// CORRECT — access variants through the container
const comp = dsLib.components.find(c => c.mainInstance()?.name === 'badge / default');
const container = comp.mainInstance().parent;
const variants = container.variants.variantComponents();
const specific = variants.find(v => v.variantProps.Tone === 'warning');
const instance = specific.instance();

// WRONG — searching by variantProps returns toast/banner, not badge
const wrong = dsLib.components.find(c => c.variantProps?.Tone === 'warning');
```

### 7. Know the color override mapping
The DS uses magenta (#FF00FF) as default brand. Inari Console uses burnt orange (#EE560D).
When designing for Inari, override DS component fills/strokes:
- `#FF00FF` → `#EE560D` (brand primary)
- `#FF33FF` → `#E56F3F` (brand hover)
- `#D600D6` → `#C44813` (brand active)
- `#330033` → `#2B1C17` (brand subtle bg)
- `#800080` → `#8B310D` (brand muted)
- Do NOT override semantic status colors (danger, warning, success, info)
```

---

## Rationale

The M1W2 design session (multi-repo GitOps UI, 12 boards) took significantly longer than expected due to avoidable issues:

| Issue | Time Cost | Prevented By |
|-------|-----------|--------------|
| Raw shapes instead of DS components | ~30 min redo | Step 2 (audit existing screens) |
| Radio component clipping/border | ~40 min debug + workaround + fix | Step 3 (inspect internals) |
| Wrong variant search pattern | ~20 min trial and error | Step 6 (variant access pattern) |
| Magenta instead of Inari orange | ~15 min color override | Step 7 (color mapping) |
| Text not centered after resize | ~15 min manual fix | Step 4 (helper function) |
| One-by-one board updates | ~20 min extra calls | Step 5 (batch operations) |

**Total avoidable time**: ~2 hours of a ~4 hour session.

---

## Implementation Notes

- The checklist references the 7K DS file's Read me page — ensure that page stays up to date as components are fixed/added
- The Inari Console Read me page has a Quick Reference Index pointing to all sections
- The inari-ui `AGENTS.md` has a "Penpot Design Conventions" section for coding agents
- Consider adding a Kandev workflow step that verifies the checklist was completed before allowing the design review step
