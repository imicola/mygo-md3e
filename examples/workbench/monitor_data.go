package main

import (
	"fmt"
	"math"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/imicola/mygo-md3e/charts"
)

const (
	telemetryInterval = 500 * time.Millisecond
	scale10K          = 10_000
	scale100K         = 100_000

	modeConnections = 0
	modeDNS         = 1

	routeProxy   = 0
	routeDirect  = 1
	routeBlocked = 2

	dnsCached  = 0
	dnsDoH     = 1
	dnsBlocked = 2
)

var (
	baseDomains = []string{
		"api.github.com",
		"edge.cloudflare.net",
		"registry.npmjs.org",
		"storage.googleapis.com",
		"cdn.jsdelivr.net",
		"proxy.golang.org",
		"metrics.datadoghq.com",
		"auth.docker.io",
		"s3.us-west-2.amazonaws.com",
		"gateway.tailscale.com",
		"streams.twitch.tv",
		"crates.io",
		"pypi.org",
		"dns.google",
		"updates.signal.org",
		"internal.corp.lan",
		"ntp.ubuntu.com",
		"telemetry.adguard-dns.com",
		"ads.tracker.example",
		"beacon.metrics.invalid",
	}
	processNames = []string{
		"chrome",
		"go",
		"git",
		"dockerd",
		"code",
		"curl",
		"node",
		"tailscaled",
		"spotify",
		"systemd-resolved",
	}
	ruleLabels = []string{
		"DOMAIN-SUFFIX,github.com",
		"RULE-SET,cdn-global",
		"GEOIP,CN,DIRECT",
		"PROCESS-NAME,go",
		"IP-CIDR,10.0.0.0/8",
		"RULE-SET,reject",
		"DOMAIN-KEYWORD,google",
		"MATCH",
	}
	routeNames    = []string{"Proxy", "Direct", "Blocked"}
	protoNames    = []string{"TCP", "UDP", "QUIC"}
	dnsTypeNames  = []string{"A", "AAAA", "HTTPS", "TXT", "SRV", "MX"}
	dnsStatusName = []string{"Cached", "DoH", "Blocked"}
	dnsResolvers  = []string{
		"https://1.1.1.1/dns-query",
		"https://dns.google/dns-query",
		"quic://dns.adguard.com",
		"192.168.1.1:53",
		"fake-ip://pool",
	}

	processNamesLower  = lowerAll(processNames)
	ruleLabelsLower    = lowerAll(ruleLabels)
	routeNamesLower    = lowerAll(routeNames)
	dnsTypeNamesLower  = lowerAll(dnsTypeNames)
	dnsStatusNameLower = lowerAll(dnsStatusName)
	dnsResolversLower  = lowerAll(dnsResolvers)
)

func lowerAll(in []string) []string {
	out := make([]string, len(in))
	for i, s := range in {
		out[i] = strings.ToLower(s)
	}
	return out
}

type connRecord struct {
	id        uint64
	hostIdx   uint16
	procIdx   uint8
	ruleIdx   uint8
	route     uint8
	proto     uint8
	port      uint16
	latency10 uint16 // 0.1 ms precision
	downKB    uint32
	upKB      uint32
}

type dnsRecord struct {
	id          uint64
	hostIdx     uint16
	qtypeIdx    uint8
	resolverIdx uint8
	status      uint8
	ttlSec      uint16
	latency10   uint16
}

type memSnapshot struct {
	heapAlloc float64
	heapFree  float64
	stackSys  float64
	sysTotal  float64
	numGC     uint32
	objects   uint64

	heapText string
	sysText  string
	gcText   string
}

type filterSignature struct {
	mode        int
	scaleIdx    int
	query       string
	filterRoute [3]bool
	filterDNS   [3]bool
	sort        ui.SortOrder
	sortBy      []ui.SortOrder
}

type cellCacheEntry struct {
	valid bool
	mode  int
	id    uint64
	col   int
	text  string
}

