# SleeperPy Improvements Tracker

## Performance Optimization

### Identified Issues
1. **Multiple API calls per league**
   - Current: Separate calls for rosters, matchups, transactions, traded_picks
   - Impact: Slow page loads for users with multiple leagues
   - Solution: Consider parallel fetching or caching improvements

2. **Dynasty values fetched every request**
   - Current: 24h cache, but still fetches on every page load if expired
   - Impact: Slow first load of the day
   - Solution: Background refresh? Longer TTL?

3. **Template rendering for large rosters**
   - Current: Renders all free agents, all transactions
   - Impact: Large HTML payloads
   - Solution: Pagination or lazy loading?

4. **No CDN for static assets**
   - Current: Serving CSS/JS from same server
   - Impact: Slower page loads
   - Solution: Consider CDN or asset optimization

### Quick Wins
- [ ] Add gzip compression for responses
- [ ] Minify CSS/JS
- [ ] Reduce default free agent limit (currently 30, maybe 20?)
- [ ] Add loading states to improve perceived performance

## Bug Hunting

### Areas to Test
1. **Edge cases for multi-league users**
   - What if user has 10+ leagues?
   - Mixed redraft and dynasty leagues
   - Leagues from different years

2. **Empty state handling**
   - No draft picks
   - No free agents available
   - No matchup data (offseason)
   - No dynasty values for players

3. **Mobile responsiveness**
   - Test all toolkit cards on phone
   - Test tables on small screens
   - Test dynasty mode toggle

4. **Premium feature display**
   - Verify premium features only show for premium users
   - Test graceful degradation for non-premium

### Known Edge Cases (Need Testing)
- [ ] Superflex leagues with no QB on roster
- [ ] Leagues with unusual roster settings (2QB, no TE, etc.)
- [ ] Players with special characters in names
- [ ] Trades involving draft picks only (no players)

## UI/UX Polish

### Visual Improvements
1. **Loading states**
   - Add skeleton screens while loading
   - Show progress for multi-league loads
   - Better error messages with retry buttons

2. **Color consistency**
   - Audit all color usage
   - Ensure dark/light theme works everywhere
   - Use CSS variables consistently

3. **Spacing and typography**
   - Review all card padding/margins
   - Ensure consistent font sizes
   - Better visual hierarchy

4. **Mobile UX**
   - Larger tap targets
   - Better collapsing behavior
   - Swipe gestures?

### Interaction Improvements
- [ ] Remember collapsed/expanded state per card
- [ ] Add "scroll to top" button
- [ ] Improve table sorting UX
- [ ] Add keyboard shortcuts for power users

## Documentation

### Code Documentation Needed
1. **Function comments**
   - Add godoc comments to all exported functions
   - Document complex algorithms (draft pick logic, trade fairness, etc.)
   - Add examples for key functions

2. **Architecture documentation**
   - Document caching strategy
   - Explain API integration patterns
   - Document premium feature flow

3. **Setup/deployment docs**
   - Environment variables
   - Systemd service setup
   - Prometheus integration
   - Backup/restore procedures

### User Documentation
- [ ] Getting started guide
- [ ] FAQ updates (explain premium features)
- [ ] Dynasty toolkit explainer
- [ ] Video walkthrough?

## Priority Order (User-Driven)
1. Performance optimization
2. Bug hunting
3. UI/UX polish
4. Documentation

---

## Session Notes
- Focus on quick wins first
- Test thoroughly before committing
- Document findings as we go
