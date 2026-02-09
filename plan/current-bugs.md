# Current Bugs - Priority List

## Critical Bugs (Fix ASAP)

### 1. Draft Capital Bug - Incorrect Ownership Display
**Status**: 🔴 NOT FIXED - Still Broken

**Problem**: Draft picks showing incorrect "from X" attribution
- User trades their pick TO team X
- UI shows "from X" (backwards)
- Should either:
  - Not show at all (pick was traded away)
  - OR show "to X" to indicate where it went

**Example**:
- User trades 2025 1st round pick TO gdyche
- UI displays: "2025 Round 1 (from gdyche)" ❌
- Should display: Nothing (don't show picks user doesn't own) ✅

**Root Cause**: Sleeper API field interpretation
- Need to verify meaning of `roster_id`, `owner_id`, `previous_owner_id`
- Current logic confuses ownership direction
- See: `plan/draft-picks-debugging.md` for debug guide

**Files Affected**:
- `main.go` lines 1188-1352 (draft picks logic)

**Fix Priority**: HIGH - Affects dynasty league users

---

### 2. Transactions Missing Team Names
**Status**: ✅ FIXED (commits: 597c363, 9c5c473)

**Problem**: Transaction display doesn't show which team made the move
- Waiver claims show: "Player X was claimed"
- Missing: WHO claimed the player
- Hard to track league activity without team context

**Current Display**:
```
📋 Waiver                          Sep 10, 2025
Claimed JuJu Smith-Schuster (dropped Andrei Iosivas)
```

**Should Display**:
```
📋 Waiver                          Sep 10, 2025
pauldhaugen claimed JuJu Smith-Schuster (dropped Andrei Iosivas)
```

**Impact**:
- Can't tell who's active on waivers
- Reduces value of transaction tracking feature

**Files Affected**:
- `main.go` - transaction aggregation logic
- `templates/tiers.html` - transaction display

**Fix Priority**: MEDIUM - Feature works but missing important context

---

### 3. Trades Section Not Populated
**Status**: ✅ FIXED (commit: 2aec378)

**Problem**: Recent trades section shows empty or doesn't populate
- Trade data exists in Sleeper API
- Not being fetched or parsed correctly
- Users can't see recent league trades

**Expected Behavior**:
- Show last 4 weeks of trades
- Display both sides of trade (who gave what)
- Include trade date/time

**Current Behavior**:
- Trades section empty OR not showing

**Debug Steps**:
1. Check if `/v1/league/{league_id}/transactions/{week}` returns trades
2. Verify transaction type filtering (type: "trade")
3. Check if trades are being filtered out incorrectly

**Files Affected**:
- `main.go` - `fetchRecentTransactions()` function
- Check transaction type parsing

**Fix Priority**: HIGH - Core dynasty feature not working

---

### 4. 2026 NFL Draft Prospects Contains 2025 Players
**Status**: ✅ FIXED (commit: a8abdfb)

**Problem**: 2026 draft prospects list has 2025 rookies mixed in
- Wrong draft class data
- Confuses users planning for 2026 rookie draft
- Reduces trust in dynasty toolkit accuracy

**Example Issues**:
- Shows players who were drafted in 2025 NFL Draft
- Should only show players eligible for 2026 NFL Draft
- Data source might be wrong or outdated

**Root Cause**:
- Hard-coded rookie data in wrong section
- OR pulling from incorrect data source
- Need to separate 2025 vs 2026 prospects

**Files Affected**:
- `main.go` - rookie prospect data (search for "2026" and "RookieProspect")
- Check where prospect data is defined

**Fix Priority**: MEDIUM - Wrong data is worse than no data

**Fix Approach**:
- Remove 2026 section entirely if data is unreliable
- OR manually curate accurate 2026 prospect list
- OR find reliable API/data source for 2026 class

---

## Product & Legal Features (Missing)

### 5. Privacy Policy Page
**Status**: ❌ Missing
**Priority**: CRITICAL (before any monetization)

**Requirements**:
- Clear statement of data collection (username via cookie)
- Explain how data is used
- Third-party services (Sleeper API, Boris Chen)
- No selling of user data
- User rights (request deletion, etc.)
- Cookie usage explanation
- GDPR compliance language
- Contact info for privacy inquiries

**Template Sources**:
- Use standard SaaS privacy policy template
- Customize for SleeperPy specifics
- Keep it simple and honest

**Route**: `/privacy`
**Link**: Footer

---

### 6. Terms of Service Page
**Status**: ❌ Missing
**Priority**: CRITICAL (before any monetization)

**Requirements**:
- User responsibilities
- Acceptable use policy
- Service availability (no guarantees)
- Data accuracy disclaimer (tiers are estimates)
- No fantasy football advice disclaimer
- Intellectual property (Boris Chen attribution)
- Termination rights
- Limitation of liability
- Changes to terms

**Route**: `/terms` or `/tos`
**Link**: Footer

---

### 7. About Page
**Status**: ❌ Missing
**Priority**: HIGH

**Content**:
- What is SleeperPy / purpose
- How it works (data sources)
- Who made it
- Why it exists
- Contact information
- Link to GitHub
- Attribution to Boris Chen, Sleeper, DynastyProcess

**Route**: `/about`
**Link**: Header/Footer

---

### 8. Help / FAQ Page
**Status**: ❌ Missing
**Priority**: HIGH

**Common Questions**:
- What are tiers and how do they work?
- How do I find my Sleeper username?
- How often do tiers update?
- What's the difference between Standard and PPR?
- What is dynasty mode?
- How are dynasty values calculated?
- What do the colored indicators mean?
- Why don't I see my league?
- How do I report a bug?

**Route**: `/help` or `/faq`
**Link**: Header/Footer

---

