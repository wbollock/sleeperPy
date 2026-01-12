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

## Premium Tier ($5/month) - Feature Planning

### Philosophy
- Free tier remains fully functional with core features
- Premium adds value through AI insights, advanced analytics, and convenience features
- Keep barrier to entry low - free users get the full tier analysis experience
- Premium justifies cost with features that save time and improve decision-making

### Authentication & Billing
**Technical Approach**:
- Use Stripe for payment processing (simple, well-documented)
- Consider alternatives: Paddle, LemonSqueezy (good for indie devs)
- Auth options:
  - Sleeper OAuth (ideal - no password needed, leverages existing Sleeper account)
  - Email magic links (passwordless, simple)
  - Traditional email/password (fallback)
- Store minimal user data: email, subscription status, Sleeper username mapping
- Use database: SQLite for simplicity, or PostgreSQL for production scale

### Premium Features

#### 1. AI-Powered Assistant (Ollama/OpenAI/Anthropic)
**Goal**: Provide intelligent, context-aware fantasy football advice

**Features**:
- **Trade Analyzer**: "Should I trade Tyreek Hill for two first-round picks?"
  - AI considers player values, team needs, league settings, your strategy
  - Provides reasoning and alternative suggestions

- **Lineup Optimizer with AI**: Beyond tier-based recommendations
  - "Who should I start this week?" with matchup-specific reasoning
  - Considers weather, injuries, recent performance, opponent defense
  - Natural language explanations

- **Waiver Wire Assistant**: "Who should I pick up this week?"
  - Prioritizes pickups based on your roster needs
  - Considers upcoming schedules, bye weeks, injury situations

- **Weekly Matchup Analysis**:
  - "How do I beat my opponent this week?"
  - Identifies key matchups and potential swing players

- **Player Outlook & Trends**:
  - "What's the outlook for Bijan Robinson ROS (rest of season)?"
  - AI synthesizes news, stats, schedule into actionable insights

**Technical Implementation**:
- Primary: Ollama (free, self-hosted, privacy-friendly)
  - Use llama3.2 or similar models
  - Host locally or on VPS
- Fallback: OpenAI API (for users who want faster responses)
- Alternative: Anthropic Claude API (high quality reasoning)
- Rate limiting: 20 AI queries per day for premium users
- Context: Feed AI with player stats, tiers, roster data, recent news

#### 2. Advanced Analytics & Insights
**Goal**: Deep data analysis beyond basic tier display

**Features**:
- **Historical Performance Tracking**:
  - Week-by-week player performance charts
  - Track your team's tier averages over the season
  - Compare actual points vs tier predictions

- **Tier Movement Alerts**:
  - Get notified when your players move tiers
  - Track tier volatility to identify boom/bust players

- **League Analytics Dashboard**:
  - Strength of schedule analysis
  - Playoff probability calculator
  - Power rankings with trend lines
  - Positional scarcity analysis

- **Dynasty-Specific Premium Analytics**:
  - Multi-year projection models
  - Dynasty value trend charts
  - Optimal rebuild timeline calculator
  - Draft pick value calculator with custom league scoring

- **Trade History & Performance**:
  - Track all your trades across seasons
  - "Did this trade work out?" retrospective analysis
  - Trade grade predictions

#### 3. Smart Notifications & Alerts
**Goal**: Never miss important fantasy events

**Features**:
- **Email Alerts**:
  - Injury updates for your players
  - Tier movements (when enabled)
  - High-value waiver wire drops
  - Trade deadline reminders

- **Browser Push Notifications** (opt-in):
  - Real-time breaking news for your players
  - Lineup lock reminders

- **Discord/Slack Integration**:
  - Post daily lineup recommendations to your league channel
  - Automated trade analysis in your league chat

#### 4. Multi-Account Management
**Goal**: Manage multiple leagues/accounts seamlessly

**Features**:
- Link multiple Sleeper usernames to one premium account
- Dashboard view of all your teams
- Cross-league analytics:
  - "Which of my teams is strongest?"
  - Portfolio diversification analysis
  - "Am I too invested in this player?"
- Quick switch between accounts without re-entering username

#### 5. Export & Sharing
**Goal**: Use data outside the app and share with league mates

**Features**:
- **Export to CSV/Excel**:
  - Full roster with tiers and dynasty values
  - Trade analysis reports
  - Season performance data

- **Printable Cheat Sheets**:
  - PDF export of your tiers for draft day
  - Custom formatting options

