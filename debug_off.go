//go:build !debug

package main

import "github.com/hajimehoshi/ebiten/v2"

func debugKeys(g *Game) {}

func debugMenuKeys(g *Game) {}
func debugMapKeys(g *Game)  {}

func debugOverlay(g *Game, dst *ebiten.Image) {}
