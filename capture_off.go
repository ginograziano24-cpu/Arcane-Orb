//go:build !(capture && js) && !heroreview

package main

import "github.com/hajimehoshi/ebiten/v2"

func captureUpdate(g *Game) bool                      { return false }
func captureSkipDraw(g *Game, dst *ebiten.Image) bool { return false }
func captureAfterDraw(g *Game, dst *ebiten.Image)     {}

