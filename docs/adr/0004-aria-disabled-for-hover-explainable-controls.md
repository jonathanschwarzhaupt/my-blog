# Use aria-disabled, not native disabled, for hover-explainable controls

The "Stay Connected" footer's email signup is disabled with a "coming soon" popover that appears on hover/focus. Native `disabled` suppresses pointer and focus events in most browsers, which would silently break the popover trigger — so the control uses `aria-disabled="true"` instead, staying focusable and hoverable, with submission blocked by not wiring an action rather than by the browser refusing interaction.

This is the standard accessibility pattern (WAI-ARIA Authoring Practices; also documented in Sarah Higley's widely-cited "Disabled buttons suck") for any control that needs to explain *why* it's disabled rather than going fully inert. Future "disabled but explains itself" UI on this site should follow the same approach rather than reaching for native `disabled` first.

## Consequences

Controls using this pattern must have their disabled behavior enforced manually (no submit handler wired, default action prevented) since the browser won't do it automatically — an `aria-disabled` control that's accidentally left interactive is a real risk if this isn't remembered.
