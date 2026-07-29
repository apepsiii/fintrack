# 🧪 FinTrack Testing Guide

## ✅ Pre-Testing Checklist

- [x] Server is running at http://localhost:8080
- [x] Database has been seeded with sample data
- [x] All templates are in place
- [x] No compilation errors

## 🌐 Browser Testing

### Test 1: Login Page
**URL**: http://localhost:8080/login

**Expected Results**:
- ✅ Lime green splash screen with FinTrack logo
- ✅ Animated floating wallet icon
- ✅ Email and password input fields
- ✅ "Masuk ke Akun" button (dark green)
- ✅ "Masuk dengan Google" button with Google logo
- ✅ "Belum punya akun? Daftar sekarang" link

**Actions to Test**:
1. Click "Masuk ke Akun" → Should redirect to Dashboard
2. Verify mobile responsiveness (resize browser)

---

### Test 2: Dashboard (Main Page)
**URL**: http://localhost:8080/

**Expected Results**:
- ✅ Balance card showing: **Rp 3.660.000**
- ✅ Monthly expense showing: **Rp 1.850.000**
- ✅ User avatar and name "ST Racson"
- ✅ Notification bell icon with red badge
- ✅ Four quick action buttons
- ✅ "Transaksi Terakhir" section with 5 recent transactions
- ✅ Bottom navigation (5 tabs)
- ✅ Floating (+) button (lime green)

**Actions to Test**:
1. Scroll through transaction list
2. Click (+) button → Bottom sheet modal should appear
3. Click bottom navigation tabs → Should navigate to other pages
4. Verify "Beranda" tab is highlighted (white, filled icon)

---

### Test 3: Transaction Modal
**Trigger**: Click the green (+) button on Dashboard

**Expected Results**:
- ✅ Bottom sheet slides up smoothly
- ✅ Gray handle bar at top
- ✅ "Catat Transaksi Baru" title
- ✅ Amount input field with "Rp" prefix
- ✅ Scan button (camera icon)
- ✅ Category dropdown with 4 options
- ✅ Two submit buttons: Red (Pengeluaran) and Green (Pemasukan)
- ✅ Backdrop blur effect

**Actions to Test**:

**A. Manual Entry**
1. Enter amount: `50000`
2. Select category: "🍔 Makan & Minum"
3. Click "Pengeluaran" (red button)
4. Modal should close
5. Balance should update to: **Rp 3.610.000**
6. New transaction appears at top of list

**B. OCR Simulation**
1. Click (+) button again
2. Click "Scan" icon button
3. Select any image file from computer
4. Wait 1.5 seconds
5. Amount field should auto-fill with: **85000**
6. Green "✓ Hasil Scan" badge appears
7. Category dropdown still selectable
8. Can click "Scan" again to retry
9. Submit transaction

**C. Close Modal**
1. Click backdrop (outside modal)
2. Modal should slide down smoothly

---

### Test 4: Statistics Page
**URL**: http://localhost:8080/stats

**Expected Results**:
- ✅ "Analisis Laporan" header
- ✅ Calendar icon button (top right)
- ✅ Gray card showing total: **Rp 1.850.000** (red)
- ✅ "Sesuai jalur (On Track)" badge (lime green)
- ✅ "Rincian Kategori" section
- ✅ Three expense categories with icons:
  - Makan: Rp 835.000 (45% progress bar)
  - Tagihan: Rp 650.000 (35% progress bar)
  - Transport: Rp 365.000 (20% progress bar)
- ✅ Dark progress bars showing correct widths
- ✅ "Statistik" tab highlighted in bottom nav

**Actions to Test**:
1. Verify percentage calculations are correct
2. Check progress bar widths match percentages
3. Hover over category items (should have subtle hover effect)

---

### Test 5: Budget Targets Page
**URL**: http://localhost:8080/targets

**Expected Results**:
- ✅ "Anggaran Bulanan" header
- ✅ (+) button to add new budget (top right)
- ✅ Dark card showing budget summary:
  - Total spent: **Rp 1.200.000**
  - Total budget: **Rp 2.000.000**
  - Progress bar at ~60%
  - Green status message
- ✅ "Rincian Anggaran" section
- ✅ Two budget items:
  
  **Makan**
  - Icon: Hamburger
  - Spent: Rp 835.000 / Rp 1.500.000
  - Progress: ~55% (GREEN bar)
  
  **Transport**
  - Icon: Car
  - Spent: Rp 365.000 / Rp 500.000
  - Progress: ~73% (GREEN bar)

