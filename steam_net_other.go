//go:build !windows

package main

import "orb/netplay"

func steamDuelAvailable() bool   { return false }
func steamDuelHost()             {}
func steamDuelList(code string)  {}
func steamDuelJoin(lobby uint64) {}
func steamDuelLeave()            {}
func steamDuelPoll(myName string) (netplay.Link, string, []duelRoom, string) {
	return nil, "", nil, ""
}

func sNetHost() bool { return false }

func steamMyID() uint64             { return 0 }
func steamPersona(id uint64) string { return "" }
