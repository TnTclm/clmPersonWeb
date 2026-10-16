<script setup>
// ============================================================
// HeartView —— 💖 爱心页（星尘之心 · 完全体，12 秒循环）
// 双 canvas 架构：
//   · 背景层：一次性绘制的静态深空（银河带 + 固定星 + 渐晕），零每帧成本
//   · 粒子层：拖尾用 destination-out 擦除法（旧帧衰减 30%，透明区保持透明）
// 心体：经典心形线（横向加宽）+ 加厚厚度，始终基本正面（±7° 缓摆）；2400 颗小粒子各自闪烁 + 微游动
// 特效：能量脉络扫光 / 星芒十字闪 / 轮廓流光 / 青色星环（匀速旋转，无跳动）
// 剧本：0-1.5s 螺旋吸入 / 1.5-9.5s 心动 / 9.5-12s 烟花三波爆散 → 吸回轮回
// ============================================================

import { ref, onMounted, onUnmounted } from 'vue'

// ---------- 剧本时间线（毫秒） ----------
const CYCLE = 12000
const T_GATHER = 1500    // 汇聚完成
const T_BURST = 9500     // 开始消散

const FOV = 1400

const cv = ref(null)     // 粒子层
const bgcv = ref(null)   // 背景层
let rafId = 0
let particles = []
let comets = []
let ring = []
let dust = []
let cw = 0
let ch = 0
let sprites = {}
let cx0 = 0
let cyH = 0
let SCALE = 0

// 该角度 t 上，心形边界离原点多远（心形是"星形域"）
function heartR(t) {
  const hx = 16 * Math.sin(t) ** 3
  const hy = 13 * Math.cos(t) - 5 * Math.cos(2 * t) - 2 * Math.cos(3 * t) - Math.cos(4 * t)
  return Math.hypot(hx, hy)
}

// 心形方程 → 屏幕方向单位向量（y 翻成屏幕向下为正）
function heartDir(t) {
  const d = heartR(t)
  return [
    (16 * Math.sin(t) ** 3) / d,
    -(13 * Math.cos(t) - 5 * Math.cos(2 * t) - 2 * Math.cos(3 * t) - Math.cos(4 * t)) / d,
  ]
}

// 随机 3D 单位向量
function randDir3() {
  const u = Math.random() * 2 - 1
  const a = Math.random() * Math.PI * 2
  const s = Math.sqrt(1 - u * u)
  return [s * Math.cos(a), u, s * Math.sin(a)]
}

// ---------- 发光小球贴图 ----------
function makeSprite(core, mid) {
  const c = document.createElement('canvas')
  c.width = 64
  c.height = 64
  const g = c.getContext('2d')
  const rg = g.createRadialGradient(32, 32, 0, 32, 32, 32)
  rg.addColorStop(0, core)
  rg.addColorStop(0.28, mid)
  rg.addColorStop(1, 'rgba(0,0,0,0)')
  g.fillStyle = rg
  g.fillRect(0, 0, 64, 64)
  return c
}

// 十字星芒贴图：两条渐变细线交叉 + 中心光点（闪烁峰值时替身登场）
function makeStarSprite() {
  const c = document.createElement('canvas')
  c.width = 64
  c.height = 64
  const g = c.getContext('2d')
  let lg = g.createLinearGradient(0, 0, 0, 64)
  lg.addColorStop(0, 'rgba(255,255,255,0)')
  lg.addColorStop(0.5, 'rgba(255,240,248,0.95)')
  lg.addColorStop(1, 'rgba(255,255,255,0)')
  g.fillStyle = lg
  g.fillRect(29, 2, 6, 60)
  lg = g.createLinearGradient(0, 0, 64, 0)
  lg.addColorStop(0, 'rgba(255,255,255,0)')
  lg.addColorStop(0.5, 'rgba(255,240,248,0.95)')
  lg.addColorStop(1, 'rgba(255,255,255,0)')
  g.fillStyle = lg
  g.fillRect(2, 29, 60, 6)
  const rg = g.createRadialGradient(32, 32, 0, 32, 32, 14)
  rg.addColorStop(0, 'rgba(255,255,255,1)')
  rg.addColorStop(1, 'rgba(255,220,238,0)')
  g.fillStyle = rg
  g.beginPath()
  g.arc(32, 32, 14, 0, Math.PI * 2)
  g.fill()
  return c
}

