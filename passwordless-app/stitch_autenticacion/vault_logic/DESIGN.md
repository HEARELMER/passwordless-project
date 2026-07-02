---
name: Vault Logic
colors:
  surface: '#131313'
  surface-dim: '#131313'
  surface-bright: '#393939'
  surface-container-lowest: '#0e0e0e'
  surface-container-low: '#1c1b1b'
  surface-container: '#201f1f'
  surface-container-high: '#2a2a2a'
  surface-container-highest: '#353534'
  on-surface: '#e5e2e1'
  on-surface-variant: '#c6c5d4'
  inverse-surface: '#e5e2e1'
  inverse-on-surface: '#313030'
  outline: '#908f9d'
  outline-variant: '#454652'
  surface-tint: '#bdc2ff'
  primary: '#bdc2ff'
  on-primary: '#1b247f'
  primary-container: '#1a237e'
  on-primary-container: '#8690ee'
  inverse-primary: '#4c56af'
  secondary: '#ffffff'
  on-secondary: '#273500'
  secondary-container: '#bef500'
  on-secondary-container: '#536d00'
  tertiary: '#bbc3ff'
  on-tertiary: '#112286'
  tertiary-container: '#102285'
  on-tertiary-container: '#8290f4'
  error: '#ffb4ab'
  on-error: '#690005'
  error-container: '#93000a'
  on-error-container: '#ffdad6'
  primary-fixed: '#e0e0ff'
  primary-fixed-dim: '#bdc2ff'
  on-primary-fixed: '#000767'
  on-primary-fixed-variant: '#343d96'
  secondary-fixed: '#bef500'
  secondary-fixed-dim: '#a6d700'
  on-secondary-fixed: '#151f00'
  on-secondary-fixed-variant: '#3a4d00'
  tertiary-fixed: '#dfe0ff'
  tertiary-fixed-dim: '#bbc3ff'
  on-tertiary-fixed: '#000d5f'
  on-tertiary-fixed-variant: '#2d3c9c'
  background: '#131313'
  on-background: '#e5e2e1'
  surface-variant: '#353534'
typography:
  display-sm:
    fontFamily: Inter
    fontSize: 32px
    fontWeight: '700'
    lineHeight: 40px
    letterSpacing: -0.02em
  headline-lg:
    fontFamily: Inter
    fontSize: 24px
    fontWeight: '600'
    lineHeight: 32px
    letterSpacing: -0.01em
  headline-lg-mobile:
    fontFamily: Inter
    fontSize: 20px
    fontWeight: '600'
    lineHeight: 28px
  body-md:
    fontFamily: Inter
    fontSize: 16px
    fontWeight: '400'
    lineHeight: 24px
  body-sm:
    fontFamily: Inter
    fontSize: 14px
    fontWeight: '400'
    lineHeight: 20px
  label-md:
    fontFamily: Geist
    fontSize: 12px
    fontWeight: '500'
    lineHeight: 16px
    letterSpacing: 0.05em
  mono-code:
    fontFamily: Geist
    fontSize: 13px
    fontWeight: '400'
    lineHeight: 18px
rounded:
  sm: 0.125rem
  DEFAULT: 0.25rem
  md: 0.375rem
  lg: 0.5rem
  xl: 0.75rem
  full: 9999px
spacing:
  base: 4px
  xs: 8px
  sm: 12px
  md: 16px
  lg: 24px
  xl: 32px
  container-padding: 20px
  gutter: 16px
---

## Brand & Style

The design system is engineered for high-stakes security environments where trust and technical precision are paramount. The brand personality is **Authoritative, Kinetic, and Hyper-Reliable**. It balances the stoicism of traditional banking with the forward-leaning energy of decentralized cryptography.

The visual style is **Technological Minimalism** with a focus on **Tonal Layering**. It avoids decorative clutter in favor of meaningful motion and structural clarity. The interface should feel like a high-performance instrument—responsive, precise, and immutable. We utilize a dark-first aesthetic to reduce visual noise and emphasize "active" cryptographic states.

