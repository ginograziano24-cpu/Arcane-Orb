//go:build windows

package main

import (
	"os"
	"runtime"
	"path/filepath"
	"syscall"
	"unsafe"
)

// A minimal Steamworks bridge that talks to steam_api64.dll through its
// flat C API. It loads lazily and only if the DLL sits next to the game,
// so non-Steam builds (and machines without Steam) run unchanged.

// steamAppID is the store's App ID. Set it once the app exists on Steam;
// while it is 0 the game never asks Steam to relaunch it.
const steamAppID = 0

type steamBridge struct {
	ok                       bool
	stats, utils             uintptr
	runCallbacks, setAch     *syscall.LazyProc
	storeStats, getAch, deck *syscall.LazyProc
}

var steam steamBridge

func boolRet(r uintptr) bool { return r&0xff != 0 }

func cstr(s string) []byte { return append([]byte(s), 0) }

// steamInit connects to Steam if possible. It returns true when the game
// should quit because Steam is relaunching it.
func steamInit() (relaunch bool) {
	exe, _ := os.Executable()
	dllPath := filepath.Join(filepath.Dir(exe), "steam_api64.dll")
	if _, err := os.Stat(dllPath); err != nil {
		return false
	}
	dll := syscall.NewLazyDLL(dllPath)
	if dll.Load() != nil {
		return false
	}
	defer func() {
		if recover() != nil {
			steam.ok = false
		}
	}()
	if steamAppID != 0 {
		if r, _, _ := dll.NewProc("SteamAPI_RestartAppIfNecessary").Call(uintptr(steamAppID)); boolRet(r) {
			return true
		}
	}
	var errMsg [1024]byte
	if p := dll.NewProc("SteamAPI_InitFlat"); p.Find() == nil {
		if r, _, _ := p.Call(uintptr(unsafe.Pointer(&errMsg[0]))); r != 0 {
			return false
		}
	} else if p := dll.NewProc("SteamAPI_Init"); p.Find() == nil {
		if r, _, _ := p.Call(); !boolRet(r) {
			return false
		}
	} else {
		return false
	}
	us, _, _ := dll.NewProc("SteamAPI_SteamUserStats_v013").Call()
	ut, _, _ := dll.NewProc("SteamAPI_SteamUtils_v011").Call()
	steam = steamBridge{
		ok: us != 0, stats: us, utils: ut,
		runCallbacks: dll.NewProc("SteamAPI_RunCallbacks"),
		setAch:       dll.NewProc("SteamAPI_ISteamUserStats_SetAchievement"),
		storeStats:   dll.NewProc("SteamAPI_ISteamUserStats_StoreStats"),
		getAch:       dll.NewProc("SteamAPI_ISteamUserStats_GetAchievement"),
		deck:         dll.NewProc("SteamAPI_ISteamUtils_IsSteamRunningOnSteamDeck"),
	}
	return false
}

func steamTick() {
	if steam.ok {
		steam.runCallbacks.Call()
	}
}

// steamUnlock reports an achievement to Steam (no-op without Steam).
func steamUnlock(id string) {
	if !steam.ok {
		return
	}
	b := cstr(id)
	steam.setAch.Call(steam.stats, uintptr(unsafe.Pointer(&b[0])))
	runtime.KeepAlive(b)
	steam.storeStats.Call(steam.stats)
}

// steamHas asks Steam whether an achievement is already unlocked.
func steamHas(id string) bool {
	if !steam.ok {
		return false
	}
	var got byte
	b := cstr(id)
	r, _, _ := steam.getAch.Call(steam.stats, uintptr(unsafe.Pointer(&b[0])), uintptr(unsafe.Pointer(&got)))
	runtime.KeepAlive(b)
	return boolRet(r) && got != 0
}

func steamOnDeck() bool {
	if !steam.ok || steam.utils == 0 || steam.deck.Find() != nil {
		return false
	}
	r, _, _ := steam.deck.Call(steam.utils)
	return boolRet(r)
}

func steamActive() bool { return steam.ok }