function buildSprites() {
  sprites = {
    hot: makeSprite('rgba(255,252,250,1)', 'rgba(255,185,212,0.95)'),
    mid: makeSprite('rgba(255,210,230,0.9)', 'rgba(200,75,130,0.75)'),
    edge: makeSprite('rgba(255,255,255,1)', 'rgba(255,180,214,0.9)'),
    star: makeStarSprite(),
    dust: makeSprite('rgba(255,255,255,0.9)', 'rgba(190,160,255,0.5)'),
    ring: makeSprite('rgba(255,255,255,0.95)', 'rgba(80,220,255,0.8)'),
    glow: makeSprite('rgba(160,70,160,0.5)', 'rgba(90,25,120,0.25)'),
    fogP: makeSprite('rgba(140,60,190,0.4)', 'rgba(80,30,130,0.18)'),   // 银河带用
    fogC: makeSprite('rgba(40,170,230,0.35)', 'rgba(20,90,150,0.15)'),  // 银河带用
  }
}

// 高清屏适配（两块画布都要）
function fitCanvas(canvas, ctx) {
  const dpr = window.devicePixelRatio || 1
  cw = canvas.clientWidth          // 先读真实布局尺寸
  ch = canvas.clientHeight
  canvas.width = cw * dpr          // 物理像素 = CSS 尺寸 × dpr
  canvas.height = ch * dpr
  ctx.setTransform(dpr, 0, 0, dpr, 0, 0)
}

function layout() {
  SCALE = Math.min(cw, ch) / 38
  cx0 = cw / 2
  cyH = ch * 0.46
}

// ---------- 背景层：一次性画好整片深空，之后每帧零成本 ----------
function buildBg() {
  const canvas = bgcv.value
  if (!canvas) return
  const ctx = canvas.getContext('2d')
  fitCanvas(canvas, ctx)
  const W = cw
  const H = ch

  // 深空底色：中央略亮的紫黑
  const bg = ctx.createRadialGradient(W / 2, H * 0.46, 0, W / 2, H * 0.46, Math.max(W, H) * 0.75)
  bg.addColorStop(0, '#0d0520')
  bg.addColorStop(1, '#030109')
  ctx.fillStyle = bg
  ctx.fillRect(0, 0, W, H)

  // 银河带：斜穿画面的星雾带（旋转坐标系里沿 x 轴撒微尘 + 团雾）
  ctx.save()
  ctx.translate(W / 2, H / 2)
  ctx.rotate(-0.32)
  for (let i = 0; i < 520; i++) {
    const x = (Math.random() - 0.5) * W * 1.5
    const gau = (Math.random() + Math.random() + Math.random() - 1.5) / 1.5
    const y = gau * H * 0.1
    const colors = ['255,255,255', '200,160,255', '140,210,255', '255,170,205']
    const c = colors[(Math.random() * colors.length) | 0]
    ctx.fillStyle = 'rgba(' + c + ',' + (0.03 + Math.random() * 0.09) + ')'
    const s = 0.6 + Math.random() * 1.6
    ctx.fillRect(x, y, s, s)
  }
  for (let i = 0; i < 4; i++) {
    const fr = Math.min(W, H) * (0.22 + Math.random() * 0.16)
    const x = (Math.random() - 0.5) * W * 0.9
    const y = (Math.random() - 0.5) * H * 0.12
    ctx.globalAlpha = 0.05
    ctx.drawImage(sprites.fogP, x - fr, y - fr, fr * 2, fr * 2)
    ctx.globalAlpha = 1
  }
  ctx.restore()

  // 固定星点：一小半带十字光芒
  for (let i = 0; i < 190; i++) {
    const x = Math.random() * W
    const y = Math.random() * H
    const r = 0.4 + Math.random() * 0.9
    const colors = ['255,255,255', '255,220,238', '190,225,255']
    ctx.fillStyle = 'rgba(' + colors[(Math.random() * 3) | 0] + ',' + (0.15 + Math.random() * 0.5) + ')'
    ctx.beginPath()
    ctx.arc(x, y, r, 0, Math.PI * 2)
    ctx.fill()
    if (Math.random() < 0.18) {
      ctx.fillStyle = 'rgba(255,255,255,0.25)'
      ctx.fillRect(x - r * 4, y - 0.4, r * 8, 0.8)
      ctx.fillRect(x - 0.4, y - r * 4, 0.8, r * 8)
    }
  }

  // 渐晕：四角压暗，视线聚焦中央
  const vg = ctx.createRadialGradient(W / 2, H / 2, Math.min(W, H) * 0.35, W / 2, H / 2, Math.max(W, H) * 0.72)
  vg.addColorStop(0, 'rgba(0,0,0,0)')
  vg.addColorStop(1, 'rgba(0,0,0,0.5)')
  ctx.fillStyle = vg
  ctx.fillRect(0, 0, W, H)
}

