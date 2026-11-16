package main

import (
	"bytes"
	"compress/zlib"
	"embed"
	"encoding/binary"
	"image"
	"image/color"
	"io"
	"math"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
)

// Live 3D heroes for the class select. heroes/export_live.py decimates the
// Tripo models in Blender and writes assets/heroes/<id>.orbt (triangles with
// UVs and smooth normals, feet at y=0, one unit tall) and <id>.jpg (the
// texture). When a hero has no model the screen falls back to its portrait.
//
//go:embed assets/heroes
var heroFS embed.FS

type heroMesh struct {
	skin *heroSkin    // the idle animation, when the hero is rigged
	pos  []V3         // three per triangle
	uv   [][2]float32 // texture pixels
	nrm  []V3
	tex  uint8
	minX float64 // silhouette extents, for framing
	maxX float64
}

var heroMeshCache = map[string]*heroMesh{}

// heroSkin is a rigged hero: unique vertices with up to four bone weights,
// and a looping idle as per-frame skin matrices (game space, 3x4).
type heroSkin struct {
	pos, nrm []V3
	joint    [][4]uint8
	weight   [][4]float32
	index    []uint32
	bones    int
	frames   int
	fps      float64
	mats     []float32 // frames*bones*12
	at       float64   // the time the corners were last skinned for
	tmp      []float32 // one blended frame
	vp, vn   []V3      // skinned corners (reused every pose)
}

// loadHeroSkin reads assets/heroes/<id>.orbs (see heroes/rigged/export_anim.py).
func loadHeroSkin(id string) (*heroMesh, bool) {
	data, err := heroFS.ReadFile("assets/heroes/" + id + ".orbs")
	if err != nil || len(data) < 8 || string(data[:4]) != "ORBS" {
		return nil, false
	}
	zr, err := zlib.NewReader(bytes.NewReader(data[4:]))
	if err != nil {
		return nil, false
	}
	raw, err := io.ReadAll(zr)
	_ = zr.Close()
	if err != nil || len(raw) < 24 {
		return nil, false
	}
	u32 := func(off int) int { return int(binary.LittleEndian.Uint32(raw[off:])) }
	f32 := func(off int) float32 { return math.Float32frombits(binary.LittleEndian.Uint32(raw[off:])) }
	nv, nt, nb, nf := u32(4), u32(8), u32(12), u32(16)
	fps := float64(f32(20))
	need := 24 + nv*(12+3+8+4+4) + nt*12 + nf*nb*48
	if nv <= 0 || nt <= 0 || nb <= 0 || nf <= 0 || len(raw) < need {
		return nil, false
	}
	sk := &heroSkin{pos: make([]V3, nv), nrm: make([]V3, nv), joint: make([][4]uint8, nv), weight: make([][4]float32, nv),
		index: make([]uint32, nt*3), bones: nb, frames: nf, fps: fps, mats: make([]float32, nf*nb*12), at: -1, tmp: make([]float32, nb*12)}
	o := 24
	for i := 0; i < nv; i++ {
		sk.pos[i] = V3{float64(f32(o)), float64(f32(o + 4)), float64(f32(o + 8))}
		o += 12
	}
	for i := 0; i < nv; i++ {
		n := V3{float64(int8(raw[o])), float64(int8(raw[o+1])), float64(int8(raw[o+2]))}
		if n.Len() > 0 {
			n = n.Norm()
		}
		sk.nrm[i] = n
		o += 3
	}
	uvo := o
	o += nv * 8
	for i := 0; i < nv; i++ {
		copy(sk.joint[i][:], raw[o:o+4])
		o += 4
	}
	for i := 0; i < nv; i++ {
		for k := 0; k < 4; k++ {
			sk.weight[i][k] = float32(raw[o+k]) / 255
		}
		o += 4
	}
	for i := range sk.index {
		sk.index[i] = binary.LittleEndian.Uint32(raw[o:])
		o += 4
	}
	for i := range sk.mats {
		sk.mats[i] = f32(o)
		o += 4
	}
	ti, err := heroFS.ReadFile("assets/heroes/" + id + ".jpg")
	if err != nil {
		return nil, false
	}
	img, _, err := image.Decode(bytes.NewReader(ti))
	if err != nil {
		return nil, false
	}
	tw, th := float32(img.Bounds().Dx()), float32(img.Bounds().Dy())
	m := &heroMesh{skin: sk, pos: make([]V3, nt*3), uv: make([][2]float32, nt*3), nrm: make([]V3, nt*3), minX: 1e9, maxX: -1e9}
	for c, vi := range sk.index {
		if int(vi) >= nv {
			return nil, false
		}
		m.uv[c] = [2]float32{f32(uvo+int(vi)*8) * tw, f32(uvo+int(vi)*8+4) * th}
		m.pos[c], m.nrm[c] = sk.pos[vi], sk.nrm[vi]
	}
	for _, p := range sk.pos {
		m.minX, m.maxX = math.Min(m.minX, p.X), math.Max(m.maxX, p.X)
	}
	m.tex = registerTex(ebiten.NewImageFromImage(img))
	return m, true
}

