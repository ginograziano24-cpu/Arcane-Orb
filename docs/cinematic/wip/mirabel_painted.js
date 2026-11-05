  // ================================================================ Mirabel, painted (baked once into sprites)
  // Units: feet at (0,0), y up is negative, about 565 units tall with the hat.
  // Key light from the upper left (her staff), magic rim light from the right hand.
  var MPaint = (function () {
    var SC = 2; // bake scale, set per bake (pixels per unit)
    function tan(P, i, closed) {
      var n = P.length, p = P[i]; if (p[2]) return [0, 0];
      var a = P[closed ? (i - 1 + n) % n : Math.max(0, i - 1)], b = P[closed ? (i + 1) % n : Math.min(n - 1, i + 1)];
      var k = closed || (i > 0 && i < n - 1) ? .5 : 1; return [(b[0] - a[0]) * k, (b[1] - a[1]) * k];
    }
    function segs(P, closed) {
      var out = [], n = P.length, m = closed ? n : n - 1;
      for (var i = 0; i < m; i++) { var j = (i + 1) % n, t0 = tan(P, i, closed), t1 = tan(P, j, closed);
        out.push([P[i], [P[i][0] + t0[0] / 3, P[i][1] + t0[1] / 3], [P[j][0] - t1[0] / 3, P[j][1] - t1[1] / 3], P[j]]); }
      return out;
    }
    var PC = new Map();
    function path(P, closed) {
      var key = P; if (PC.has(key) && PC.get(key).c === closed) return PC.get(key).p;
      var s = segs(P, closed), p = new Path2D(); p.moveTo(P[0][0], P[0][1]);
      s.forEach(function (q) { p.bezierCurveTo(q[1][0], q[1][1], q[2][0], q[2][1], q[3][0], q[3][1]); });
      if (closed) p.closePath(); PC.set(key, { p: p, c: closed }); return p;
    }
    function sample(P, closed) {
      var pts = []; segs(P, closed).forEach(function (q, si) {
        var len = Math.hypot(q[3][0] - q[0][0], q[3][1] - q[0][1]) + Math.hypot(q[1][0] - q[0][0], q[1][1] - q[0][1]) * .5, steps = Math.max(4, Math.ceil(len / .8));
        for (var k = si ? 1 : 0; k <= steps; k++) { var t = k / steps, u = 1 - t;
          pts.push([u * u * u * q[0][0] + 3 * u * u * t * q[1][0] + 3 * u * t * t * q[2][0] + t * t * t * q[3][0], u * u * u * q[0][1] + 3 * u * u * t * q[1][1] + 3 * u * t * t * q[2][1] + t * t * t * q[3][1]]); }
      }); return pts;
    }
    // tapered, light-aware ink: thicker on the side turned away from the light, thin into the ends
    function ink(g, P, closed, w, col, o) {
      o = o || {}; var pts = sample(P, closed), n = pts.length; if (n < 2) return;
      var area = 0; for (var i = 0; i < n; i++) { var a = pts[i], b = pts[(i + 1) % n]; area += a[0] * b[1] - b[0] * a[1]; }
      var sgn = area > 0 ? 1 : -1, L = [], R = [], acc = [0];
      for (var i2 = 1; i2 < n; i2++) acc.push(acc[i2 - 1] + Math.hypot(pts[i2][0] - pts[i2 - 1][0], pts[i2][1] - pts[i2 - 1][1]));
      var tot = acc[n - 1] || 1, tp = o.taper == null ? (closed ? 0 : .22) : o.taper;
      for (var j = 0; j < n; j++) {
        var p0 = pts[Math.max(0, j - 1)], p1 = pts[Math.min(n - 1, j + 1)];
        if (closed) { p0 = pts[(j - 1 + n) % n]; p1 = pts[(j + 1) % n]; }
        var dx = p1[0] - p0[0], dy = p1[1] - p0[1], dl = Math.hypot(dx, dy) || 1, nx = dy / dl * sgn, ny = -dx / dl * sgn;
        var s = acc[j] / tot, prof = tp > 0 ? Math.pow(Math.min(1, s / tp, (1 - s) / tp), .65) : 1;
        if (o.start) prof = Math.min(prof * 0 + 1, prof); // keep
        var lit = o.flat ? 1 : .62 + .76 * Math.max(0, nx * .55 + ny * .83);
        var ww = Math.max(.12, w * prof * lit * (o.wob ? 1 + Math.sin(acc[j] * .21 + w) * .12 : 1)) / 2;
        L.push([pts[j][0] + nx * ww, pts[j][1] + ny * ww]); R.push([pts[j][0] - nx * ww, pts[j][1] - ny * ww]);
      }
      g.beginPath(); g.moveTo(L[0][0], L[0][1]); for (var a2 = 1; a2 < n; a2++) g.lineTo(L[a2][0], L[a2][1]);
      if (closed) { g.closePath(); g.moveTo(R[n - 1][0], R[n - 1][1]); for (var b2 = n - 2; b2 >= 0; b2--) g.lineTo(R[b2][0], R[b2][1]); g.closePath(); g.fillStyle = col; g.fill('evenodd'); return; }
      for (var b3 = n - 1; b3 >= 0; b3--) g.lineTo(R[b3][0], R[b3][1]); g.closePath(); g.fillStyle = col; g.fill();
    }
    function fill(g, P, col) { g.fillStyle = col; g.fill(path(P, true)); }
    function clip(g, P, fn) { g.save(); g.clip(path(P, true)); fn(); g.restore(); }
    function soft(g, P, col, blur, alpha, mode, closed) { // a soft-edged painted shape
      g.save(); g.globalCompositeOperation = mode || 'multiply'; g.globalAlpha = alpha == null ? 1 : alpha; if (blur) g.filter = 'blur(' + (blur * SC).toFixed(1) + 'px)';
      g.fillStyle = col; g.fill(path(P, closed !== false)); g.restore();
    }
    function blob(g, x, y, rx, ry, rot, col, blur, alpha, mode) {
      g.save(); g.globalCompositeOperation = mode || 'multiply'; g.globalAlpha = alpha == null ? 1 : alpha; if (blur) g.filter = 'blur(' + (blur * SC).toFixed(1) + 'px)';
      g.fillStyle = col; g.beginPath(); g.ellipse(x, y, rx, ry, rot || 0, 0, 7); g.fill(); g.restore();
    }
    function star(g, x, y, r, col, a) { g.save(); g.globalAlpha = a == null ? 1 : a; g.fillStyle = col; g.beginPath(); for (var i = 0; i < 8; i++) { var rr = i % 2 ? r * .32 : r, an = i / 8 * Math.PI * 2 - Math.PI / 2; g.lineTo(x + Math.cos(an) * rr, y + Math.sin(an) * rr); } g.closePath(); g.fill(); g.restore(); }
    function moon(g, x, y, r, col, rot) { g.save(); g.translate(x, y); g.rotate(rot || 0); g.fillStyle = col; g.beginPath(); g.arc(0, 0, r, 0, 7); g.arc(r * .45, -r * .2, r * .82, 0, 7, true); g.fill('evenodd'); g.restore(); }

    // ---------------------------------------------------------------- palette
    var C = {
      skin: '#f6cfb4', skinSh: '#d98a7c', skinDk: '#b5635c', skinHi: '#fff3ea', skinLn: '#7c3b33',
      hair: '#d9472b', hairSh: '#7d1517', hairDk: '#4e0b12', hairHi: '#ffb27a', hairLn: '#4a0d10',
      gown: '#a8318f', gownSh: '#5d1866', gownDk: '#36103f', gownHi: '#ff9fe2', gownLn: '#2e0a33',
      vio: '#432a7e', vioSh: '#1f1245', vioHi: '#8f74e0', vioLn: '#140a2c', lining: '#b9a0ff',
      gold: '#f3c24f', goldSh: '#a86f1c', goldHi: '#fff4bf', goldLn: '#6b400e',
      wood: '#7a4a2e', woodSh: '#43261a', woodHi: '#b07a4e', woodLn: '#2c1710',
      eye: '#2e9e6a', lip: '#d42a4e', lipHi: '#ff8ea3', rimC: '#7fe9ff', rimP: '#ff9be6'
    };

    // ---------------------------------------------------------------- shapes
    var S = {};
    S.backHair = [[-6, -472], [22, -470], [40, -454], [48, -428], [50, -400], [58, -372], [66, -340], [64, -306], [70, -276], [62, -246, 1], [56, -262], [48, -238, 1], [40, -262], [24, -270], [-24, -272], [-40, -262], [-48, -236, 1], [-56, -258], [-66, -240, 1], [-70, -272], [-66, -306], [-66, -342], [-58, -374], [-50, -404], [-46, -432], [-36, -456], [-22, -468]];
    S.collarL = [[-12, -398], [-30, -401], [-46, -414], [-56, -438, 1], [-44, -430], [-30, -418], [-16, -410]];
    S.collarR = [[10, -400], [28, -403], [44, -416], [54, -440, 1], [42, -432], [28, -420], [14, -412]];
    S.neck = [[-8, -416], [7, -416], [9, -398], [12, -392], [-13, -390], [-10, -398]];
    S.chest = [[-14, -396], [12, -398], [27, -388], [32, -372], [0, -364], [-33, -368], [-29, -386]];
    S.gown = [[-31, -370], [0, -365], [30, -373], [41, -371], [54, -355], [54, -338], [40, -321], [24, -296], [40, -268], [47, -246], [44, -214], [36, -176], [31, -136], [36, -96], [52, -48], [80, -4, 1], [54, 4], [26, 2], [0, 6], [-30, 2], [-60, 4], [-86, -6, 1], [-62, -46], [-44, -94], [-36, -140], [-42, -180], [-54, -222], [-62, -256], [-54, -276], [-27, -298], [-40, -321], [-55, -338], [-55, -355], [-44, -370, 1]];
    S.skirtFold = [[[-6, -232], [-4, -170], [-8, -100], [-14, -10]], [[20, -150], [24, -90], [30, -6]], [[-30, -120], [-38, -60], [-50, -4]], [[44, -60], [56, -10]], [[8, -60], [10, -4]]];
    S.cincher = [[-40, -322], [-20, -316], [0, -318], [20, -316], [40, -322], [27, -298], [31, -279], [16, -271], [0, -268], [-18, -272], [-36, -280], [-30, -298]];
    S.capeL = [[-12, -398], [-28, -394], [-46, -386], [-58, -372], [-64, -352], [-62, -334, 1], [-52, -340], [-44, -336, 1], [-38, -350], [-32, -362], [-28, -378], [-18, -392]];
    S.capeR = [[10, -400], [26, -397], [44, -390], [57, -378], [63, -358], [62, -340, 1], [52, -345], [44, -340, 1], [38, -354], [33, -366], [29, -382], [18, -394]];
    S.armUpL = [[-62, -346], [-50, -342], [-51, -318], [-56, -298], [-66, -294], [-70, -318]];
    S.sleeveL = [[-70, -316], [-58, -304], [-74, -290], [-92, -284, 1], [-98, -266], [-97, -236], [-92, -206, 1], [-84, -230], [-74, -262], [-64, -286]];
    S.liningL = [[-92, -284], [-98, -266], [-97, -236], [-92, -206], [-90, -232], [-90, -262]];
    S.armUpR = [[50, -346], [62, -344], [68, -320], [66, -298], [56, -296], [52, -318]];
    S.sleeveR = [[54, -312], [68, -312], [84, -292], [100, -278, 1], [104, -258], [101, -228], [96, -206, 1], [88, -234], [76, -264], [60, -290]];
    S.liningR = [[100, -278], [104, -258], [101, -228], [96, -206], [95, -236], [96, -264]];
    S.handL = [[-108, -290], [-97, -292], [-90, -284], [-90, -272], [-96, -264], [-106, -264], [-112, -272]];
    S.palmR = [[98, -279], [110, -282], [119, -276], [120, -266], [113, -259], [102, -261]];
    S.fingersR = [[[117, -275], [128, -270], [136, -262]], [[118, -270], [128, -262], [134, -253]], [[116, -265], [124, -256], [128, -248]], [[112, -262], [117, -255], [119, -249]], [[105, -279], [112, -284], [119, -283]]];
    S.face = [[0, -467], [13, -463], [20, -451], [21, -437], [18, -424], [12, -413], [4, -406], [0, -405], [-4, -406], [-12, -413], [-18, -424], [-21, -437], [-20, -451], [-13, -463]];
    S.bangs = [[-22, -470], [-6, -475], [14, -475], [28, -468], [32, -450], [30, -432, 1], [22, -440], [12, -446], [2, -451], [-8, -454], [-16, -455], [-22, -450, 1], [-23, -460]];
    S.strandL = [[-21, -464], [-33, -452], [-40, -432], [-38, -414], [-44, -398], [-52, -380], [-52, -360], [-46, -344], [-50, -326], [-44, -312, 1], [-38, -326], [-40, -344], [-42, -362], [-38, -380], [-30, -396], [-26, -414], [-24, -436], [-22, -452]];
    S.strandR = [[20, -464], [33, -452], [38, -432], [36, -414], [42, -398], [48, -384], [50, -366, 1], [40, -378], [32, -394], [26, -414], [24, -436], [22, -452]];
    S.hatCone = [[-28, -466], [-20, -494], [-8, -522], [6, -544], [28, -560], [44, -556, 1], [30, -548], [18, -530], [22, -500], [32, -468]];
    S.staff = [[-112, 0], [-106, 0], [-98, -470], [-104, -470]];
    // shading shapes
    S.gownShade = [[52, -356], [52, -338], [40, -321], [24, -296], [40, -268], [47, -246], [44, -214], [36, -176], [31, -136], [36, -96], [52, -48], [80, -4], [40, 2], [24, -40], [20, -100], [22, -140], [28, -180], [34, -220], [28, -256], [14, -284], [24, -322], [40, -338]];
    S.bustL = [-22, -350]; S.bustR = [23, -352];

    // ---------------------------------------------------------------- the painting
    function paint(g, o) {
      o = o || {}; var wink = !!o.wink;
      g.lineJoin = 'round';
      // staff glow behind everything
      g.save(); g.globalCompositeOperation = 'lighter'; var sg = g.createRadialGradient(-100, -486, 2, -100, -486, 70); sg.addColorStop(0, 'rgba(160,240,255,.55)'); sg.addColorStop(1, 'rgba(160,240,255,0)'); g.fillStyle = sg; g.fillRect(-180, -560, 160, 150); g.restore();

      // back hair
      fill(g, S.backHair, C.hair);
      clip(g, S.backHair, function () {
        soft(g, [[-70, -330], [-40, -330], [-40, -230], [-70, -230]], C.hairSh, 6, .7);
        soft(g, [[30, -400], [70, -400], [70, -230], [30, -230]], C.hairSh, 8, .7);
        blob(g, -40, -440, 6, 3, .6, C.hairHi, 2, .55, 'screen'); blob(g, 40, -440, 6, 3, -.6, C.hairHi, 2, .4, 'screen');
        for (var i = 0; i < 9; i++) ink(g, [[-58 + i * 14, -420 + (i % 3) * 8], [-60 + i * 14 + (i % 2 ? 6 : -4), -330], [-56 + i * 14, -262]], false, 1.1, 'rgba(78,11,18,.55)');
      });
      ink(g, S.backHair, true, 2.4, C.hairLn);

      // collar wings (behind the head)
      [S.collarL, S.collarR].forEach(function (cw, k) {
        fill(g, cw, C.vio); clip(g, cw, function () { soft(g, cw.map(function (p) { return [p[0] + (k ? 6 : 6), p[1] + 6]; }), C.vioSh, 3, .8); });
        ink(g, cw, true, 1.6, C.vioLn); ink(g, cw.slice(1, 4), false, 1.2, C.gold, { taper: .1 });
      });

      // neck and chest skin
      fill(g, S.neck, C.skin); fill(g, S.chest, C.skin);
      clip(g, S.neck, function () { soft(g, [[-12, -420], [12, -420], [12, -400], [-12, -404]], C.skinSh, 3, .9); soft(g, [[2, -416], [12, -416], [12, -390], [4, -392]], C.skinSh, 2, .6); });
      clip(g, S.chest, function () {
        soft(g, [[-30, -396], [30, -398], [30, -388], [-30, -386]], C.skinSh, 3, .6);
        ink(g, [[-20, -384], [-10, -381], [-3, -383]], false, .7, 'rgba(124,59,51,.45)'); ink(g, [[3, -384], [11, -382], [21, -386]], false, .7, 'rgba(124,59,51,.45)');
        blob(g, -12, -376, 10, 4, -.1, C.skinHi, 3, .5, 'screen');
      });
      ink(g, S.neck.slice(0, 3), false, 1.2, C.skinLn); ink(g, [S.neck[0], S.neck[5], S.neck[4]], false, 1.2, C.skinLn);

      // gown
      var gg = g.createLinearGradient(0, -380, 0, 0); gg.addColorStop(0, C.gown); gg.addColorStop(.55, '#9a2b86'); gg.addColorStop(1, '#7c2272');
      g.fillStyle = gg; g.fill(path(S.gown, true));
      clip(g, S.gown, function () {
        // core shadow down the right side, soft into the form
        soft(g, S.gownShade, C.gownSh, 6, .72);
        // bust forms: shadow under and to the right, satin sheen upper left
        soft(g, [[-46, -333], [46, -335], [44, -322], [-44, -322]], C.gownSh, 3, .38); // the bust's cast shadow on the body
        [S.bustL, S.bustR].forEach(function (b, k) {
          blob(g, b[0] + 4, b[1] + 11, 18, 6.5, .12 * (k ? -1 : 1), C.gownSh, 3, .8);
          blob(g, b[0] + 9, b[1] + 3, 9, 15, 0, C.gownSh, 5, k ? .55 : .25);
          blob(g, b[0] - 4, b[1] - 3, 11, 9, -.5, C.gownHi, 5, k ? .28 : .42, 'screen');
          blob(g, b[0] - 8, b[1] - 7, 3.5, 2.4, -.5, '#ffe0f4', 1.6, .35, 'screen');
        });
        blob(g, 0, -346, 3.5, 9, 0, C.gownSh, 3, .45); // the soft dip between
        // hips and thighs
        blob(g, -46, -250, 10, 20, .2, C.gownHi, 7, .4, 'screen');
        blob(g, -28, -196, 9, 36, .12, C.gownHi, 9, .32, 'screen');
        blob(g, 4, -244, 22, 9, 0, C.gownSh, 6, .45); // the soft dip below the belly
        blob(g, 2, -160, 5, 60, .05, C.gownSh, 7, .45); // between the legs
        blob(g, 18, -142, 7, 12, 0, C.gownHi, 5, .35, 'screen'); blob(g, 26, -150, 4, 30, 0, C.gownSh, 5, .4); // the relaxed knee, forward
        S.skirtFold.forEach(function (f, i) { ink(g, f, false, 3.2, 'rgba(54,16,63,.38)', { taper: .3 }); ink(g, f.map(function (p) { return [p[0] - 4, p[1]]; }), false, 2.2, 'rgba(255,170,230,.18)', { taper: .35 }); });
        // embroidered stars across the skirt
        for (var s = 0; s < 22; s++) { var sx = -70 + ((s * 37) % 140), sy = -200 + ((s * 53) % 190); star(g, sx, sy, 2 + (s % 3), C.gold, .35 + (s % 2) * .2); }
        // hem band: gold with moons and stars
        g.save(); g.beginPath(); g.moveTo(-90, -20); g.bezierCurveTo(-40, -14, 40, -14, 90, -20); g.lineTo(90, 10); g.lineTo(-90, 10); g.closePath(); g.fillStyle = C.vio; g.fill();
        g.lineWidth = 1.6; g.strokeStyle = C.gold; g.beginPath(); g.moveTo(-90, -20); g.bezierCurveTo(-40, -14, 40, -14, 90, -20); g.stroke(); g.beginPath(); g.moveTo(-90, -15); g.bezierCurveTo(-40, -9, 40, -9, 90, -15); g.lineWidth = .8; g.stroke(); g.restore();
        for (var hx = -74; hx <= 74; hx += 16) { if ((hx / 16 | 0) % 2) star(g, hx, -6, 3.4, C.gold, .95); else moon(g, hx, -6, 3.6, C.gold, -.4); }
        // rim light on the right edge (magic from her hand)
        g.save(); g.translate(-1.6, 0); ink(g, S.gown.slice(3, 16), false, 2.2, 'rgba(127,233,255,.55)', { taper: .15, flat: 1 }); g.restore();
      });
      ink(g, S.gown, true, 2.4, C.gownLn);
      // bust underside and flank contour lines (shadow side only, as accents)
      ink(g, [[-43, -350], [-39, -338], [-28, -330], [-16, -331], [-7, -338]], false, .9, C.gownLn, { taper: .3 }); ink(g, [[7, -340], [16, -333], [28, -332], [39, -338], [45, -350]], false, 1.2, C.gownLn, { taper: .25 });
      // neckline trim
      ink(g, [[-33, -369], [0, -364], [31, -373]], false, 2.2, C.gold, { taper: .05 });
      // corset cincher: violet, gold piping, laced front
      fill(g, S.cincher, C.vio);
      clip(g, S.cincher, function () { soft(g, [[10, -322], [44, -322], [40, -270], [12, -266]], C.vioSh, 4, .8); blob(g, -22, -300, 8, 14, 0, C.vioHi, 5, .5, 'screen'); for (var b = 0; b < 4; b++) ink(g, [[-24 + b * 16, -318], [-25 + b * 16, -270]], false, .8, 'rgba(243,194,79,.55)'); });
      ink(g, S.cincher, true, 1.8, C.vioLn);
      [-1, 1].forEach(function (sd) { ink(g, [[sd * 12, -310], [sd * 20, -304], [sd * 16, -296], [sd * 22, -288], [sd * 28, -292]], false, .9, C.gold, { taper: .3 }); ink(g, [[sd * 14, -282], [sd * 20, -278], [sd * 26, -282]], false, .7, C.gold, { taper: .3 }); });
      ink(g, S.cincher.slice(0, 5), false, 1.4, C.gold, { taper: .06 }); ink(g, S.cincher.slice(6, 11), false, 1.4, C.gold, { taper: .06 });
      g.save(); g.strokeStyle = C.goldHi; g.lineWidth = .8; for (var lc = 0; lc < 5; lc++) { var ly = -312 + lc * 8.5; g.beginPath(); g.moveTo(-4, ly); g.lineTo(4, ly + 6); g.moveTo(4, ly); g.lineTo(-4, ly + 6); g.stroke(); } g.restore();
      // star charm at the waist
      ink(g, [[0, -268], [1, -256]], false, .7, C.gold, { flat: 1 }); star(g, 1, -251, 6, C.gold); star(g, 1, -251, 2.5, C.goldHi);

      // capelet halves with gold trim and a gem clasp
      [S.capeL, S.capeR].forEach(function (cp, k) {
        fill(g, cp, C.vio);
        clip(g, cp, function () { if (k) soft(g, [[30, -400], [70, -400], [70, -330], [36, -340]], C.vioSh, 5, .85); else soft(g, [[-70, -360], [-40, -360], [-40, -330], [-70, -330]], C.vioSh, 4, .5); blob(g, k ? 38 : -42, -388, 12, 5, k ? .4 : -.4, C.vioHi, 4, k ? .3 : .55, 'screen'); for (var st = 0; st < 4; st++) star(g, (k ? 36 : -50) + st * 5, -376 + (st % 2) * 12, 1.6, C.gold, .7); });
        ink(g, cp, true, 1.9, C.vioLn); ink(g, cp.slice(4, 10), false, 1.4, C.gold, { taper: .06 });
      });
      g.save(); g.shadowColor = '#ff5fd2'; g.shadowBlur = 8 * SC; g.fillStyle = '#e8459e'; g.beginPath(); g.moveTo(-1, -404); g.lineTo(5, -398); g.lineTo(-1, -389); g.lineTo(-7, -398); g.closePath(); g.fill(); g.restore();
      ink(g, [[-1, -404], [5, -398], [-1, -389], [-7, -398]], true, 1, C.goldLn); blob(g, -2.5, -400, 1.6, 1, 0, '#ffffff', .4, .9, 'source-over');

      // arms: fitted upper sleeves, bell sleeves with lavender lining
      [[S.armUpL, S.sleeveL, S.liningL, 0], [S.armUpR, S.sleeveR, S.liningR, 1]].forEach(function (a) {
        fill(g, a[0], C.gown); clip(g, a[0], function () { soft(g, a[0].map(function (p) { return [p[0] + 5, p[1]]; }), C.gownSh, 3, a[3] ? .85 : .55); });
        ink(g, a[0], true, 1.8, C.gownLn);
        fill(g, a[1], C.gown); clip(g, a[1], function () { soft(g, a[1].map(function (p) { return [p[0] + (a[3] ? 3 : 6), p[1] + 6]; }), C.gownSh, 4, .75); blob(g, a[3] ? 70 : -72, -296, 6, 12, a[3] ? -.7 : .7, C.gownHi, 4, .4, 'screen'); });
        fill(g, a[2], C.lining); clip(g, a[2], function () { soft(g, a[2], '#5b3fa8', 3, .5); });
        ink(g, a[1], true, 1.8, C.gownLn); ink(g, a[2].slice(0, 4), false, 1.6, C.gold, { taper: .05 });
        for (var cs = 0; cs < 4; cs++) { var cp2 = a[2][Math.min(3, cs)], nx2 = a[2][Math.min(3, cs + 1)]; star(g, (cp2[0] + nx2[0]) / 2 + (a[3] ? -3 : 3), (cp2[1] + nx2[1]) / 2, 1.8, C.gold, .9); }
      });

      // staff: dark wood with a crescent holding a little orb
      var staffP = [[-112, -2], [-106, -2], [-101, -150], [-99, -300], [-97, -466], [-104, -468], [-106, -300], [-108, -150]];
      fill(g, staffP, C.wood); clip(g, staffP, function () { soft(g, [[-106, 0], [-100, 0], [-96, -470], [-101, -470]], C.woodSh, 1.5, .8); soft(g, [[-111, 0], [-109, 0], [-104, -470], [-106, -470]], C.woodHi, 1, .6, 'screen'); });
      ink(g, staffP, true, 1.6, C.woodLn);
      [[-106, -120], [-103, -360]].forEach(function (k) { blob(g, k[0], k[1], 4.5, 3, 0, C.woodSh, .6, .9, 'source-over'); ink(g, [[k[0] - 4, k[1]], [k[0], k[1] - 2.5], [k[0] + 4, k[1]]], false, .8, C.woodLn); });
      g.save(); g.translate(-101, -488); g.rotate(-.35);
      g.beginPath(); g.arc(0, 0, 24, .5, Math.PI * 2 - .5 + Math.PI * 0, false); g.arc(7, -2, 19, Math.PI * 2 - .7, .7, true); g.closePath();
      var mg = g.createLinearGradient(-24, -24, 24, 24); mg.addColorStop(0, C.goldHi); mg.addColorStop(.5, C.gold); mg.addColorStop(1, C.goldSh); g.fillStyle = mg; g.fill(); g.lineWidth = 1.4; g.strokeStyle = C.goldLn; g.stroke(); g.restore();
      g.save(); g.globalCompositeOperation = 'lighter'; var og = g.createRadialGradient(-96, -490, 1, -96, -490, 13); og.addColorStop(0, '#ffffff'); og.addColorStop(.35, '#bff4ff'); og.addColorStop(1, 'rgba(110,220,255,0)'); g.fillStyle = og; g.beginPath(); g.arc(-96, -490, 13, 0, 7); g.fill(); g.restore();
      star(g, -96, -490, 9, 'rgba(255,255,255,.9)');

      // hands
      fill(g, S.handL, C.skin); clip(g, S.handL, function () { soft(g, [[-100, -292], [-88, -292], [-88, -262], [-100, -262]], C.skinSh, 2.5, .8); });
      ink(g, S.handL, true, 1.4, C.skinLn);
      [[-106, -282], [-106, -276], [-106, -270]].forEach(function (f) { ink(g, [[f[0], f[1]], [f[0] + 8, f[1] - 1], [f[0] + 13, f[1] + 1]], false, .8, C.skinLn); });
      blob(g, -91, -280, 1.6, 1.2, 0, C.lip, .4, .9, 'source-over'); // a painted nail
      ink(g, [[-94, -290], [-90, -282]], false, 2.2, C.gold, { flat: 1, taper: 0 }); ink(g, [[100, -281], [101, -273]], false, 2.2, C.gold, { flat: 1, taper: 0 });
      S.fingersR.forEach(function (f, i) { g.save(); g.lineCap = 'round'; g.lineJoin = 'round'; g.strokeStyle = C.skinLn; g.lineWidth = i === 4 ? 5.4 : i === 3 ? 4.4 : 5; g.stroke(path(f, false)); g.strokeStyle = C.skin; g.lineWidth = i === 4 ? 3.8 : i === 3 ? 2.9 : 3.5; g.stroke(path(f, false)); g.restore(); var e = f[f.length - 1]; blob(g, e[0], e[1], 1.3, 1, 0, C.lip, .3, .95, 'source-over'); });
      fill(g, S.palmR, C.skin); clip(g, S.palmR, function () { soft(g, [[100, -266], [118, -266], [118, -258], [100, -258]], C.skinSh, 3, .7); });
      ink(g, S.palmR, true, 1.3, C.skinLn);
      // a gold ring on the casting hand
      ink(g, [[112, -268], [115, -265]], false, 1.8, C.gold, { flat: 1, taper: 0 });

      // face
      fill(g, S.face, C.skin);
      clip(g, S.face, function () {
        soft(g, [[8, -470], [24, -470], [24, -404], [6, -404], [14, -430]], C.skinSh, 5, .55); // turned-away side
        soft(g, [[-24, -458], [26, -458], [30, -440], [20, -446], [-24, -447]], C.skinSh, 2.5, .55); // hair cast shadow
        blob(g, -12, -425, 6, 3.6, 0, '#f07a8a', 3, .4, 'multiply'); blob(g, 12, -425, 5.5, 3.4, 0, '#f07a8a', 3, .36, 'multiply'); // blush
        blob(g, -8, -436, 8, 10, 0, C.skinHi, 5, .4, 'screen'); blob(g, 0, -408, 4, 2, 0, C.skinHi, 2, .5, 'screen');
      });
      ink(g, S.face.slice(2, 11), false, 1.5, C.skinLn, { taper: .12 });
      // eyes (adult, almond, lashes flicked out)
      function eye(cx, side, closed) {
        var cy = -436, w2 = 7.2;
        if (closed) { ink(g, [[cx - w2, cy - .5], [cx, cy + 2.2], [cx + w2, cy - .5]], false, 1.6, '#2a0f14', { taper: .2 }); ink(g, [[cx + side * w2, cy - .5], [cx + side * (w2 + 3), cy - 2.5]], false, 1, '#2a0f14'); ink(g, [[cx + side * (w2 - 2), cy + .8], [cx + side * (w2 + 1), cy + 2]], false, .8, '#2a0f14'); return; }
        var up = [[cx - w2, cy + .8], [cx - 2.4, cy - 4], [cx + 3, cy - 3.8], [cx + w2, cy - .6]], lo = [[cx + w2, cy - .6], [cx + 2, cy + 3.2], [cx - 3, cy + 3.2], [cx - w2, cy + .8]];
        blob(g, cx + side * 1.5, cy - 4.6, 8, 3.2, 0, '#a2428f', 1.5, .6);
        g.save(); var ep = path(up.concat(lo.slice(1, 3)), true); g.fillStyle = '#fffaf4'; g.fill(ep); g.clip(ep);
        g.fillStyle = C.eye; g.beginPath(); g.arc(cx + .4, cy, 4.1, 0, 7); g.fill(); g.fillStyle = '#16402d'; g.beginPath(); g.arc(cx + .4, cy - .6, 3.7, Math.PI, 0); g.fill();
        g.fillStyle = '#0d1a14'; g.beginPath(); g.arc(cx + .4, cy, 1.7, 0, 7); g.fill(); g.fillStyle = '#9ff0c6'; g.globalAlpha = .7; g.beginPath(); g.arc(cx + .4, cy + 1.8, 1.6, 0, Math.PI); g.fill(); g.globalAlpha = 1;
        g.fillStyle = '#fff'; g.beginPath(); g.arc(cx - .9, cy - 1.4, 1.1, 0, 7); g.fill(); g.beginPath(); g.arc(cx + 1.8, cy + 1.1, .5, 0, 7); g.fill();
        soft(g, [[cx - 8, cy - 5], [cx + 8, cy - 5], [cx + 8, cy - 1.8], [cx - 8, cy - 1.8]], C.skinSh, .8, .5); g.restore();
        ink(g, up, false, 3.1, '#24090f', { taper: .08 }); ink(g, lo.slice(0, 3), false, .7, 'rgba(124,59,51,.8)', { taper: .3 });
        ink(g, [[cx + side * (w2 - 1), cy - .8], [cx + side * (w2 + 2), cy - 2.4], [cx + side * (w2 + 4), cy - 4.4]], false, 1.6, '#24090f', { taper: .35 }); ink(g, [[cx + side * (w2 - 2.6), cy - 2.8], [cx + side * (w2 - .4), cy - 5], [cx + side * (w2 + 1), cy - 7]], false, 1.1, '#24090f', { taper: .35 }); ink(g, [[cx + side * (w2 - 4.6), cy - 3.6], [cx + side * (w2 - 3), cy - 6.4]], false, .8, '#24090f', { taper: .35 });
        ink(g, [[cx - 4, cy - 6], [cx + 1, cy - 6.8], [cx + 5, cy - 5.6]], false, .5, 'rgba(124,59,51,.5)'); // lid crease
      }
      eye(-9, -1, false); eye(9.5, 1, wink);
      ink(g, [[-17, -445], [-11, -450], [-4, -448.4]], false, 1.5, '#5a1410', { taper: .3 }); ink(g, [[4, -449], [11, -451], [18, -446]], false, 1.5, '#5a1410', { taper: .3 });
      // nose and lips
      ink(g, [[2, -429], [2.6, -425.4], [.6, -424]], false, .8, 'rgba(124,59,51,.75)'); blob(g, -.5, -423.6, 1.8, .8, 0, C.skinSh, .5, .7);
      blob(g, -.5, -430, .8, 3, 0, C.skinHi, .6, .7, 'screen');
      var lips = [[-6.6, -417.6], [-2.8, -418.9], [0, -418.1], [2.8, -418.9], [6.6, -417.6], [3.2, -414.2], [0, -413.5], [-3.2, -414.2]];
      fill(g, lips, C.lip); clip(g, lips, function () { soft(g, [[-8, -420], [8, -420], [8, -416.3], [-8, -416.3]], '#8f1430', .4, .45); blob(g, -1.2, -414.9, 2.2, .7, 0, C.lipHi, .4, .7, 'screen'); });
      ink(g, [[-6.8, -417.8], [-3.2, -416.4], [0, -416.1], [3.2, -416.4], [6.8, -417.8]], false, .75, '#6e1022', { taper: .25 }); 
      ink(g, [[-7.4, -418.6], [-6.9, -417.4]], false, .5, 'rgba(124,59,51,.7)'); ink(g, [[7.4, -418.6], [6.9, -417.4]], false, .5, 'rgba(124,59,51,.7)');

      // front hair: swept bangs and long strands framing the face
      [S.strandL, S.strandR].forEach(function (sh, k) {
        fill(g, sh, C.hair); clip(g, sh, function () { soft(g, sh.map(function (p) { return [p[0] + (k ? 5 : 6), p[1] + 2]; }), C.hairSh, 3, .55); blob(g, k ? 31 : -32, -438, 6, 3, 0, C.hairHi, 1.6, .7, 'screen'); blob(g, k ? 40 : -45, -385, 3, 8, k ? -.5 : .5, C.hairHi, 2, .45, 'screen'); for (var q = 0; q < 3; q++) ink(g, sh.slice(1, 8).map(function (p, ii) { return [p[0] + (k ? -1 : 1) * (2 + q * 3.2) * (ii > 4 ? .6 : 1), p[1] + q * 2]; }), false, .7, 'rgba(110,20,20,.55)'); });
        ink(g, sh, true, 1.8, C.hairLn); ink(g, sh.slice(2, 5).map(function (p) { return [p[0] + (k ? -3 : 3), p[1]]; }), false, .8, 'rgba(78,11,18,.6)');
      });
      fill(g, S.bangs, C.hair);
      clip(g, S.bangs, function () { soft(g, [[0, -470], [32, -470], [32, -440], [10, -444]], C.hairSh, 3, .7); blob(g, -6, -463, 14, 2.6, -.1, C.hairHi, 1.4, .75, 'screen'); blob(g, 12, -462, 9, 2.2, .1, C.hairHi, 1.4, .5, 'screen'); });
      ink(g, S.bangs, true, 1.7, C.hairLn);
      ink(g, [[-12, -466], [-6, -456], [-4, -450]], false, .8, 'rgba(78,11,18,.7)'); ink(g, [[10, -468], [14, -458], [18, -450]], false, .8, 'rgba(78,11,18,.7)');

      // hat: wide tilted brim, bent cone, gold band, a crescent pin
      g.save(); g.translate(0, -8);
      g.save(); g.translate(2, -463); g.rotate(-.1);
      g.beginPath(); g.ellipse(0, 0, 66, 13.5, 0, 0, 7); var hb = g.createLinearGradient(-66, 0, 66, 0); hb.addColorStop(0, C.vioHi); hb.addColorStop(.35, C.vio); hb.addColorStop(1, C.vioSh); g.fillStyle = hb; g.fill();
      g.save(); g.clip(); g.globalCompositeOperation = 'multiply'; g.filter = 'blur(' + (2 * SC) + 'px)'; g.fillStyle = C.vioSh; g.beginPath(); g.ellipse(0, 7, 60, 7, 0, 0, 7); g.fill(); g.restore();
      g.lineWidth = 2.2; g.strokeStyle = C.vioLn; g.beginPath(); g.ellipse(0, 0, 66, 13.5, 0, 0, 7); g.stroke();
      g.lineWidth = 1; g.strokeStyle = C.gold; g.beginPath(); g.ellipse(0, 0, 62, 11.5, 0, .15, Math.PI - .15); g.stroke();
      g.restore();
      fill(g, S.hatCone, C.vio); clip(g, S.hatCone, function () { soft(g, [[6, -560], [50, -560], [40, -464], [12, -464], [20, -510]], C.vioSh, 5, .9); soft(g, [[-26, -470], [-14, -520], [-8, -520], [-16, -470]], C.vioHi, 3, .55, 'screen'); for (var hs = 0; hs < 6; hs++) star(g, -12 + hs * 6 + (hs % 2) * 3, -486 - hs * 9, 1.8, C.gold, .8); });
      ink(g, S.hatCone, true, 2.2, C.vioLn);
      var band = [[-27, -469], [2, -474], [31, -471], [31, -480], [2, -484], [-25, -479]];
      fill(g, band, C.gold); clip(g, band, function () { soft(g, [[10, -486], [34, -486], [34, -466], [10, -466]], C.goldSh, 2, .7); blob(g, -12, -478, 8, 1.4, 0, C.goldHi, .8, .8, 'screen'); });
      ink(g, band, true, 1.1, C.goldLn);
      moon(g, 15, -478, 6, C.goldHi, -.5); moon(g, 15, -478, 6, 'rgba(168,111,28,.35)', -.5); star(g, 21, -485, 2.6, '#ffffff', .9);
      g.restore();

      // a soft halo of magic around the casting hand
      g.save(); g.globalCompositeOperation = 'lighter'; var hg = g.createRadialGradient(140, -250, 2, 140, -250, 34); hg.addColorStop(0, 'rgba(255,160,230,.45)'); hg.addColorStop(1, 'rgba(255,160,230,0)'); g.fillStyle = hg; g.fillRect(100, -290, 90, 90); g.restore();
      star(g, 136, -246, 6, 'rgba(255,240,255,.95)'); star(g, 146, -228, 3.4, 'rgba(200,250,255,.9)'); star(g, 128, -276, 2.6, 'rgba(255,255,255,.9)');
      // heels peeking out
      ink(g, [[-30, 4], [-22, 2], [-14, 5]], false, 2.6, C.vioLn, { taper: .3 }); ink(g, [[10, 6], [18, 3], [26, 6]], false, 2.6, C.vioLn, { taper: .3 });
    }

    // bake: returns a canvas with the figure, and the offset of the feet within it
    var BOX = { x0: -190, y0: -580, x1: 190, y1: 24 };
    function bake(scale, opts) {
      SC = scale; var c = document.createElement('canvas'); c.width = Math.ceil((BOX.x1 - BOX.x0) * scale); c.height = Math.ceil((BOX.y1 - BOX.y0) * scale);
      var g = c.getContext('2d'); g.scale(scale, scale); g.translate(-BOX.x0, -BOX.y0); paint(g, opts); return c;
    }
    return { bake: bake, box: BOX, paint: paint };
  })();