// ---------- 撒粒子 ----------
function buildParticles() {
  layout()
  particles = []
  const total = cw < 600 ? 1300 : 2400
  const zMax = SCALE * 6.5                 // 厚度加厚 → 更立体

  for (let i = 0; i < total; i++) {
    const t = Math.random() * Math.PI * 2
    const dir = heartDir(t)
    const R = heartR(t)
    const k = Math.pow(Math.random(), 0.5) * 1.05
    const layer = k > 0.9 ? 'edge' : k < 0.28 ? 'hot' : 'mid'

    const g3 = (Math.random() + Math.random() + Math.random() - 1.5) / 1.5
    const lz = g3 * zMax * (1 - 0.5 * Math.min(k, 1))

    const dv = randDir3()
    const dist = SCALE * (20 + Math.random() * 24)

    particles.push({
      t,                        // 心形线角度（能量脉络扫光用）
      lx: dir[0] * R * k * SCALE * 1.08,   // 横向加宽 8% → 更圆润饱满
      ly: dir[1] * R * k * SCALE,
      lz,
      layer,
      star: layer === 'hot' && Math.random() < 0.4,   // 少数核心粒子会变星芒
      sx3: dv[0] * dist,
      sy3: dv[1] * dist,
      sz3: dv[2] * dist,
      size: layer === 'hot' ? 2.4 + Math.random() * 2.2 : 1.6 + Math.random() * 2.2,
      tw: 0.003 + Math.random() * 0.005,
      group: (Math.random() * 8) | 0,
      phase: Math.random() * Math.PI * 2,
    })
  }

  // 轮廓流光
  comets = []
  for (let i = 0; i < 70; i++) {
    comets.push({
      base: Math.random() * Math.PI * 2,
      spd: 0.0011 + Math.random() * 0.0016,
      size: 2.5 + Math.random() * 2.5,
      phase: Math.random() * Math.PI * 2,
    })
  }

  // 星环
  ring = []
  const tilt = 0.42
  for (let i = 0; i < 220; i++) {
    ring.push({
      rr: SCALE * (20 + Math.random() * 9),
      base: Math.random() * Math.PI * 2,
      spd: 0.0004 + Math.random() * 0.0005,
      tilt,
      size: 1.5 + Math.random() * 2.2,
      phase: Math.random() * Math.PI * 2,
    })
  }

  // 星尘
  dust = []
  for (let i = 0; i < 26; i++) {
    dust.push({
      x: Math.random() * cw,
      y: Math.random() * ch,
      vy: 0.1 + Math.random() * 0.3,
      size: 1.5 + Math.random() * 2.5,
      phase: Math.random() * Math.PI * 2,
    })
  }
}

function smooth01(x) {
  const v = Math.max(0, Math.min(1, x))
  return v * v * (3 - 2 * v)
}