## Colors

The palette is anchored by **Deep Security Blue**, representing stability and the "Vault" state. This is contrasted sharply by **Cyber Lime**, which is reserved strictly for "Success," "Authenticated," and "Actionable" states, creating a high-contrast focal point that guides the user through complex flows.

- **Primary (Indigo 900):** Used for brand identity, primary actions, and secure backgrounds.
- **Accent (Cyber Lime):** Used for biometric confirmation, successful digital signatures, and active toggle states.
- **Background:** A deep **Dark Charcoal** (#121212) provides a premium, low-light environment that reduces eye strain during frequent authentication checks.
- **Surface Containers:** Elevated backgrounds (#1E1E1E) distinguish cards and interactive modules from the base layout.

## Typography

This design system utilizes **Inter** for all primary interface elements due to its exceptional legibility at small sizes on mobile displays. For technical metadata, cryptographic hashes, and labels, we introduce **Geist** to provide a distinct "developer-friendly" and precise character.

- **Headlines:** Should be tight and bold to convey strength.
- **Labels:** Use Geist in uppercase with slight letter spacing to denote system-level information or "Read-Only" secure data.
- **Numerical Data:** Always use tabular figures to ensure alignment when displaying cryptographic challenges or countdown timers.

## Layout & Spacing

The layout follows a **Fluid Mobile Grid** based on a 4px baseline rhythm. 

- **Margins:** Standard horizontal padding is 20px to keep content away from screen edges while maintaining a dense, high-utility feel.
- **Vertical Rhythm:** Components are stacked using 16px (md) or 24px (lg) gaps to maintain a clear hierarchy between distinct security tasks.
- **Safe Areas:** Strict adherence to bottom-sheet safe zones for primary authorization buttons to ensure they are reachable with a thumb.

## Elevation & Depth

Hierarchy is established through **Tonal Layers** rather than heavy shadows. In this dark-themed design system, elevation is communicated by increasing the lightness of the surface color.

1. **Level 0 (Base):** #121212 - The main application canvas.
2. **Level 1 (Cards/Containers):** #1E1E1E - Used for device health cards and challenge modules.
3. **Level 2 (In-app Overlays):** #2C2C2C - Used for biometric prompts and modal dialogs.

**Borders:** Use low-contrast 1px solid borders (#333333) to define card boundaries. When an element is "Active" or "Awaiting Input," the border should transition to a 1px Cyber Lime or Primary Blue glow.

## Shapes

The shape language is **Soft but Structured**. We use a 4px (Soft) radius for most UI components to maintain a professional, slightly industrial aesthetic. 

- **Primary Buttons:** 8px (rounded-lg) for a substantial, tactile feel.
- **Input Fields:** 4px (default) to keep the "form" feel professional and disciplined.
- **Status Pills:** Fully rounded (pill) to distinguish status indicators from interactive buttons.

## Components

### Buttons
- **Primary Authorization:** Large, 56px height, Deep Security Blue background with white text. On "Success," the button transitions to Cyber Lime with black text.
- **Ghost Actions:** 1px border with no fill, used for secondary options like "View Details."

### Security Status Cards
Cards should include a **Geist Mono** label at the top right indicating status (e.g., "ENCRYPTED" or "VERIFIED"). Use a subtle inner-glow when the device health is optimal.

### Biometric Prompts
Fingerprint and FaceID icons should be centered within a circular "scanning" ring. This ring uses a dashed border that rotates during the "Challenge" phase.

### Cryptographic Progress Indicators
Instead of standard loaders, use a "Hex-Stream." A series of 8-10 hexadecimal characters that flicker rapidly as data is being signed, stopping only when the process is complete.

### Input Fields
Darker than the surface (#121212) with a 1px border. The cursor and focus state must use the Primary Blue to indicate an "active secure session."

### Challenges & Signatures
Visualized using an abstract **SVG Sine Wave**. When a signature is requested, the wave is erratic (Primary Blue); once signed, it flattens into a steady, rhythmic pulse (Cyber Lime).