type monitorState struct {
	running   bool
	lastTick  time.Time
	tickCount uint64
	rng       uint64

	routingMode   int // 0=Rule, 1=Global, 2=Direct
	tunEnabled    bool
	fakeIPEnabled bool

	downRing      *charts.RingBuffer
	upRing        *charts.RingBuffer
	downSpeedMBps float64
	upSpeedMBps   float64
	downPeakMBps  float64
	upPeakMBps    float64
	downSpeedText string
	upSpeedText   string
	downPeakText  string
	upPeakText    string

	activeConns int
	dnsQPS      int
	dnsHitRate  float64
	dnsHitText  string
	dnsQPSText  string

	mem memSnapshot

	tableMode int // 0=Connections, 1=DNS Queries
	scaleIdx  int // 0=10K, 1=100K
	query     string

	filterRoute [3]bool
	filterDNS   [3]bool

	connTableState ui.ListState
	dnsTableState  ui.ListState
	connSort       ui.SortOrder
	connSortBy     []ui.SortOrder
	dnsSort        ui.SortOrder
	dnsSortBy      []ui.SortOrder

	selectedRow int
	selectedID  uint64

	hostPool      []string
	hostPoolLower []string
	conns         []connRecord
	dns           []dnsRecord
	filtered      []int
	lastSig       filterSignature
	hasFilter     bool
	filterDirty   bool

	cellCache [64]cellCacheEntry
}

func newMonitorState() *monitorState {
	m := &monitorState{
		running:       true,
		rng:           0x9e3779b97f4a7c15,
		tunEnabled:    true,
		fakeIPEnabled: true,
		downRing:      charts.NewRingBuffer(72),
		upRing:        charts.NewRingBuffer(72),
		dnsHitRate:    0.884,
		dnsQPS:        342,
		selectedRow:   -1,
		connSort:      ui.SortOrder{Column: "Down", Descending: true},
		dnsSort:       ui.SortOrder{Column: "Latency", Descending: true},
		hostPool:      buildHostPool(),
	}
	m.hostPoolLower = lowerAll(m.hostPool)
	m.connSortBy = []ui.SortOrder{m.connSort}
	m.dnsSortBy = []ui.SortOrder{m.dnsSort}

	// Pre-fill 72 smooth baseline samples so initial render is immediately rich.
	for i := range 72 {
		phase := float64(i) * 0.22
		down := 62 + 34*math.Sin(phase) + 14*math.Cos(phase*1.7)
		up := 18 + 9*math.Cos(phase*0.85) + 4*math.Sin(phase*1.9)
		m.pushTrafficSample(down, up)
	}
	m.ensureScale(scale10K)
	m.activeConns = len(m.conns)
	m.refreshDNSLabels()
	m.sampleMemory()
	m.ensureFiltered()
	return m
}

func buildHostPool() []string {
	pool := make([]string, 0, len(baseDomains)*16)
	for i := range 16 {
		for _, base := range baseDomains {
			if i == 0 {
				pool = append(pool, base)
			} else {
				pool = append(pool, fmt.Sprintf("n%02d.%s", i, base))
			}
		}
	}
	return pool
}

func (m *monitorState) nextRand() uint64 {
	x := m.rng
	x ^= x << 13
	x ^= x >> 7
	x ^= x << 17
	m.rng = x
	return x
}

func (m *monitorState) pushTrafficSample(down, up float64) {
	if down < 0.2 {
		down = 0.2
	}
	if up < 0.1 {
		up = 0.1
	}
	m.downSpeedMBps = down
	m.upSpeedMBps = up
	if down > m.downPeakMBps {
		m.downPeakMBps = down
	}
	if up > m.upPeakMBps {
		m.upPeakMBps = up
	}
	m.downRing.Push(down)
	m.upRing.Push(up)
	m.downSpeedText = fmt.Sprintf("%.1f", m.downSpeedMBps)
	m.upSpeedText = fmt.Sprintf("%.1f", m.upSpeedMBps)
	m.downPeakText = fmt.Sprintf("Peak %.1f MB/s", m.downPeakMBps)
	m.upPeakText = fmt.Sprintf("Peak %.1f MB/s", m.upPeakMBps)
}

func (m *monitorState) refreshDNSLabels() {
	m.dnsHitText = fmt.Sprintf("%.1f%%", m.dnsHitRate*100)
	m.dnsQPSText = fmt.Sprintf("%d qps", m.dnsQPS)
}

func (m *monitorState) sampleMemory() {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	heapAlloc := float64(ms.HeapAlloc)
	var heapFree float64
	if ms.HeapInuse > ms.HeapAlloc {
		heapFree = float64(ms.HeapInuse - ms.HeapAlloc)
	}
	stackSys := float64(ms.StackInuse + ms.MSpanInuse + ms.MCacheInuse)
	sysTotal := float64(ms.Sys)
	if sysTotal < heapAlloc+heapFree+stackSys {
		sysTotal = heapAlloc + heapFree + stackSys
	}
	m.mem = memSnapshot{
		heapAlloc: heapAlloc,
		heapFree:  heapFree,
		stackSys:  stackSys,
		sysTotal:  sysTotal,
		numGC:     ms.NumGC,
		objects:   ms.HeapObjects,
		heapText:  formatBytesIEC(heapAlloc),
		sysText:   fmt.Sprintf("Sys %s · GC #%d", formatBytesIEC(sysTotal), ms.NumGC),
		gcText:    fmt.Sprintf("%d objs", ms.HeapObjects),
	}
}