// 每帧画图（粒子层）
function draw(now) {
  const canvas = cv.value
  if (!canvas) return
  const ctx = canvas.getContext('2d')

  // ---------- 拖尾：destination-out 擦除法 ----------
  // 把已有内容按 30% 透明度"擦掉"→ 旧帧残影慢慢衰减，透明区保持透明（不遮背景层）
  ctx.globalCompositeOperation = 'destination-out'
  ctx.globalAlpha = 1
  ctx.fillStyle = 'rgba(0, 0, 0, 0.3)'
  ctx.fillRect(0, 0, cw, ch)

  // ---------- 读剧本 ----------
  const ct = now % CYCLE
  const g = ct < T_BURST ? smooth01(ct / T_GATHER) : 0
  const bu = ct >= T_BURST ? (ct - T_BURST) / (CYCLE - T_BURST) : 0
  const bE = 1 - Math.pow(1 - bu, 3)      // 全局消散缓动（场景级明暗/位置用）
  const heartOn = ct >= T_GATHER && ct < T_BURST
  const bob = Math.sin(now * 0.0009) * 8 * g * (1 - bE)

  // 能量脉络：一道亮波沿心形轮廓巡游（heartOn 时，约 3.3s 绕两圈）
  const waveP = heartOn ? ((now - T_GATHER) * 0.0019) % (Math.PI * 2) : -99

  // 全局姿态：恢复匀速慢转（~14s 一圈），立体感来自旋转 + 厚度 + 景深
  const A = now * 0.00045
  const swirl = (1 - g) * 2.4 - bE * 1.6
  const cAy = Math.cos(A + swirl)
  const sAy = Math.sin(A + swirl)
  const pit = Math.sin(now * 0.0004) * 0.04
  const cpit = Math.cos(pit)
  const spit = Math.sin(pit)
  const A2 = -now * 0.0008 + 0.9
  const cA2 = Math.cos(A2)
  const sA2 = Math.sin(A2)

  // ---------- 动态星尘 ----------
  ctx.globalCompositeOperation = 'source-over'
  for (const d of dust) {
    d.y -= d.vy
    if (d.y < -10) {
      d.y = ch + 10
      d.x = Math.random() * cw
    }
    ctx.globalAlpha = 0.1 + 0.08 * Math.sin(now * 0.001 + d.phase)
    ctx.drawImage(sprites.dust, d.x - d.size, d.y - d.size, d.size * 2, d.size * 2)
  }

  // ---------- 发光体进 lighter ----------
  ctx.globalCompositeOperation = 'lighter'

  // 心后紫晕
  const glowR = SCALE * 26
  ctx.globalAlpha = 0.08
  ctx.drawImage(sprites.glow, cx0 - glowR, cyH - glowR, glowR * 2, glowR * 2)

  // ---------- 心粒子 ----------
  for (const p of particles) {
    // 心的本地 3D：心跳脉冲 + 三轴微游动（活的云）
    const wx = Math.sin(now * 0.0011 + p.phase) * SCALE * 0.22
    const wy = Math.cos(now * 0.0009 + p.phase * 1.3) * SCALE * 0.22
    const wz = Math.sin(now * 0.0013 + p.phase * 0.7) * SCALE * 0.3
    const hx3 = p.lx + wx
    const hy3 = p.ly + wy
    const hz3 = p.lz + wz

    // 两段插值：星云⇄心
    // 烟花三波爆散：按 group 分三批，各延迟 0.16 份时间，连环炸开
    let bb = 0
    if (bu > 0) {
      const delay = (p.group % 3) * 0.16
      const raw = Math.min(1, Math.max(0, (bu - delay) / (1 - delay)))
      bb = 1 - Math.pow(1 - raw, 3)
    }
    let lx, ly, lz
    if (bb > 0) {
      lx = hx3 + (p.sx3 - hx3) * bb
      ly = hy3 + (p.sy3 - hy3) * bb
      lz = hz3 + (p.sz3 - hz3) * bb
    } else {
      lx = p.sx3 + (hx3 - p.sx3) * g
      ly = p.sy3 + (hy3 - p.sy3) * g
      lz = p.sz3 + (hz3 - p.sz3) * g
    }

    // 呼吸俯仰
    const py2 = ly * cpit - lz * spit
    const pz2 = ly * spit + lz * cpit

    // 绕竖直轴旋转（含螺旋偏角）→ 透视
    const xa = lx * cAy + pz2 * sAy
    const za = -lx * sAy + pz2 * cAy
    const persp = FOV / (FOV + za)
    const sx = cx0 + xa * persp
    const sy = cyH + bob + py2 * persp

    const depth = Math.max(0, Math.min(1, za / (SCALE * 10) + 0.5))

    // 能量脉络扫光：波前经过时粒子瞬间提亮
    let near = 0
    if (heartOn && p.layer !== 'hot') {
      const m = ((p.t - waveP) % (Math.PI * 2) + Math.PI * 2) % (Math.PI * 2)
      const d = Math.min(m, Math.PI * 2 - m)
      near = Math.exp(-d * d * 6)
    }

    let a = bb > 0 ? (1 - bb) * 0.85 : 0.3 + g * 0.6
    const patch = p.layer === 'edge' ? 1 : 0.72 + 0.28 * Math.sin(now * 0.0012 + p.group * 1.7)
    const sparkle = 0.35 + 0.65 * (0.5 + 0.5 * Math.sin(now * p.tw + p.phase * 3))
    a *= sparkle * patch
    a *= 1.25 - depth * 0.7
    a *= 1 + near * 1.5        // 扫过 = 爆亮
    if (p.layer === 'hot') a *= 1.4
    if (p.layer === 'mid') a *= 0.72
    if (p.layer === 'edge') a *= 1.3

    let ss = p.size * persp * (0.55 + 0.45 * g) * (1 + bb * 0.5)
    ss *= 1 + near * 0.3

    // 星芒变身：sparkle 冲上峰值时，个别核心粒子展开十字星芒
    let img = sprites[p.layer]
    if (p.star && sparkle > 0.86 && heartOn) {
      img = sprites.star
      ss = p.size * 2.6 * persp
      a *= 1.6
    }

    ctx.globalAlpha = Math.max(0, Math.min(a, 1))
    ctx.drawImage(img, sx - ss, sy - ss, ss * 2, ss * 2)
  }

  // ---------- 轮廓流光 ----------
  for (const m of comets) {
    const tc = m.base + now * m.spd
    const dir = heartDir(tc)
    const rr = heartR(tc) * SCALE
    const lx = dir[0] * rr * 1.08
    const ly = dir[1] * rr
    const py2 = ly * cpit
    const pz2 = ly * spit
    const xa = lx * cAy + pz2 * sAy
    const za = -lx * sAy + pz2 * cAy
    const persp = FOV / (FOV + za)
    const sx = cx0 + xa * persp
    const sy = cyH + bob + py2 * persp

    const a = (0.35 + 0.45 * Math.sin(now * 0.004 + m.phase) ** 2) * g * (1 - bE)
    const ss = m.size * persp
    ctx.globalAlpha = Math.max(0, Math.min(a, 1))
    ctx.drawImage(sprites.edge, sx - ss, sy - ss, ss * 2, ss * 2)
  }

  // ---------- 星环 ----------
  for (const q of ring) {
    const th = q.base + now * q.spd
    const ex = q.rr * Math.cos(th)
    const ez0 = q.rr * Math.sin(th)
    const ey = -ez0 * Math.sin(q.tilt)
    const ez = ez0 * Math.cos(q.tilt)
    const xr = ex * cA2 + ez * sA2
    const zr = -ex * sA2 + ez * cA2

    const persp = FOV / (FOV + zr)
    const sx = cx0 + xr * persp
    const sy = cyH + bob + ey * persp

    const depth = Math.max(0, Math.min(1, zr / (SCALE * 26) + 0.5))
    let a = (0.5 + 0.25 * Math.sin(now * 0.002 + q.phase)) * (1.2 - depth * 0.6)
    a *= Math.min(1, g + 0.25) * (1 - bE)

    ctx.globalAlpha = Math.max(0, Math.min(a, 1))
    ctx.drawImage(sprites.ring, sx - q.size, sy - q.size, q.size * 2, q.size * 2)
  }

  // ---------- 消散瞬间的全屏闪光 ----------
  if (bu > 0 && bu < 0.1) {
    ctx.globalAlpha = (1 - bu / 0.1) * 0.15
    ctx.fillStyle = '#ffd9ec'
    ctx.fillRect(0, 0, cw, ch)
  }

  ctx.globalAlpha = 1
  rafId = requestAnimationFrame(draw)
}

