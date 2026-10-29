//go:build windows

package main

import (
	"math/rand"
	"runtime"
	"time"
	"unsafe"

	"orb/netplay"
)

// Duels over Steam: a public lobby tagged with a short room code, found
// by code or from the open-room list, and SteamNetworkingMessages for the
// traffic (Steam relays it, so nobody forwards ports). Every request is
// an async API call polled with ISteamUtils, like the leaderboards.

const (
	cbLobbyEnter     = 504
	cbLobbyMatchList = 510
	cbLobbyCreated   = 513
)

type lobbyCreatedT struct {
	Result int32
	_      int32
	Lobby  uint64
}

type lobbyEnterT struct {
	Lobby  uint64
	Perms  uint32
	Locked uint8
	_      [3]byte
	Resp   uint32
	_      uint32
}

type lobbyMatchListT struct {
	Count uint32
}

type steamCall struct {
	id   uint64
	kind int
}

var sNet struct {
	mm, nm  uintptr
	relayOn bool
	calls   []steamCall
	lobby   uint64
	host    bool
	code    string
	joinBy  string // a room code being searched for
	rooms   []duelRoom
	listing bool
	err     string
	link    *steamLink
	checkT  time.Time
}

func steamNetIfaces() bool {
	if !steam.ok || steam.dll == nil {
		return false
	}
	if sNet.mm == 0 {
		sNet.mm = iface(steam.dll, "SteamAPI_SteamMatchmaking_v009")
	}
	if sNet.nm == 0 {
		sNet.nm = iface(steam.dll, "SteamAPI_SteamNetworkingMessages_SteamAPI_v002")
	}
	if !sNet.relayOn {
		if nu := iface(steam.dll, "SteamAPI_SteamNetworkingUtils_SteamAPI_v004"); nu != 0 {
			lbCall("SteamAPI_ISteamNetworkingUtils_InitRelayNetworkAccess", nu)
			sNet.relayOn = true
		}
	}
	return sNet.mm != 0 && sNet.nm != 0
}

func steamDuelAvailable() bool { return steamNetIfaces() }

func steamCStr(p uintptr, max int) string {
	if p == 0 {
		return ""
	}
	bp := *(**byte)(unsafe.Pointer(&p))
	raw := unsafe.Slice(bp, max)
	n := 0
	for n < len(raw) && raw[n] != 0 {
		n++
	}
	return string(raw[:n])
}

func steamLobbyData(lobby uint64, key string) string {
	k := cstr(key)
	p := lbCall("SteamAPI_ISteamMatchmaking_GetLobbyData", sNet.mm, uintptr(lobby), uintptr(unsafe.Pointer(&k[0])))
	runtime.KeepAlive(k)
	return steamCStr(p, 256)
}

func steamSetLobbyData(lobby uint64, key, val string) {
	k, v := cstr(key), cstr(val)
	lbCall("SteamAPI_ISteamMatchmaking_SetLobbyData", sNet.mm, uintptr(lobby), uintptr(unsafe.Pointer(&k[0])), uintptr(unsafe.Pointer(&v[0])))
	runtime.KeepAlive(k)
	runtime.KeepAlive(v)
}

func steamFilter(key, val string) {
	k, v := cstr(key), cstr(val)
	lbCall("SteamAPI_ISteamMatchmaking_AddRequestLobbyListStringFilter", sNet.mm, uintptr(unsafe.Pointer(&k[0])), uintptr(unsafe.Pointer(&v[0])), 0 /* equal */)
	runtime.KeepAlive(k)
	runtime.KeepAlive(v)
}

// steamDuelHost opens a public two-player lobby.
func steamDuelHost() {
	if !steamNetIfaces() {
		return
	}
	steamDuelLeave()
	sNet.err = ""
	if id := lbCall("SteamAPI_ISteamMatchmaking_CreateLobby", sNet.mm, 2 /* public */, 2); id != 0 {
		sNet.calls = append(sNet.calls, steamCall{uint64(id), cbLobbyCreated})
	} else {
		sNet.err = "Steam couldn't create a room"
	}
}

// steamDuelList asks for open rooms (code == "") or one room by code.
func steamDuelList(code string) {
	if !steamNetIfaces() || sNet.listing {
		return
	}
	sNet.joinBy = code
	steamFilter("aorb", "1")
	steamFilter("aorbv", itoa(netplay.Version))
	if code != "" {
		steamFilter("code", code)
	}
	lbCall("SteamAPI_ISteamMatchmaking_AddRequestLobbyListDistanceFilter", sNet.mm, 3 /* worldwide */)
	lbCall("SteamAPI_ISteamMatchmaking_AddRequestLobbyListFilterSlotsAvailable", sNet.mm, 1)
	lbCall("SteamAPI_ISteamMatchmaking_AddRequestLobbyListResultCountFilter", sNet.mm, 30)
	if id := lbCall("SteamAPI_ISteamMatchmaking_RequestLobbyList", sNet.mm); id != 0 {
		sNet.listing = true
		sNet.calls = append(sNet.calls, steamCall{uint64(id), cbLobbyMatchList})
	}
}

