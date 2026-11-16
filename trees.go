package main

import (
	"image/color"
	"sync"
)

// Textured trees: a flared, slightly leaning trunk wrapped in embossed bark
// with a couple of boughs, under a crown of several lumpy leaf masses
// wrapped in an embossed leaf texture. The crown sways a little in the
// breeze, each mass on its own beat, so the trees feel alive.

var (
	treeTexOnce      sync.Once
	barkTex, leafTex *bakedTex
	leafWhiteTex     *bakedTex
)

func treeTextures() (bark, leaf, leafWhite *bakedTex) {
	treeTexOnce.Do(func() {
		barkTex = bakedTexture("tex_bark")
		leafTex = bakedTexture("tex_leaf")
		leafWhiteTex = bakedTexture("tex_leaf_white")
	})
	return barkTex, leafTex, leafWhiteTex
}

func trunkMesh(v int) *Mesh {
	return cached("tTrunk"+itoa(v), func() *Mesh {
		// flared roots, a waist, then a taper up into the crown
		return Lathe([][2]float64{{0.95, 0}, {0.62, 0.35}, {0.42, 0.9}, {0.36, 2.2}, {0.3, 3.4}, {0.18, 4.6}}, 9,
			[]color.RGBA{{255, 255, 255, 255}}, false)
	})
}

func boughMesh() *Mesh {
	return cached("tBough", func() *Mesh {
		return Lathe([][2]float64{{0.2, 0}, {0.13, 1.4}, {0.06, 2.4}}, 6, []color.RGBA{{255, 255, 255, 255}}, false)
	})
}

func crownMesh(v int) *Mesh {
	return cached("tCrown"+itoa(v), func() *Mesh {
		return Icosphere(1, 0.22, int64(40+v), color.RGBA{255, 255, 255, 255}, color.RGBA{255, 255, 255, 255})
	})
}

// leafyTree draws a tree with its base at (x, y, z), scale s: a grown
// broadleaf (flora.go) with barked trunk, roots, limbs and leaf clusters.
// leaf tints the canopy (white keeps its own greens).
func leafyTree(B *Batch, x, y, z, s float64, seed int64, leaf, bark color.RGBA, t float64) {
	base := V3{x, y, z}
	set := treeVariantLOD("oak", seed, floraLOD(B, V3{x, 0, z}), func(sd int64, lite bool) *floraSet {
		return growTree(sd, barkOak, palOak, lite, nil)
	})
	tint := color.RGBA{}
	if leaf != (color.RGBA{255, 255, 255, 255}) {
		tint = leaf
	}
	drawFlora(B, set, base, v3(s), float64(seed)*1.3, tint, t+float64(seed), 0.012)
}