// 窗口变化：两块画布重铺 + 场景重撒 + 背景重画
function onResize() {
  const canvas = cv.value
  // 守卫：面板折叠/抖动瞬间 clientWidth 可能是 0，此时绝不能重建画布（会打成 0×0 全黑）
  if (!canvas || canvas.clientWidth === 0 || canvas.clientHeight === 0) return
  fitCanvas(canvas, canvas.getContext('2d'))
  buildParticles()
  buildBg()
}

// ---------- 启动：等 CSS 布局就位再初始化 ----------
// Vite 的 scoped CSS 是异步注入的，onMounted 那一刻 clientWidth 可能还是 0
// （画布会被建成 0×0 → 全黑）。所以每帧探测，拿到真实尺寸才开工
let bootRaf = 0
function boot() {
  const canvas = cv.value
  if (!canvas || canvas.clientWidth === 0 || canvas.clientHeight === 0) {
    bootRaf = requestAnimationFrame(boot)   // 还没布局好，下一帧再试
    return
  }
  fitCanvas(canvas, canvas.getContext('2d'))
  buildSprites()
  buildParticles()
  buildBg()
  rafId = requestAnimationFrame(draw)
}

onMounted(() => {
  bootRaf = requestAnimationFrame(boot)
  window.addEventListener('resize', onResize)
})

onUnmounted(() => {
  cancelAnimationFrame(bootRaf)
  cancelAnimationFrame(rafId)
  window.removeEventListener('resize', onResize)
})
</script>

<style scoped src="@/views/css/HeartView.css"></style>

<template>
  <div class="heart-page">
    <!-- 一整屏深空舞台：背景层(静态深空) + 粒子层(拖尾发光体) 叠放 -->
    <div class="heart-stage">
      <canvas ref="bgcv" class="bg-canvas"></canvas>
      <canvas ref="cv" class="heart-canvas"></canvas>
      <p class="heart-tip">星尘之心 · 12 秒轮回 🎞️</p>
    </div>
  </div>
</template>
