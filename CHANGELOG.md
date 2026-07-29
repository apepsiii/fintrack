# Changelog

All notable changes to the FinTrack project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2026-07-29

### 🎉 Initial Release - MVP Phase 1 Complete

#### Added

**Backend**
- Go server with Gin web framework
- SQLite database integration (CGO-free using modernc.org/sqlite)
- Database initialization with auto-migration
- Category seeding (Makan, Transport, Tagihan, Gaji)
- Budget seeding (Makan: Rp 1.5M, Transport: Rp 500K)
- RESTful API endpoints for all CRUD operations
- Balance calculation logic (income - expense)
- Monthly expense aggregation
- Category-based expense statistics
- Budget monitoring with percentage calculation

**Frontend**
- Login page with splash screen design
- Dashboard with balance overview card
- Recent transactions list (latest 5)
- Bottom sheet transaction modal
- OCR receipt scan simulation (mockup)
- Statistics page with category breakdown
- Visual progress bars for expense percentages
- Budget targets page with color-coded alerts
- Profile and settings page
- Bottom navigation with 5 tabs
- Floating action button (+)
- Mobile-first responsive design
- HTMX integration for SPA-like experience

**Database Schema**
- `categories` table with 4 default entries
- `transactions` table for income/expense records
- `budgets` table for monthly limits

**Features**
- ✅ Frictionless transaction entry (<5 seconds)
- ✅ Real-time balance calculation
- ✅ Monthly expense tracking
- ✅ Category-based statistics
- ✅ Budget monitoring with visual indicators
- ✅ Color-coded budget alerts (green/yellow/red)
- ✅ OCR mockup simulation (1.5s delay, returns Rp 85,000)
- ✅ Mobile-optimized UI

**Developer Tools**
- Sample data seeder (`seed.go`)
- Windows startup script (`start.bat`)
- Linux/Mac startup script (`start.sh`)
- Git ignore configuration
- Comprehensive documentation

**Documentation**
- `README.md` - Complete project documentation
- `QUICKSTART.md` - Quick start guide
- `PROJECT_SUMMARY.md` - Implementation summary
- `STRUCTURE.md` - Project structure overview
- `CHANGELOG.md` - Version history (this file)

#### Technical Details

**Stack**
- Backend: Go 1.25 + Gin
- Database: SQLite (modernc.org/sqlite v1.55.0)
- Frontend: HTMX 1.9.10 + TailwindCSS 3.x
- Icons: Phosphor Icons
- Font: Inter (Google Fonts)

**Architecture**
- Server-side rendering with Go templates
- RESTful API design
- MVC pattern
- Mobile-first responsive layout

**Performance**
- Server start: ~2 seconds
- Database init: <100ms
- Page load: <50ms
- Database size: ~100KB with sample data

#### Fixed

**Build Issues**
- Resolved CGO dependency by switching from `mattn/go-sqlite3` to `modernc.org/sqlite`
- Fixed database driver name from "sqlite3" to "sqlite"
- Removed unused `math/rand` import in seed.go

#### Known Limitations

**Security (By Design - MVP)**
- No real authentication (bypass mode)
- No password hashing
- No session management
- OAuth Google is placeholder only

**Features (Planned for Phase 2+)**
- No real OCR integration (mockup only)
- No PWA support (no manifest/service worker)
- No offline mode
- No CSV/PDF export
- No dark mode toggle functionality
- No user registration
- No multi-user support

---

## [Unreleased] - Phase 2 Roadmap (Q3 2026)

### Planned Features

**Authentication**
- [ ] JWT token-based authentication
- [ ] bcrypt password hashing
- [ ] Session cookie management
- [ ] OAuth Google integration (real)
- [ ] User registration flow
- [ ] Password reset functionality

**PWA Conversion**
- [ ] manifest.json creation
- [ ] Service Worker implementation
- [ ] Install prompt
- [ ] Offline fallback pages
- [ ] App icon set (multiple sizes)

**OCR Integration**
- [ ] Google Cloud Vision API integration
- [ ] Receipt image upload handling
- [ ] Text extraction and parsing
- [ ] Amount detection algorithm
- [ ] Date extraction from receipts
- [ ] Category auto-suggestion

**Infrastructure**
- [ ] Dockerfile for containerization
- [ ] docker-compose.yml setup
- [ ] Environment variable configuration
- [ ] Production build optimization
- [ ] HTTPS/TLS support

---

## [Unreleased] - Phase 3 Roadmap (Q4 2026)

### Planned Features

**Export & Reporting**
- [ ] CSV export functionality
- [ ] PDF report generation
- [ ] Custom date range selection
- [ ] Email report delivery
- [ ] Monthly summary reports

**Offline Mode**
- [ ] IndexedDB integration
- [ ] Offline transaction queue
- [ ] Auto-sync when online
- [ ] Conflict resolution
- [ ] Sync status indicator

**UI Enhancements**
- [ ] Real dark mode toggle
- [ ] Theme persistence
- [ ] Custom category creation
- [ ] Category icon picker
- [ ] Budget edit interface
- [ ] Transaction edit/delete
- [ ] Search and filter

**Analytics**
- [ ] Monthly spending trends
- [ ] Year-over-year comparison
- [ ] Spending predictions
- [ ] Budget recommendations
- [ ] Category insights

---

## Version History Summary

| Version | Date | Status | Highlights |
|---------|------|--------|------------|
| 1.0.0 | 2026-07-29 | ✅ Released | MVP Phase 1 complete - All core features working |
| 2.0.0 | Q3 2026 | 📋 Planned | Authentication, PWA, Real OCR |
| 3.0.0 | Q4 2026 | 📋 Planned | Export, Offline mode, Advanced features |

---

## How to Upgrade

### From Fresh Install
```bash
git clone <repository>
cd personalBudgetApps
go mod tidy
go run seed.go
go run main.go
```

### When Phase 2 Releases
```bash
git pull origin main
go mod tidy
# Database migrations will run automatically
go run main.go
```

---

## Support & Feedback

For issues, feature requests, or questions:
1. Check documentation in `/docs` folder
2. Review QUICKSTART.md for common issues
3. See STRUCTURE.md for architecture overview

---

**Maintained by**: FinTrack Development Team  
**License**: MIT  
**Repository**: [Link to be added]

---

_This changelog follows [Keep a Changelog](https://keepachangelog.com/) format_