- ✅ "Target" tab highlighted in bottom nav

**Actions to Test**:
1. Verify progress bars are correct colors:
   - Green: < 80%
   - Yellow: 80-99%
   - Red: ≥ 100%
2. Check "Sisa" (remaining) calculations

---

### Test 6: Profile Page
**URL**: http://localhost:8080/profile

**Expected Results**:
- ✅ "Profil Saya" header
- ✅ Dark card with user info:
  - Avatar with "ST Racson" initials
  - Name: "ST Racson"
  - Email: "racson@fintrack.id"
  - Join date: "Bergabung sejak Jan 2026"
  - Edit pencil icon (top right of card)
- ✅ "PENGATURAN AKUN" section:
  - Data Pribadi option
  - Notifikasi option
- ✅ "APLIKASI" section:
  - Mode Gelap toggle (OFF state)
  - Ekspor Data (CSV) option
- ✅ Red "Keluar" button at bottom
- ✅ "Profil" tab highlighted in bottom nav

**Actions to Test**:
1. Click options → Currently placeholder, no action
2. Verify toggle switch is in OFF position
3. Check hover effects on buttons

---

## 📱 Mobile Responsiveness Testing

### Chrome DevTools Method

1. Open Chrome DevTools (F12)
2. Click device toolbar icon (Ctrl+Shift+M)
3. Select device from dropdown:
   - iPhone 12 Pro
   - Samsung Galaxy S20
   - iPad Air

### Test Points:

**Portrait Mode (375px)**
- [ ] All content fits within viewport width
- [ ] No horizontal scrolling
- [ ] Bottom navigation accessible
- [ ] Floating (+) button visible
- [ ] Cards have proper padding
- [ ] Text is readable (min 14px)

**Landscape Mode**
- [ ] Content adapts appropriately
- [ ] Bottom navigation still accessible
- [ ] Modal remains centered

---

## 🎯 Functional Testing

### Add Income Transaction
1. Go to Dashboard
2. Click (+) button
3. Enter: `1000000`
4. Select: "💰 Gaji & Pendapatan"
5. Click "Pemasukan" (green)
6. **Verify**: Balance increases by Rp 1.000.000
7. **Verify**: New transaction appears with green (+) prefix

### Add Expense Transaction
1. Click (+) button
2. Enter: `200000`
3. Select: "🍔 Makan & Minum"
4. Click "Pengeluaran" (red)
5. **Verify**: Balance decreases by Rp 200.000
6. **Verify**: New transaction appears with red (-) prefix
7. Go to Statistics page
8. **Verify**: Makan amount increased
9. Go to Targets page
10. **Verify**: Makan progress bar increased

### Test OCR Simulation
1. Dashboard → Click (+)
2. Click "Scan" icon
3. Upload any image (e.g., screenshot)
4. Wait for animation
5. **Verify**: Amount = 85000
6. **Verify**: "✓ Hasil Scan" badge appears
7. **Verify**: Can still change category
8. **Verify**: Can click scan again to retry

---

## 🧭 Navigation Testing

### Bottom Navigation Flow
1. Start at Dashboard (Home icon filled)
2. Click "Statistik" → Page loads, icon fills
3. Click "Target" → Page loads, icon fills
4. Click "Profil" → Page loads, icon fills
5. Click "Beranda" → Back to dashboard

**Expected Behavior**:
- ✅ Smooth page transitions (HTMX boost)
- ✅ Active tab has white color + filled icon
- ✅ Inactive tabs have white/50 opacity
- ✅ URL changes in address bar
- ✅ No full page reload (SPA-like)

### Floating (+) Button
- [ ] Visible on all main pages
- [ ] Always at same position (bottom center, elevated)
- [ ] Opens modal on click
- [ ] Has lime green color with shadow

---

## 🎨 Visual/UI Testing

### Brand Colors Check
- [ ] Primary: #c3f545 (lime green) - FAB, accents, badges
- [ ] Dark: #01381b (forest green) - navigation, headers
- [ ] Background: #f1f5f9 (light gray) - page background
- [ ] Cards: #ffffff (white) - content cards

### Typography
- [ ] Font: Inter (Google Fonts)
- [ ] Headings are bold (700-800 weight)
- [ ] Body text is medium (500 weight)
- [ ] Small text is readable (min 11px)