// animate skins the hero's corners for time t (seconds) on the idle loop.
func (m *heroMesh) animate(t float64) {
	sk := m.skin
	if sk == nil {
		return
	}
	t = math.Floor(t*30) / 30 // a slow idle needs no more than 30 poses a second
	if t == sk.at {
		return
	}
	sk.at = t
	f := math.Mod(t*sk.fps, float64(sk.frames))
	if f < 0 {
		f += float64(sk.frames)
	}
	f0 := int(f)
	f1 := (f0 + 1) % sk.frames
	a := float32(f - float64(f0))
	m0, m1 := sk.mats[f0*sk.bones*12:], sk.mats[f1*sk.bones*12:]
	for i := range sk.tmp {
		sk.tmp[i] = m0[i] + (m1[i]-m0[i])*a
	}
	if len(sk.vp) != len(sk.pos) {
		sk.vp = make([]V3, len(sk.pos))
		sk.vn = make([]V3, len(sk.pos))
	}
	vp, vn := sk.vp, sk.vn
	skinRange := func(lo, hi int) {
		for i := lo; i < hi; i++ {
			p := sk.pos[i]
			var M [12]float32
			for k := 0; k < 4; k++ {
				w := sk.weight[i][k]
				if w == 0 {
					continue
				}
				b := sk.tmp[int(sk.joint[i][k])*12:]
				for e := 0; e < 12; e++ {
					M[e] += w * b[e]
				}
			}
			px, py, pz := float32(p.X), float32(p.Y), float32(p.Z)
			vp[i] = V3{float64(M[0]*px + M[1]*py + M[2]*pz + M[3]), float64(M[4]*px + M[5]*py + M[6]*pz + M[7]), float64(M[8]*px + M[9]*py + M[10]*pz + M[11])}
			n := sk.nrm[i]
			nx, ny, nz := float32(n.X), float32(n.Y), float32(n.Z)
			q := V3{float64(M[0]*nx + M[1]*ny + M[2]*nz), float64(M[4]*nx + M[5]*ny + M[6]*nz), float64(M[8]*nx + M[9]*ny + M[10]*nz)}
			if q.Len() > 1e-9 {
				q = q.Norm()
			}
			vn[i] = q
		}
	}
	// split across a few cores: an idle hero is ~27k vertices
	const parts = 4
	var wg sync.WaitGroup
	nv := len(sk.pos)
	for k := 0; k < parts; k++ {
		wg.Add(1)
		go func(lo, hi int) { defer wg.Done(); skinRange(lo, hi) }(nv*k/parts, nv*(k+1)/parts)
	}
	wg.Wait()
	for c, vi := range sk.index {
		m.pos[c], m.nrm[c] = vp[vi], vn[vi]
	}
}

// heroModel returns the hero's live model, or nil when it isn't in the build.
func heroModel(id string) *heroMesh {
	if m, ok := heroMeshCache[id]; ok {
		return m
	}
	m := loadHeroMesh(id)
	heroMeshCache[id] = m
	return m
}