func (m *monitorState) triggerGC() {
	runtime.GC()
	m.sampleMemory()
}

func (m *monitorState) targetCount() int {
	if m.scaleIdx == 1 {
		return scale100K
	}
	return scale10K
}

func (m *monitorState) ensureScale(target int) {
	if len(m.conns) < target {
		start := len(m.conns)
		grown := make([]connRecord, target)
		copy(grown, m.conns)
		rng := uint64(0x517cc1b727220a95) ^ uint64(start*2654435761+1)
		stepRand := func() uint64 {
			rng ^= rng << 13
			rng ^= rng >> 7
			rng ^= rng << 17
			return rng
		}
		for i := start; i < target; i++ {
			r1 := stepRand()
			r2 := stepRand()
			hostIdx := uint16(r1 % uint64(len(m.hostPool)))
			procIdx := uint8((r1 >> 16) % uint64(len(processNames)))
			ruleIdx := uint8((r1 >> 24) % uint64(len(ruleLabels)))
			route := uint8(routeProxy)
			if ruleIdx == 2 || ruleIdx == 4 {
				route = routeDirect
			} else if ruleIdx == 5 || int(hostIdx)%len(baseDomains) >= len(baseDomains)-2 {
				route = routeBlocked
			}
			proto := uint8((r2 >> 8) % uint64(len(protoNames)))
			port := uint16(443)
			switch (r2 >> 16) % 5 {
			case 1:
				port = 80
			case 2:
				port = 8443
			case 3:
				port = 53
			case 4:
				port = 8080
			}
			lat10 := uint16(80 + (r2 % 2400))
			if route == routeDirect {
				lat10 = uint16(12 + (r2 % 420))
			} else if route == routeBlocked {
				lat10 = uint16(1 + (r2 % 25))
			}
			downKB := uint32(16 + ((r1 >> 32) % 420_000))
			upKB := uint32(4 + ((r2 >> 24) % 96_000))
			if route == routeBlocked {
				downKB, upKB = 0, 0
			}
			grown[i] = connRecord{
				id:        uint64(i + 1),
				hostIdx:   hostIdx,
				procIdx:   procIdx,
				ruleIdx:   ruleIdx,
				route:     route,
				proto:     proto,
				port:      port,
				latency10: lat10,
				downKB:    downKB,
				upKB:      upKB,
			}
		}
		m.conns = grown
	}

	if len(m.dns) < target {
		start := len(m.dns)
		grown := make([]dnsRecord, target)
		copy(grown, m.dns)
		rng := uint64(0x94d049bb133111eb) ^ uint64(start*2246822519+7)
		stepRand := func() uint64 {
			rng ^= rng << 13
			rng ^= rng >> 7
			rng ^= rng << 17
			return rng
		}
		for i := start; i < target; i++ {
			r1 := stepRand()
			r2 := stepRand()
			hostIdx := uint16(r1 % uint64(len(m.hostPool)))
			qtypeIdx := uint8((r1 >> 16) % uint64(len(dnsTypeNames)))
			resolverIdx := uint8((r1 >> 24) % uint64(len(dnsResolvers)))
			status := uint8(dnsCached)
			switch (r2 >> 4) % 10 {
			case 6, 7, 8:
				status = dnsDoH
			case 9:
				status = dnsBlocked
			}
			ttl := uint16(30 + (r2 % 3570))
			lat10 := uint16(1 + (r2>>16)%18)
			if status == dnsDoH {
				lat10 = uint16(140 + ((r2 >> 16) % 1600))
			} else if status == dnsBlocked {
				ttl = 0
				lat10 = uint16(1 + ((r2 >> 16) % 8))
			}
			grown[i] = dnsRecord{
				id:          uint64(1_000_000 + i + 1),
				hostIdx:     hostIdx,
				qtypeIdx:    qtypeIdx,
				resolverIdx: resolverIdx,
				status:      status,
				ttlSec:      ttl,
				latency10:   lat10,
			}
		}
		m.dns = grown
	}
}