### Spacing & Layout
- [ ] Consistent padding (px-6 = 1.5rem)
- [ ] Cards have rounded corners (rounded-3xl = 1.5rem)
- [ ] Proper gap between elements
- [ ] Bottom navigation has safe area padding

### Animations
- [ ] Modal slides up/down smoothly (300ms)
- [ ] Backdrop fades in/out (300ms)
- [ ] Progress bars animate (1000ms ease-out)
- [ ] Wallet icon floats (3s loop)
- [ ] Button active states (scale-95)

---

## 🔍 Data Validation Testing

### Current Database State
After seeding, verify these values:

**Dashboard**
- Balance: Rp 3.660.000
- Monthly Expense: Rp 1.850.000
- Transaction count: 5 displayed (most recent)

**Statistics**
- Total Expense: Rp 1.850.000
- Makan: Rp 835.000 (45%)
- Tagihan: Rp 650.000 (35%)
- Transport: Rp 365.000 (20%)

**Targets**
- Makan: 835K / 1.5M = 55% (GREEN)
- Transport: 365K / 500K = 73% (GREEN)

### Add Transaction and Verify Cascade
1. Add expense: Makan Rp 100.000
2. **Check Dashboard**: Balance decreased
3. **Check Statistics**: Makan total increased
4. **Check Targets**: Makan percentage increased

---

## ⚡ Performance Testing

### Load Time
- [ ] Initial page load < 100ms
- [ ] Template rendering < 50ms
- [ ] Database query < 10ms
- [ ] OCR simulation = 1500ms (intentional)

### Browser Console
- [ ] No JavaScript errors
- [ ] No CSS warnings
- [ ] No 404 network errors
- [ ] HTMX requests complete successfully

---

## 🐛 Bug Testing

### Edge Cases to Test

**Empty Amount**
1. Open modal, leave amount empty
2. Try to submit
3. **Expected**: HTML5 validation prevents submit

**Zero Amount**
1. Enter "0" as amount
2. Submit transaction
3. **Expected**: Transaction saved (no validation yet)

**Negative Amount**
1. Try entering "-100"
2. **Expected**: Number input may allow it (future: add validation)

**Very Large Amount**
1. Enter "999999999999"
2. Submit
3. **Expected**: Should save (integer overflow possible - future fix)

**Rapid Clicks**
1. Click submit button multiple times rapidly
2. **Expected**: May create duplicate transactions (future: add debounce)

**Browser Back Button**
1. Navigate through pages
2. Click browser back
3. **Expected**: HTMX boost should handle it

---

## ✅ Testing Completion Checklist

### Basic Functionality
- [ ] Server starts without errors
- [ ] All 5 pages load successfully
- [ ] No console errors in browser
- [ ] Database operations work

### Core Features
- [ ] Login page displays correctly
- [ ] Dashboard shows accurate balance
- [ ] Can add income transaction
- [ ] Can add expense transaction
- [ ] OCR simulation works
- [ ] Statistics page calculates correctly
- [ ] Budget progress bars show correct percentages
- [ ] Profile page displays user info

### Navigation
- [ ] Bottom navigation works
- [ ] Active tab is highlighted
- [ ] Floating (+) button opens modal
- [ ] Modal closes on backdrop click
- [ ] Page transitions are smooth

### Visual/UX
- [ ] Mobile responsive (375px - 768px)
- [ ] Brand colors applied correctly
- [ ] Icons display properly (Phosphor)
- [ ] Animations work smoothly
- [ ] Hover states visible

### Data Integrity
- [ ] Balance calculation is correct
- [ ] Monthly expense aggregation accurate
- [ ] Category statistics match transactions
- [ ] Budget percentages calculated correctly
- [ ] Transactions saved to database
- [ ] Recent transactions list updates

---

## 🎉 Test Results Summary

**Date**: 2026-07-29  
**Tester**: _______________  
**Browser**: _______________  
**Device**: _______________  

**Overall Status**: [ ] PASS  [ ] FAIL  [ ] PARTIAL

**Notes**:
_______________________________________________________
_______________________________________________________
_______________________________________________________

**Issues Found**:
1. _________________________________________________
2. _________________________________________________
3. _________________________________________________

**Recommended Actions**:
_______________________________________________________
_______________________________________________________

---

**Next**: After all tests pass, the application is ready for user acceptance testing (UAT) and deployment preparation for Phase 2.