### 9. Cookie Consent Banner
**Status**: ❌ Missing
**Priority**: HIGH (GDPR/CCPA compliance)

**Requirements**:
- Notify users about cookie usage
- Explain what cookies are used for (username persistence)
- Accept/Decline options
- Link to Privacy Policy
- Only show once (store preference)
- Compliant with GDPR/CCPA

**Implementation**:
- Simple banner at bottom of page
- "We use cookies to remember your username. [Learn more](/privacy) [Accept] [Decline]"
- Store acceptance in cookie (ironic but legal)

---

### 10. Footer with Legal Links
**Status**: ⚠️ Partial (has footer but missing legal links)

**Required Links**:
- Privacy Policy
- Terms of Service
- About
- Help / FAQ
- Contact
- GitHub
- Version number
- Copyright notice

**Current Footer**: Minimal
**Needed Footer**: Professional with all legal links

---

### 11. Contact Page / Email
**Status**: ❌ Missing
**Priority**: MEDIUM

**Options**:
- **Option 1**: Simple mailto: link
  - `contact@sleeperpy.com` or your email
  - No form needed, just email

- **Option 2**: Contact form
  - Name, email, message fields
  - Send via SMTP or service (SendGrid, etc.)
  - More professional but more complex

- **Option 3**: GitHub Issues only
  - Direct users to GitHub for bug reports
  - Less formal but works for open source

**Recommendation**: Start with Option 1 (email) or Option 3 (GitHub)

---

### 12. SEO Meta Tags
**Status**: ❌ Missing
**Priority**: MEDIUM-HIGH

**Required Meta Tags**:
```html
<!-- Basic SEO -->
<title>SleeperPy - Fantasy Football Tier Analysis</title>
<meta name="description" content="Tier-based lineup recommendations for your Sleeper fantasy football leagues. See which players to start, bench, or pick up based on expert rankings.">
<meta name="keywords" content="fantasy football, sleeper, tiers, boris chen, lineup optimizer, dynasty">

<!-- Open Graph (Facebook, LinkedIn) -->
<meta property="og:title" content="SleeperPy - Fantasy Football Tier Analysis">
<meta property="og:description" content="Make smarter lineup decisions with tier-based player rankings.">
<meta property="og:type" content="website">
<meta property="og:url" content="https://yoursite.com">
<meta property="og:image" content="https://yoursite.com/og-image.png">

<!-- Twitter Card -->
<meta name="twitter:card" content="summary_large_image">
<meta name="twitter:title" content="SleeperPy - Fantasy Football Tier Analysis">
<meta name="twitter:description" content="Make smarter lineup decisions with tier-based player rankings.">
<meta name="twitter:image" content="https://yoursite.com/twitter-card.png">

<!-- Mobile -->
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<meta name="theme-color" content="#1a1f2e">
```

**Files Affected**: `templates/*.html` - add to `<head>` section

---

### 13. Favicon
**Status**: ❌ Missing
**Priority**: LOW (but easy to add)

**Requirements**:
- Simple icon representing app
- Multiple sizes (16x16, 32x32, 180x180, etc.)
- PNG format
- Simple design (football, chart, checkmark)

**Options**:
- Use emoji as favicon (quick & easy)
- Design simple icon (30 minutes)
- Use Figma/Canva for free design

---

### 14. Robots.txt & Sitemap
**Status**: ❌ NOT FIXED - Still Missing
**Priority**: LOW-MEDIUM (for SEO)

**robots.txt**:
```
User-agent: *
Allow: /
Disallow: /api/

Sitemap: https://yoursite.com/sitemap.xml
```

**sitemap.xml**:
```xml
<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url>
    <loc>https://yoursite.com/</loc>
    <priority>1.0</priority>
  </url>
  <url>
    <loc>https://yoursite.com/about</loc>
    <priority>0.8</priority>
  </url>
  <!-- Add more pages -->
</urlset>
```

---

## Testing & Validation Needed

After fixes:
- [ ] Test draft picks with multiple trade scenarios
- [ ] Verify all transaction types show team names
- [ ] Confirm trades populate correctly
- [ ] Validate 2026 prospects are accurate
- [ ] Review all legal pages with lawyer (optional but recommended)
- [ ] Test cookie consent banner on EU IP
- [ ] Verify SEO meta tags render correctly
- [ ] Check mobile responsiveness of new pages

---

## Priority Order for Fixes

### Sprint 1: Critical Bugs (Week 1)
1. ✅ Fix draft capital ownership bug (HIGH - breaks dynasty feature)
2. ✅ Fix trades not populating (HIGH - missing core feature)
3. ✅ Add team names to transactions (MEDIUM - UX improvement)
4. ✅ Fix/remove 2026 prospects data (MEDIUM - data accuracy)

### Sprint 2: Legal/Product (Week 2)
5. ✅ Add Privacy Policy page (CRITICAL for launch)
6. ✅ Add Terms of Service page (CRITICAL for launch)
7. ✅ Add cookie consent banner (HIGH for compliance)
8. ✅ Update footer with legal links (HIGH)

### Sprint 3: Content & SEO (Week 3)
9. ✅ Add About page (HIGH for credibility)
10. ✅ Add Help/FAQ page (MEDIUM for UX)
11. ✅ Add SEO meta tags (MEDIUM for growth)
12. ✅ Add favicon (LOW but easy)
13. ✅ Add robots.txt & sitemap (LOW)
14. ✅ Add contact method (MEDIUM)

---

## Notes

- Legal pages are REQUIRED before any monetization attempts
- Privacy policy is REQUIRED for GDPR compliance (EU users)
- Cookie consent is REQUIRED for EU users
- Terms of Service protects you legally
- All these features are "table stakes" for a professional web app
- Don't skip legal stuff - it's important even for free apps