func (m *monitorState) advanceIfDue(now time.Time) {
	if !m.running {
		return
	}
	if m.lastTick.IsZero() {
		m.lastTick = now
		m.sampleMemory()
		return
	}
	elapsed := now.Sub(m.lastTick)
	if elapsed < telemetryInterval {
		return
	}
	steps := int(elapsed / telemetryInterval)
	if steps > 4 {
		steps = 4
	}
	for range steps {
		m.stepSample()
	}
	m.lastTick = now
	m.sampleMemory()
}

func (m *monitorState) scheduleNext(c *ui.Context, now time.Time) {
	if !m.running {
		m.lastTick = time.Time{}
		return
	}
	if m.lastTick.IsZero() {
		m.lastTick = now
	}
	elapsed := now.Sub(m.lastTick)
	rem := telemetryInterval - elapsed
	if rem <= 0 {
		rem = telemetryInterval
	}
	c.After(rem)
}

func (m *monitorState) stepSample() {
	m.tickCount++
	r := m.nextRand()
	phase := float64(m.tickCount) * 0.28
	jitterDown := float64(int64(r%31)-15) * 1.1
	jitterUp := float64(int64((r>>8)%17)-8) * 0.45

	scaleFactor := 1.0
	if m.routingMode == 2 {
		scaleFactor = 0.72
	}
	down := (68+32*math.Sin(phase)+14*math.Cos(phase*0.55))*scaleFactor + jitterDown
	up := (20+9*math.Cos(phase*0.9)+4*math.Sin(phase*1.4))*scaleFactor + jitterUp
	m.pushTrafficSample(down, up)

	baseConns := m.targetCount()
	delta := int(r%47) - 23
	m.activeConns = max(1, baseConns+delta)
	m.dnsQPS = max(24, 320+int((r>>16)%95)-45)
	m.dnsHitRate = min(0.985, max(0.62, 0.86+0.08*math.Sin(phase*0.4)+float64(int64((r>>24)%11)-5)*0.004))
	m.refreshDNSLabels()
}

func (m *monitorState) resetPeaks() {
	m.downPeakMBps = m.downSpeedMBps
	m.upPeakMBps = m.upSpeedMBps
	m.downPeakText = fmt.Sprintf("Peak %.1f MB/s", m.downPeakMBps)
	m.upPeakText = fmt.Sprintf("Peak %.1f MB/s", m.upPeakMBps)
}

func (m *monitorState) activeSort() (*ui.SortOrder, *[]ui.SortOrder) {
	if m.tableMode == modeDNS {
		return &m.dnsSort, &m.dnsSortBy
	}
	return &m.connSort, &m.connSortBy
}

func (m *monitorState) activeListState() *ui.ListState {
	if m.tableMode == modeDNS {
		return &m.dnsTableState
	}
	return &m.connTableState
}

func (m *monitorState) currentSignature() filterSignature {
	s, sb := m.activeSort()
	return filterSignature{
		mode:        m.tableMode,
		scaleIdx:    m.scaleIdx,
		query:       m.query,
		filterRoute: m.filterRoute,
		filterDNS:   m.filterDNS,
		sort:        *s,
		sortBy:      slices.Clone(*sb),
	}
}

func signaturesEqual(a, b filterSignature) bool {
	return a.mode == b.mode &&
		a.scaleIdx == b.scaleIdx &&
		a.query == b.query &&
		a.filterRoute == b.filterRoute &&
		a.filterDNS == b.filterDNS &&
		a.sort == b.sort &&
		slices.Equal(a.sortBy, b.sortBy)
}

func (m *monitorState) markFilterDirty() {
	m.filterDirty = true
}