func loadHeroMesh(id string) *heroMesh {
	if m, ok := loadHeroSkin(id); ok {
		return m
	}
	data, err := heroFS.ReadFile("assets/heroes/" + id + ".orbt")
	if err != nil || len(data) < 8 || string(data[:4]) != "ORBT" {
		return nil
	}
	zr, err := zlib.NewReader(bytes.NewReader(data[4:]))
	if err != nil {
		return nil
	}
	raw, err := io.ReadAll(zr)
	_ = zr.Close()
	if err != nil || len(raw) < 8 {
		return nil
	}
	n := int(binary.LittleEndian.Uint32(raw[4:8]))
	if n <= 0 || len(raw) < 8+n*3*(12+8+3) {
		return nil
	}
	ti, err := heroFS.ReadFile("assets/heroes/" + id + ".jpg")
	if err != nil {
		return nil
	}
	img, _, err := image.Decode(bytes.NewReader(ti))
	if err != nil {
		return nil
	}
	tw, th := float32(img.Bounds().Dx()), float32(img.Bounds().Dy())
	m := &heroMesh{pos: make([]V3, n*3), uv: make([][2]float32, n*3), nrm: make([]V3, n*3), minX: 1e9, maxX: -1e9}
	f32 := func(off int) float64 { return float64(math.Float32frombits(binary.LittleEndian.Uint32(raw[off:]))) }
	po, uo, no := 8, 8+n*36, 8+n*36+n*24
	for i := 0; i < n*3; i++ {
		p := V3{f32(po + i*12), f32(po + i*12 + 4), f32(po + i*12 + 8)}
		m.pos[i] = p
		m.minX, m.maxX = math.Min(m.minX, p.X), math.Max(m.maxX, p.X)
		m.uv[i] = [2]float32{float32(f32(uo+i*8)) * tw, float32(f32(uo+i*8+4)) * th}
		nv := V3{float64(int8(raw[no+i*3])), float64(int8(raw[no+i*3+1])), float64(int8(raw[no+i*3+2]))}
		if nv.Len() > 0 {
			nv = nv.Norm()
		}
		m.nrm[i] = nv
	}
	// the exporter swaps Blender's y and z, a mirror that reverses winding:
	// (the .orbs exporter fixes that itself; this is the static .orbt path)
	// put it back so front faces face the camera
	for i := 0; i+2 < len(m.pos); i += 3 {
		m.pos[i+1], m.pos[i+2] = m.pos[i+2], m.pos[i+1]
		m.uv[i+1], m.uv[i+2] = m.uv[i+2], m.uv[i+1]
		m.nrm[i+1], m.nrm[i+2] = m.nrm[i+2], m.nrm[i+1]
	}
	m.tex = registerTex(ebiten.NewImageFromImage(img))
	return m
}

// heroStage renders a live hero into its own target.
type heroStage struct {
	cam   Camera
	batch *Batch
	rt    *ebiten.Image
	key   heroShotKey // what rt currently holds
}

// heroShotKey identifies a rendered hero frame; when nothing in it has
// changed the last render is reused instead of redrawn.
type heroShotKey struct {
	id        string
	w, h      int
	pose, yaw float64
	rim       color.RGBA
	dim       float64
	zoom      float64
	ok        bool
}

var heroStageOne heroStage

// heroClock drives the idle loops; the class select advances it.
var heroClock float64