- **Shareable Reports**:
  - Generate read-only links to share league analysis
  - "Look at my power rankings" shareable cards
  - Social media-friendly graphics

#### 6. Premium Dynasty Toolkit
**Goal**: Enhanced dynasty league management

**Features**:
- **AI Rebuild vs Contend Advisor**:
  - "Should I rebuild or go for it?"
  - AI analyzes roster age, value, draft capital
  - Provides multi-year strategy roadmap

- **Draft Pick Trade Calculator**:
  - "Is this pick worth more than this player?"
  - Dynamic value based on draft position predictions

- **Prospect Deep Dives**:
  - Detailed scouting reports on rookies (AI-generated)
  - College stat analysis and projections
  - Landing spot impact analysis

- **Contract Year Tracking**:
  - For keeper leagues with contracts
  - Alerts for players in final year
  - Extension value recommendations

#### 7. Early Access & Priority Support
**Goal**: Make premium users feel valued

**Features**:
- Beta access to new features 2-4 weeks early
- Priority email support (24-hour response time)
- Vote on feature roadmap
- Exclusive Discord channel for premium users

### Pricing Strategy

**Free Tier** (Always Free):
- Core tier-based analysis
- Weekly lineup recommendations
- Free agent suggestions
- Basic dynasty features
- Cookie-based username persistence
- All current features remain free

**Premium Tier** ($5/month or $50/year):
- All free features +
- 20 AI queries per day
- Advanced analytics dashboard
- Email/push notifications
- Multi-account management
- Export to CSV/PDF
- Enhanced dynasty toolkit
- Early access to features
- Priority support

**Future Premium+ Tier** ($10/month) - Ideas for later:
- Unlimited AI queries
- Custom AI training on your league history
- White-label league websites
- Advanced API access
- League commissioner tools

### Implementation Phases

**Phase 1: Authentication Foundation**
1. Add user accounts (email-based or Sleeper OAuth)
2. Implement Stripe subscription management
3. Add subscription status checks
4. Create basic account dashboard

**Phase 2: AI Features**
1. Set up Ollama backend
2. Implement trade analyzer
3. Add lineup optimizer
4. Create AI chat interface

**Phase 3: Analytics & Alerts**
1. Historical data tracking
2. Email notification system
3. Tier movement alerts
4. Analytics dashboard

**Phase 4: Premium Polish**
1. Multi-account management
2. Export features
3. Early access program
4. Premium support system

### Technical Considerations

**Cost Analysis**:
- Ollama hosting: $10-20/month (VPS) or free (self-hosted)
- Database hosting: $5-15/month (managed PostgreSQL)
- Email service: $0-10/month (SendGrid free tier, then paid)
- Stripe fees: 2.9% + $0.30 per transaction
- Break-even: ~5-10 paid users covers infrastructure
- Profitable: 20+ users = $70-100/month profit

**Data Privacy**:
- Only store necessary user data
- No selling of user data (ever)
- Clear privacy policy
- GDPR compliance for EU users
- Option to delete account and all data

**Feature Flags**:
- Use feature flags to enable/disable premium features
- A/B testing for pricing and features
- Gradual rollout of new features

**Downgrade Grace Period**:
- If subscription lapses, keep data for 30 days
- Allow re-activation without data loss
- Soft paywall (remind but don't block immediately)

## Free vs Premium Decision Matrix

| Feature | Free | Premium |
|---------|------|---------|
| Tier-based lineup recommendations | ✅ | ✅ |
| Free agent suggestions | ✅ | ✅ |
| Dynasty values & toolkit | ✅ | ✅ Enhanced |
| Boris Chen tier integration | ✅ | ✅ |
| Cookie-based username memory | ✅ | ✅ |
| AI trade analyzer | ❌ | ✅ 20/day |
| AI lineup optimizer | ❌ | ✅ |
| Historical performance tracking | ❌ | ✅ |
| Tier movement alerts | ❌ | ✅ |
| Email notifications | ❌ | ✅ |
| Multi-account management | ❌ | ✅ |
| Export to CSV/PDF | ❌ | ✅ |
| Advanced analytics dashboard | ❌ | ✅ |
| Priority support | ❌ | ✅ |
| Early access to features | ❌ | ✅ |

## Future Enhancements
- Remember which league was last viewed
- Quick switch between multiple saved usernames
- Share league view with friends (read-only link)
- Export league data to CSV
- Browser notifications for tier changes
