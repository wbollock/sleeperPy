# Product Experience Features

## Overview
Transform sleeperPy from a utility tool into a polished web product while maintaining its simplicity and ease of use.

## Core Philosophy
- No real authentication (username/password)
- Remember last successful username using cookies
- Seamless, product-like experience
- Simple sign out to switch accounts

## Features to Implement

### 1. Username Persistence (Cookie-based)
**Goal**: Remember the last successful Sleeper username and auto-populate it on return visits

**Technical Implementation**:
- Set HTTP cookie `sleeper_username` after successful lookup
- Cookie expiration: 30 days
- Cookie attributes: `HttpOnly=false, Secure=true (in prod), SameSite=Lax`
- Read cookie on page load and pre-populate username field

**User Experience**:
- First-time visitors: See normal login form
- Returning users: See "Welcome back, [username]" with pre-filled form
- One-click to view their tiers again
- Easy to switch to different username

### 2. Sign Out Functionality
**Goal**: Allow users to clear their saved username and start fresh

**Technical Implementation**:
- Add `/signout` endpoint that clears the `sleeper_username` cookie
- Redirect to home page after sign out
- Simple button in header/footer

**User Experience**:
- "Sign Out" button visible when username is saved
- Clicking signs out and returns to clean slate
- Can enter a different username

### 3. Welcome Back Experience
**Goal**: Make returning users feel recognized and streamline their workflow

**UI Changes**:
- When cookie exists: Show "Welcome back!" message
- Display last username with option to use it or change
- Quick action button: "View My Tiers"
- Secondary action: "Use Different Username"

### 4. Session/State Indicators
**Goal**: Help users understand their current state

**Visual Elements**:
- Show current username in header when logged in
- Visual distinction between "guest" and "returning user" states
- Clear indication of which account data is being displayed

### 5. Enhanced Error Handling
**Goal**: Better user feedback for common issues

**Improvements**:
- Friendly error messages
- Suggestions for next steps
- Clear distinction between user errors and system errors
- "Try Again" actions inline with errors

### 6. Loading States & Feedback
**Goal**: Keep users informed during data fetching

**Current State**: Basic "Loading..." text exists
**Enhancements**:
- Skeleton screens for league cards
- Progressive disclosure (show leagues as they load)
- Clear status messages ("Fetching your leagues...", "Loading tiers...")

### 7. Product Polish
**Goal**: Professional look and feel

**UI/UX Improvements**:
- Smooth transitions when switching between states
- Consistent spacing and typography
- Better mobile responsiveness
- Accessibility improvements (ARIA labels, keyboard navigation)
- Favicon and meta tags for sharing

### 8. User Preferences (Future)
**Goal**: Remember user settings beyond username

**Potential Features** (not implementing now, but documenting for future):
- Preferred league view (expanded/collapsed)
- Sort preferences
- Dark/light mode preference
- Default to specific league

## Implementation Priority

### Phase 1 (This PR)
1. Cookie-based username persistence
2. Sign out functionality
3. Welcome back UI
4. Enhanced header with current user

### Phase 2 (Future)
1. Better loading states
2. Error handling improvements
3. Accessibility audit
4. Mobile optimizations

## Technical Considerations

### Cookie Security
- Use `HttpOnly=false` so JavaScript can read for UI logic
- Use `Secure=true` in production (HTTPS only)
- Set `SameSite=Lax` to prevent CSRF while allowing normal navigation
- No sensitive data in cookies (only username)

### Backward Compatibility
- Existing users without cookies: Same experience as today
- No breaking changes to existing functionality
- Progressive enhancement approach

### Testing
- Test with cookies enabled/disabled
- Test cookie expiration
- Test sign out flow
- Test username switching
- Test with invalid/expired usernames in cookie

## Success Metrics
- User returns and sees their username remembered
- Sign out works correctly and clears state
- No degradation for first-time users
- Feels like a "real" web app, not a utility script

## Code Refactoring Needed

### Split main.go into Multiple Files
**Problem**: main.go is currently 2000+ lines and exceeds token limits for LLM tools

**Solution**: Split into logical modules:
- `main.go` - Entry point, routing, server setup
- `handlers.go` - HTTP handlers (indexHandler, lookupHandler, etc.)
- `sleeper.go` - Sleeper API client functions
- `tiers.go` - Boris Chen tier fetching and parsing
- `dynasty.go` - Dynasty-specific logic and value calculations
- `models.go` - Data structures (PlayerRow, LeagueData, etc.)
- `rendering.go` - Template rendering functions
- `middleware.go` - Logging, metrics, etc.

**Priority**: Medium (doesn't block features but improves maintainability)

## Future Enhancements
- Remember which league was last viewed
- Quick switch between multiple saved usernames
- Share league view with friends (read-only link)
- Export league data to CSV
- Browser notifications for tier changes