func (m *monitorState) ensureFiltered() {
	target := m.targetCount()
	m.ensureScale(target)
	sig := m.currentSignature()
	if m.hasFilter && !m.filterDirty && signaturesEqual(m.lastSig, sig) {
		return
	}
	m.lastSig = sig
	m.hasFilter = true
	m.filterDirty = false

	q := strings.ToLower(strings.TrimSpace(m.query))
	m.filtered = m.filtered[:0]
	if cap(m.filtered) < target {
		m.filtered = make([]int, 0, target)
	}

	if m.tableMode == modeDNS {
		hasFilter := m.filterDNS[0] || m.filterDNS[1] || m.filterDNS[2]
		for i := range target {
			rec := &m.dns[i]
			if hasFilter && !m.filterDNS[rec.status] {
				continue
			}
			if q != "" {
				host := m.hostPoolLower[rec.hostIdx]
				resolver := dnsResolversLower[rec.resolverIdx]
				qtype := dnsTypeNamesLower[rec.qtypeIdx]
				status := dnsStatusNameLower[rec.status]
				if !strings.Contains(host, q) && !strings.Contains(resolver, q) && !strings.Contains(qtype, q) && !strings.Contains(status, q) {
					continue
				}
			}
			m.filtered = append(m.filtered, i)
		}
		m.sortDNSIndices()
	} else {
		hasFilter := m.filterRoute[0] || m.filterRoute[1] || m.filterRoute[2]
		for i := range target {
			rec := &m.conns[i]
			if hasFilter && !m.filterRoute[rec.route] {
				continue
			}
			if q != "" {
				host := m.hostPoolLower[rec.hostIdx]
				proc := processNamesLower[rec.procIdx]
				rule := ruleLabelsLower[rec.ruleIdx]
				route := routeNamesLower[rec.route]
				if !strings.Contains(host, q) && !strings.Contains(proc, q) && !strings.Contains(rule, q) && !strings.Contains(route, q) {
					continue
				}
			}
			m.filtered = append(m.filtered, i)
		}
		m.sortConnIndices()
	}

	// Keep selectedRow synchronized with selectedID, or clear when filtered out.
	m.selectedRow = -1
	if m.selectedID != 0 {
		for rowIdx, recIdx := range m.filtered {
			if m.recordIDRaw(recIdx) == m.selectedID {
				m.selectedRow = rowIdx
				break
			}
		}
	}
}

func (m *monitorState) recordIDRaw(recIdx int) uint64 {
	if m.tableMode == modeDNS {
		return m.dns[recIdx].id
	}
	return m.conns[recIdx].id
}

func (m *monitorState) recordIDAt(row int) uint64 {
	if row < 0 || row >= len(m.filtered) {
		return 0
	}
	return m.recordIDRaw(m.filtered[row])
}

func (m *monitorState) syncSelectedRow(row int) {
	if row < 0 || row >= len(m.filtered) {
		m.selectedRow = -1
		m.selectedID = 0
		return
	}
	m.selectedRow = row
	m.selectedID = m.recordIDAt(row)
}

func (m *monitorState) activeOrders() []ui.SortOrder {
	s, sb := m.activeSort()
	if len(*sb) > 0 {
		return *sb
	}
	if s.Column != "" {
		return []ui.SortOrder{*s}
	}
	return nil
}

func (m *monitorState) sortConnIndices() {
	orders := m.activeOrders()
	if len(orders) == 0 {
		return
	}
	conns := m.conns
	hosts := m.hostPool
	slices.SortFunc(m.filtered, func(ia, ib int) int {
		a, b := &conns[ia], &conns[ib]
		for _, ord := range orders {
			var cmp int
			switch ord.Column {
			case "Host":
				cmp = strings.Compare(hosts[a.hostIdx], hosts[b.hostIdx])
			case "Process":
				cmp = strings.Compare(processNames[a.procIdx], processNames[b.procIdx])
			case "Rule":
				cmp = strings.Compare(ruleLabels[a.ruleIdx], ruleLabels[b.ruleIdx])
			case "Route":
				cmp = int(a.route) - int(b.route)
			case "Latency":
				cmp = int(a.latency10) - int(b.latency10)
			case "Down":
				if a.downKB < b.downKB {
					cmp = -1
				} else if a.downKB > b.downKB {
					cmp = 1
				}
			case "Up":
				if a.upKB < b.upKB {
					cmp = -1
				} else if a.upKB > b.upKB {
					cmp = 1
				}
			}
			if cmp != 0 {
				if ord.Descending {
					return -cmp
				}
				return cmp
			}
		}
		if a.id < b.id {
			return -1
		}
		if a.id > b.id {
			return 1
		}
		return 0
	})
}