// drawHero3D draws the hero standing with its feet at (fx, fy) on dst, h
// pixels tall, turned by yaw radians. rim tints the back light; dim darkens a
// locked hero. It reports false when the hero has no live model.
func drawHero3D(dst *ebiten.Image, id string, fx, fy, w, h, yaw float64, rim color.RGBA, alpha, dim float64) bool {
	m := heroModel(id)
	if m == nil || alpha <= 0 {
		return m != nil
	}
	m.animate(heroClock)
	const ss = 2 // supersample, resolved by the linear downscale below
	W, H := int(math.Ceil(w*ss)), int(math.Ceil(h*ss))
	if W < 8 || H < 8 {
		return true
	}
	st := &heroStageOne
	if st.rt == nil || st.rt.Bounds().Dx() != W || st.rt.Bounds().Dy() != H {
		if st.rt != nil {
			st.rt.Deallocate()
		}
		st.rt = ebiten.NewImage(W, H)
		st.key = heroShotKey{}
	}
	shot := heroShotKey{id: id, w: W, h: H, zoom: heroAuditZoom, pose: math.Floor(heroClock*30) / 30, yaw: math.Round(yaw*400) / 400, rim: rim, dim: dim, ok: true}
	if st.key == shot {
		heroComposite(dst, st.rt, fx, fy, w, h, ss, alpha)
		return true // nothing visible changed since the last render
	}
	st.key = shot
	yaw = shot.yaw
	st.rt.Clear()
	// a long lens keeps the figure flat-ish, like the painted portraits
	dist := 4.2
	st.cam = Camera{Eye: V3{0, 0.62, -dist}, Target: V3{0, 0.5, 0}, FOV: 2 * math.Atan(0.53/dist)}
	if z := heroAuditZoom; z > 1 {
		// the audit's face close-up: same lens, framed on the head
		st.cam = Camera{Eye: V3{0, 0.92, -dist}, Target: V3{0, 0.86, 0}, FOV: 2 * math.Atan(0.53/z/dist)}
	}
	st.cam.Setup(float64(W), float64(H))
	if st.batch == nil {
		st.batch = newBatch(&st.cam)
	}
	b := st.batch
	b.cam = &st.cam
	b.Reset()
	b.fogNear, b.fogFar = 1e6, 2e6
	rot := rotXYZ(0, yaw, 0)
	if drawHeroDepth(st, m, rot, rim, dim) {
		heroComposite(dst, st.rt, fx, fy, w, h, ss, alpha)
		return true
	}
	key := V3{-0.55, 0.7, -0.75}.Norm()
	back := V3{0.5, 0.35, 0.9}.Norm()
	fill := V3{0.7, 0.1, -0.4}.Norm()
	eye := st.cam.Eye
	rimF := [3]float64{float64(rim.R) / 255, float64(rim.G) / 255, float64(rim.B) / 255}
	for i := 0; i+2 < len(m.pos); i += 3 {
		var p [4]V3
		var sh [4]float32
		for k := 0; k < 3; k++ {
			p[k] = rot.Mul(m.pos[i+k])
		}
		fn := p[1].Sub(p[0]).Cross(p[2].Sub(p[0]))
		if fn.Len() < 1e-12 {
			continue
		}
		fn = fn.Norm()
		c := p[0].Add(p[1]).Add(p[2]).Mul(1.0 / 3)
		if fn.Dot(eye.Sub(c)) < 0 {
			continue // facing away
		}
		rimAmt := 0.0
		for k := 0; k < 3; k++ {
			n := rot.Mul(m.nrm[i+k])
			s := 0.5 + 0.55*math.Max(0, n.Dot(key)) + 0.16*math.Max(0, n.Dot(fill)) + 0.1*n.Y
			sh[k] = float32(math.Min(1.15, s))
			rimAmt += math.Max(0, n.Dot(back)) / 3
		}
		p[3], sh[3] = p[2], sh[2]
		r := rimAmt * rimAmt * 0.55
		col := color.RGBA{
			uint8(255 * dim * math.Min(1, 1-r+r*rimF[0]*1.6)),
			uint8(255 * dim * math.Min(1, 1-r+r*rimF[1]*1.6)),
			uint8(255 * dim * math.Min(1, 1-r+r*rimF[2]*1.6)), 255}
		b.QuadUV(p, m.tex, [4][2]float32{m.uv[i], m.uv[i+1], m.uv[i+2], m.uv[i+2]}, col, sh, false)
	}
	own := aaOwn
	aaOwn = false
	b.Flush(st.rt)
	aaOwn = own
	heroComposite(dst, st.rt, fx, fy, w, h, ss, alpha)
	return true
}

func heroComposite(dst, rt *ebiten.Image, fx, fy, w, h, ss, alpha float64) {
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	op.GeoM.Scale(1.0/ss, 1.0/ss)
	op.GeoM.Translate(fx-w/2, fy-h)
	op.ColorScale.ScaleAlpha(float32(alpha))
	dst.DrawImage(rt, op)
}
