//go:build windows

package main

import (
	"runtime"
	"syscall"
	"unsafe"
)

// Steam leaderboards through the flat API. Every request is an async
// "API call" that we poll from the game loop with ISteamUtils, so no
// callback registration is needed.

const (
	cbLeaderboardFind       = 1104 // LeaderboardFindResult_t
	cbLeaderboardDownloaded = 1105 // LeaderboardScoresDownloaded_t
	cbLeaderboardUploaded   = 1106 // LeaderboardScoreUploaded_t
)

// Valve packs its callback structs to 8 bytes on Windows.
type lbFindResult struct {
	Handle uint64
	Found  uint8
	_      [7]byte
}

type lbDownloaded struct {
	Board, Entries uint64
	Count          int32
	_              int32
}

type lbEntryRaw struct {
	SteamID uint64
	Rank    int32
	Score   int32
	Details int32
	_       int32
	UGC     uint64
}

type pendingCall struct {
	id   uint64
	kind int
	tag  string // which board / request
}

var lbSteam struct {
	handle  uint64
	pending []pendingCall
}

func lbProc(name string) *syscall.LazyProc {
	if steam.dll == nil {
		return nil
	}
	p := steam.dll.NewProc(name)
	if p.Find() != nil {
		return nil
	}
	return p
}

func lbCall(name string, args ...uintptr) uintptr {
	p := lbProc(name)
	if p == nil {
		return 0
	}
	r, _, _ := p.Call(args...)
	return r
}

func steamLBAvailable() bool { return steam.ok && steam.utils != 0 && !steam.testApp }

// steamLBFind asks for the board by name (created on first use).
func steamLBFind(name string) {
	if !steamLBAvailable() {
		return
	}
	b := cstr(name)
	id := lbCall("SteamAPI_ISteamUserStats_FindOrCreateLeaderboard", steam.stats, uintptr(unsafe.Pointer(&b[0])), 2 /* descending */, 1 /* numeric */)
	runtime.KeepAlive(b)
	if id != 0 {
		lbSteam.pending = append(lbSteam.pending, pendingCall{uint64(id), cbLeaderboardFind, name})
	}
}

// steamLBDownload requests entries: kind 0 = global, 2 = friends.
func steamLBDownload(kind, from, to int) {
	if lbSteam.handle == 0 {
		return
	}
	id := lbCall("SteamAPI_ISteamUserStats_DownloadLeaderboardEntries", steam.stats, uintptr(lbSteam.handle), uintptr(kind), uintptr(from), uintptr(to))
	if id != 0 {
		tag := "global"
		if kind == 2 {
			tag = "friends"
		}
		lbSteam.pending = append(lbSteam.pending, pendingCall{uint64(id), cbLeaderboardDownloaded, tag})
	}
}

// steamLBUpload keeps the player's best score; details carry loop, depth,
// origin and ascension.
func steamLBUpload(score int, details []int32) {
	if lbSteam.handle == 0 || len(details) == 0 {
		return
	}
	id := lbCall("SteamAPI_ISteamUserStats_UploadLeaderboardScore", steam.stats, uintptr(lbSteam.handle), 1 /* keep best */, uintptr(uint32(int32(score))),
		uintptr(unsafe.Pointer(&details[0])), uintptr(len(details)))
	runtime.KeepAlive(details)
	if id != 0 {
		lbSteam.pending = append(lbSteam.pending, pendingCall{uint64(id), cbLeaderboardUploaded, "upload"})
	}
}

// steamLBPoll finishes any completed calls and reports what arrived.
func steamLBPoll(onFound func(), onEntries func(tag string, rows []lbRow), onUploaded func()) {
	if !steamLBAvailable() || len(lbSteam.pending) == 0 {
		return
	}
	keep := lbSteam.pending[:0]
	for _, pc := range lbSteam.pending {
		var failed uint8
		done := lbCall("SteamAPI_ISteamUtils_IsAPICallCompleted", steam.utils, uintptr(pc.id), uintptr(unsafe.Pointer(&failed)))
		if !boolRet(done) {
			keep = append(keep, pc)
			continue
		}
		if failed != 0 {
			continue
		}
		switch pc.kind {
		case cbLeaderboardFind:
			var res lbFindResult
			if boolRet(lbCall("SteamAPI_ISteamUtils_GetAPICallResult", steam.utils, uintptr(pc.id), uintptr(unsafe.Pointer(&res)), unsafe.Sizeof(res), cbLeaderboardFind, uintptr(unsafe.Pointer(&failed)))) && res.Found != 0 {
				lbSteam.handle = res.Handle
				onFound()
			}
		case cbLeaderboardDownloaded:
			var res lbDownloaded
			if !boolRet(lbCall("SteamAPI_ISteamUtils_GetAPICallResult", steam.utils, uintptr(pc.id), uintptr(unsafe.Pointer(&res)), unsafe.Sizeof(res), cbLeaderboardDownloaded, uintptr(unsafe.Pointer(&failed)))) {
				continue
			}
			var rows []lbRow
			me := steamMyID()
			for i := 0; i < int(res.Count) && i < 50; i++ {
				var e lbEntryRaw
				var det [4]int32
				if !boolRet(lbCall("SteamAPI_ISteamUserStats_GetDownloadedLeaderboardEntry", steam.stats, uintptr(res.Entries), uintptr(i), uintptr(unsafe.Pointer(&e)), uintptr(unsafe.Pointer(&det[0])), 4)) {
					continue
				}
				row := lbRow{Rank: int(e.Rank), Name: steamPersona(e.SteamID), Score: int(e.Score), You: e.SteamID == me}
				if e.Details >= 1 {
					row.Loop = int(det[0])
				}
				if e.Details >= 3 {
					row.Arch = int(det[2])
				}
				rows = append(rows, row)
			}
			onEntries(pc.tag, rows)
		case cbLeaderboardUploaded:
			onUploaded()
		}
	}
	lbSteam.pending = keep
}

func steamMyID() uint64 {
	user := iface(steam.dll, "SteamAPI_SteamUser_v023", "SteamAPI_SteamUser_v022", "SteamAPI_SteamUser_v021")
	if user == 0 {
		return 0
	}
	return uint64(lbCall("SteamAPI_ISteamUser_GetSteamID", user))
}

func steamPersona(id uint64) string {
	if steam.friends == 0 {
		return "Player"
	}
	p := lbCall("SteamAPI_ISteamFriends_GetFriendPersonaName", steam.friends, uintptr(id))
	if p == 0 {
		return "Player"
	}
	// copy the C string out before Steam reuses its buffer
	bp := *(**byte)(unsafe.Pointer(&p)) // Steam owns this C string
	raw := unsafe.Slice(bp, 128)
	n := 0
	for n < len(raw) && raw[n] != 0 {
		n++
	}
	return string(raw[:n])
}