func steamDuelJoin(lobby uint64) {
	if !steamNetIfaces() {
		return
	}
	steamDuelLeave()
	sNet.err = ""
	if id := lbCall("SteamAPI_ISteamMatchmaking_JoinLobby", sNet.mm, uintptr(lobby)); id != 0 {
		sNet.calls = append(sNet.calls, steamCall{uint64(id), cbLobbyEnter})
	}
}

func steamDuelLeave() {
	if sNet.link != nil {
		sNet.link.Close()
		sNet.link = nil
	}
	if sNet.lobby != 0 && sNet.mm != 0 {
		lbCall("SteamAPI_ISteamMatchmaking_LeaveLobby", sNet.mm, uintptr(sNet.lobby))
	}
	sNet.lobby, sNet.host, sNet.code = 0, false, ""
}

func randomRoomCode() string {
	const a = "ABCDEFGHJKMNPQRSTVWXYZ23456789"
	b := make([]byte, 6)
	for i := range b {
		b[i] = a[rand.Intn(len(a))]
	}
	return string(b)
}

// steamDuelPoll finishes async calls and watches the lobby for a peer.
// It returns the link once the other player is in the room.
func steamDuelPoll(myName string) (link netplay.Link, code string, rooms []duelRoom, err string) {
	if !steamNetIfaces() {
		return nil, "", nil, ""
	}
	keep := sNet.calls[:0]
	for _, c := range sNet.calls {
		var failed uint8
		if !boolRet(lbCall("SteamAPI_ISteamUtils_IsAPICallCompleted", steam.utils, uintptr(c.id), uintptr(unsafe.Pointer(&failed)))) {
			keep = append(keep, c)
			continue
		}
		switch c.kind {
		case cbLobbyCreated:
			var res lobbyCreatedT
			ok := boolRet(lbCall("SteamAPI_ISteamUtils_GetAPICallResult", steam.utils, uintptr(c.id), uintptr(unsafe.Pointer(&res)), unsafe.Sizeof(res), cbLobbyCreated, uintptr(unsafe.Pointer(&failed))))
			if !ok || failed != 0 || res.Result != 1 || res.Lobby == 0 {
				sNet.err = "Steam couldn't create a room"
				continue
			}
			sNet.lobby, sNet.host = res.Lobby, true
			sNet.code = randomRoomCode()
			steamSetLobbyData(res.Lobby, "aorb", "1")
			steamSetLobbyData(res.Lobby, "aorbv", itoa(netplay.Version))
			steamSetLobbyData(res.Lobby, "code", sNet.code)
			steamSetLobbyData(res.Lobby, "name", myName)
		case cbLobbyMatchList:
			sNet.listing = false
			var res lobbyMatchListT
			ok := boolRet(lbCall("SteamAPI_ISteamUtils_GetAPICallResult", steam.utils, uintptr(c.id), uintptr(unsafe.Pointer(&res)), unsafe.Sizeof(res), cbLobbyMatchList, uintptr(unsafe.Pointer(&failed))))
			if !ok || failed != 0 {
				sNet.err = "Couldn't reach Steam's room list"
				continue
			}
			sNet.rooms = sNet.rooms[:0]
			for i := 0; i < int(res.Count); i++ {
				id := uint64(lbCall("SteamAPI_ISteamMatchmaking_GetLobbyByIndex", sNet.mm, uintptr(i)))
				if id == 0 {
					continue
				}
				sNet.rooms = append(sNet.rooms, duelRoom{Name: steamLobbyData(id, "name"), Code: steamLobbyData(id, "code"), id: id})
			}
			if sNet.joinBy != "" {
				if len(sNet.rooms) == 0 {
					sNet.err = "No open room with code " + sNet.joinBy
				} else {
					steamDuelJoin(sNet.rooms[0].id)
				}
				sNet.joinBy = ""
			}
		case cbLobbyEnter:
			var res lobbyEnterT
			ok := boolRet(lbCall("SteamAPI_ISteamUtils_GetAPICallResult", steam.utils, uintptr(c.id), uintptr(unsafe.Pointer(&res)), unsafe.Sizeof(res), cbLobbyEnter, uintptr(unsafe.Pointer(&failed))))
			if !ok || failed != 0 || res.Resp != 1 {
				sNet.err = "That room is full or has closed"
				continue
			}
			sNet.lobby = res.Lobby
			sNet.host = uint64(lbCall("SteamAPI_ISteamMatchmaking_GetLobbyOwner", sNet.mm, uintptr(res.Lobby))) == steamMyID()
			sNet.code = steamLobbyData(res.Lobby, "code")
		}
	}
	sNet.calls = keep
	// find the other player in the room
	if sNet.lobby != 0 && time.Since(sNet.checkT) > 400*time.Millisecond {
		sNet.checkT = time.Now()
		me := steamMyID()
		var peer uint64
		n := int(lbCall("SteamAPI_ISteamMatchmaking_GetNumLobbyMembers", sNet.mm, uintptr(sNet.lobby)))
		for i := 0; i < n; i++ {
			if m := uint64(lbCall("SteamAPI_ISteamMatchmaking_GetLobbyMemberByIndex", sNet.mm, uintptr(sNet.lobby), uintptr(i))); m != 0 && m != me {
				peer = m
			}
		}
		switch {
		case peer != 0 && sNet.link == nil:
			sNet.link = &steamLink{peer: peer}
			if sNet.host {
				lbCall("SteamAPI_ISteamMatchmaking_SetLobbyJoinable", sNet.mm, uintptr(sNet.lobby), 0)
			}
		case peer == 0 && sNet.link != nil:
			sNet.link.gone = true
		}
	}
	if sNet.link != nil {
		link = sNet.link
	}
	return link, sNet.code, sNet.rooms, sNet.err
}