func (m *monitorState) sortDNSIndices() {
	orders := m.activeOrders()
	if len(orders) == 0 {
		return
	}
	dns := m.dns
	hosts := m.hostPool
	slices.SortFunc(m.filtered, func(ia, ib int) int {
		a, b := &dns[ia], &dns[ib]
		for _, ord := range orders {
			var cmp int
			switch ord.Column {
			case "Domain":
				cmp = strings.Compare(hosts[a.hostIdx], hosts[b.hostIdx])
			case "Type":
				cmp = strings.Compare(dnsTypeNames[a.qtypeIdx], dnsTypeNames[b.qtypeIdx])
			case "Resolver":
				cmp = strings.Compare(dnsResolvers[a.resolverIdx], dnsResolvers[b.resolverIdx])
			case "Status":
				cmp = int(a.status) - int(b.status)
			case "TTL":
				cmp = int(a.ttlSec) - int(b.ttlSec)
			case "Latency":
				cmp = int(a.latency10) - int(b.latency10)
			}
			if cmp != 0 {
				if ord.Descending {
					return -cmp
				}
				return cmp
			}
		}
		if a.id < b.id {
			return -1
		}
		if a.id > b.id {
			return 1
		}
		return 0
	})
}

func (m *monitorState) connCellText(row, col int) string {
	rec := &m.conns[m.filtered[row]]
	switch col {
	case 0:
		return m.hostPool[rec.hostIdx]
	case 1:
		return processNames[rec.procIdx]
	case 2:
		return ruleLabels[rec.ruleIdx]
	case 3:
		return routeNames[rec.route]
	case 4:
		return protoNames[rec.proto]
	}
	slot := ((rec.id * 7) + uint64(col)) & 63
	e := &m.cellCache[slot]
	if e.valid && e.mode == modeConnections && e.id == rec.id && e.col == col {
		return e.text
	}
	var txt string
	switch col {
	case 5:
		txt = formatLatency10(rec.latency10)
	case 6:
		txt = formatTrafficKB(rec.downKB)
	case 7:
		txt = formatTrafficKB(rec.upKB)
	}
	*e = cellCacheEntry{valid: true, mode: modeConnections, id: rec.id, col: col, text: txt}
	return txt
}

func (m *monitorState) dnsCellText(row, col int) string {
	rec := &m.dns[m.filtered[row]]
	switch col {
	case 0:
		return m.hostPool[rec.hostIdx]
	case 1:
		return dnsTypeNames[rec.qtypeIdx]
	case 2:
		return dnsResolvers[rec.resolverIdx]
	case 3:
		return dnsStatusName[rec.status]
	}
	slot := ((rec.id * 7) + uint64(col)) & 63
	e := &m.cellCache[slot]
	if e.valid && e.mode == modeDNS && e.id == rec.id && e.col == col {
		return e.text
	}
	var txt string
	switch col {
	case 4:
		txt = strconv.Itoa(int(rec.ttlSec)) + "s"
	case 5:
		txt = formatLatency10(rec.latency10)
	}
	*e = cellCacheEntry{valid: true, mode: modeDNS, id: rec.id, col: col, text: txt}
	return txt
}

func (m *monitorState) hostAt(row int) string {
	if row < 0 || row >= len(m.filtered) {
		return ""
	}
	if m.tableMode == modeDNS {
		return m.hostPool[m.dns[m.filtered[row]].hostIdx]
	}
	return m.hostPool[m.conns[m.filtered[row]].hostIdx]
}

func (m *monitorState) clearFilters() {
	m.query = ""
	m.filterRoute = [3]bool{}
	m.filterDNS = [3]bool{}
	m.markFilterDirty()
}

func (m *monitorState) hasActiveFilters() bool {
	if strings.TrimSpace(m.query) != "" {
		return true
	}
	if m.tableMode == modeDNS {
		return m.filterDNS[0] || m.filterDNS[1] || m.filterDNS[2]
	}
	return m.filterRoute[0] || m.filterRoute[1] || m.filterRoute[2]
}

func formatLatency10(lat10 uint16) string {
	return fmt.Sprintf("%.1f ms", float64(lat10)/10.0)
}

func formatTrafficKB(kb uint32) string {
	switch {
	case kb == 0:
		return "0 B"
	case kb >= 1024*1024:
		return fmt.Sprintf("%.2f GB", float64(kb)/(1024*1024))
	case kb >= 1024:
		return fmt.Sprintf("%.1f MB", float64(kb)/1024)
	default:
		return fmt.Sprintf("%d KB", kb)
	}
}

func formatBytesIEC(b float64) string {
	switch {
	case b >= 1024*1024*1024:
		return fmt.Sprintf("%.2f GiB", b/(1024*1024*1024))
	case b >= 1024*1024:
		return fmt.Sprintf("%.2f MiB", b/(1024*1024))
	case b >= 1024:
		return fmt.Sprintf("%.0f KiB", b/1024)
	default:
		return fmt.Sprintf("%.0f B", b)
	}
}