// ---------------------------------------------------------------- the link

type steamLink struct {
	peer     uint64
	closed   bool
	gone     bool
	acceptT  time.Time
	identity [136]byte
	idSet    bool
}

func (l *steamLink) id() *byte {
	if !l.idSet {
		lbCall("SteamAPI_SteamNetworkingIdentity_SetSteamID64", uintptr(unsafe.Pointer(&l.identity[0])), uintptr(l.peer))
		l.idSet = true
	}
	return &l.identity[0]
}

func (l *steamLink) Send(msg []byte, reliable bool) {
	if l.closed || len(msg) == 0 {
		return
	}
	flags := uintptr(32) // auto-restart a broken session
	if reliable {
		flags |= 8
	}
	lbCall("SteamAPI_ISteamNetworkingMessages_SendMessageToUser", sNet.nm, uintptr(unsafe.Pointer(l.id())),
		uintptr(unsafe.Pointer(&msg[0])), uintptr(len(msg)), flags, 0)
	runtime.KeepAlive(msg)
}

func (l *steamLink) Recv() [][]byte {
	if l.closed {
		return nil
	}
	if time.Since(l.acceptT) > time.Second {
		// accept the peer's session (harmless if already accepted)
		l.acceptT = time.Now()
		lbCall("SteamAPI_ISteamNetworkingMessages_AcceptSessionWithUser", sNet.nm, uintptr(unsafe.Pointer(l.id())))
	}
	var out [][]byte
	var ptrs [32]uintptr
	for {
		n := int(int32(lbCall("SteamAPI_ISteamNetworkingMessages_ReceiveMessagesOnChannel", sNet.nm, 0, uintptr(unsafe.Pointer(&ptrs[0])), uintptr(len(ptrs)))))
		for i := 0; i < n; i++ {
			p := ptrs[i]
			if p == 0 {
				continue
			}
			base := **(**[32]uint64)(unsafe.Pointer(&p)) // SteamNetworkingMessage_t header (Steam owns it)
			data, size := uintptr(base[0]), int(int32(base[1]))
			from := base[3] // m_identityPeer.m_steamID64 (offset 24)
			if from == l.peer && data != 0 && size > 0 && size < 1<<20 {
				b := make([]byte, size)
				copy(b, unsafe.Slice(*(**byte)(unsafe.Pointer(&data)), size))
				out = append(out, b)
			}
			lbCall("SteamAPI_SteamNetworkingMessage_t_Release", p)
		}
		if n < len(ptrs) {
			break
		}
	}
	return out
}

func (l *steamLink) Alive() bool { return !l.closed && !l.gone }

func (l *steamLink) Close() {
	if l.closed {
		return
	}
	l.closed = true
	lbCall("SteamAPI_ISteamNetworkingMessages_CloseSessionWithUser", sNet.nm, uintptr(unsafe.Pointer(l.id())))
}

func (l *steamLink) Kind() string { return "steam" }

func sNetHost() bool { return sNet.host }
