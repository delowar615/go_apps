package main

import (
	"fmt"
	"image/color"
	"log"
	"math"
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil" // Добавлено для отслеживания нажатий
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	W = 400
	H = 600
)

type Object struct {
	X, Y   float32
	VX, VY float32
	Life   int
}

type Game struct {
	pX, pY    float32
	bullets   []Object
	enemies   []Object
	particles []Object
	timer     int
	score     int
	frame     int
	isPaused  bool // Флаг паузы
}

func (g *Game) Update() error {
	g.frame++

	// Обработка паузы по нажатию клавиши P или Space
	if inpututil.IsKeyJustPressed(ebiten.KeyP) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		g.isPaused = !g.isPaused
	}

	// Если пауза активна, пропускаем всю игровую логику
	if g.isPaused {
		return nil
	}

	// --- ИГРОВАЯ ЛОГИКА (работает только если не пауза) ---

	// 1. Управление (плавное следование за мышью)
	mx, my := ebiten.CursorPosition()
	targetX, targetY := float32(mx), float32(my)
	
	g.pX += (targetX - g.pX) * 0.2
	g.pY += (targetY - g.pY) * 0.2

	if g.pX < 0 { g.pX = 0 }
	if g.pX > W { g.pX = W }
	if g.pY < 0 { g.pY = 0 }
	if g.pY > H { g.pY = H }

	// 2. Стрельба
	g.timer++
	if g.timer > 10 {
		g.bullets = append(g.bullets, Object{X: g.pX, Y: g.pY - 20})
		g.timer = 0
	}

	// 3. Спавн врагов
	if rand.Intn(50) == 0 {
		g.enemies = append(g.enemies, Object{
			X: rand.Float32()*(W-40) + 20, 
			Y: -20,
			VY: 2 + rand.Float32()*2,
		})
	}

	// 4. Обновление пуль
	for i := len(g.bullets) - 1; i >= 0; i-- {
		g.bullets[i].Y -= 10
		if g.bullets[i].Y < -10 {
			g.bullets[i] = g.bullets[len(g.bullets)-1]
			g.bullets = g.bullets[:len(g.bullets)-1]
		}
	}

	// 5. Обновление врагов и коллизии
	for i := len(g.enemies) - 1; i >= 0; i-- {
		e := &g.enemies[i]
		e.Y += e.VY

		dx := e.X - g.pX
		dy := e.Y - g.pY
		if dx*dx+dy*dy < 900 {
			g.createExplosion(g.pX, g.pY, color.RGBA{0, 255, 150, 255})
			g.score = 0
			g.enemies = nil
			g.bullets = nil
			continue
		}

		hit := false
		for j := len(g.bullets) - 1; j >= 0; j-- {
			b := &g.bullets[j]
			bdx := e.X - b.X
			bdy := e.Y - b.Y
			if bdx*bdx+bdy*bdy < 400 {
				g.createExplosion(e.X, e.Y, color.RGBA{255, 50, 50, 255})
				g.bullets[j] = g.bullets[len(g.bullets)-1]
				g.bullets = g.bullets[:len(g.bullets)-1]
				hit = true
				g.score += 100
				break
			}
		}

		if hit {
			g.enemies[i] = g.enemies[len(g.enemies)-1]
			g.enemies = g.enemies[:len(g.enemies)-1]
			continue
		}

		if e.Y > H+20 {
			g.enemies[i] = g.enemies[len(g.enemies)-1]
			g.enemies = g.enemies[:len(g.enemies)-1]
		}
	}

	// 6. Частицы (можно оставить обновляться даже на паузе для красоты, или тоже остановить)
	// Сейчас они останавливаются вместе с игрой, так как код выше не выполняется.
	// Если хочешь, чтобы частицы долетали, вынеси этот цикл из-под if !g.isPaused
	
	for i := len(g.particles) - 1; i >= 0; i-- {
		p := &g.particles[i]
		p.X += p.VX
		p.Y += p.VY
		p.Life--
		if p.Life <= 0 {
			g.particles[i] = g.particles[len(g.particles)-1]
			g.particles = g.particles[:len(g.particles)-1]
		}
	}

	return nil
}

func (g *Game) createExplosion(x, y float32, c color.RGBA) {
	for k := 0; k < 8; k++ {
		angle := rand.Float32() * 6.28
		speed := 2 + rand.Float32()*3
		g.particles = append(g.particles, Object{
			X: x, Y: y,
			VX: float32(math.Cos(float64(angle))) * speed,
			VY: float32(math.Sin(float64(angle))) * speed,
			Life: 20 + rand.Intn(10),
		})
	}
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{5, 5, 20, 255})

	// Звезды (статичные или мерцающие, тут простые)
	for i := 0; i < 50; i++ {
		x := float32(rand.Intn(W))
		y := float32(rand.Intn(H))
		vector.DrawFilledRect(screen, x, y, 1, 1, color.RGBA{100, 100, 150, 100}, false)
	}

	// Пули
	for _, b := range g.bullets {
		vector.DrawFilledRect(screen, b.X-2, b.Y, 4, 12, color.RGBA{255, 255, 0, 255}, false)
	}

	// Враги
	for _, e := range g.enemies {
		vector.DrawFilledCircle(screen, e.X, e.Y, 12, color.RGBA{255, 50, 50, 255}, false)
		vector.DrawFilledRect(screen, e.X-6, e.Y-4, 4, 4, color.Black, false)
		vector.DrawFilledRect(screen, e.X+2, e.Y-4, 4, 4, color.Black, false)
	}

	// Игрок
	vector.DrawFilledCircle(screen, g.pX, g.pY, 15, color.RGBA{0, 255, 150, 255}, false)
	vector.DrawFilledRect(screen, g.pX-4, g.pY+10, 8, 10, color.RGBA{0, 100, 255, 200}, false)

	// Частицы
	for _, p := range g.particles {
		alpha := uint8(float32(p.Life) / 30.0 * 255.0)
		c := color.RGBA{255, 200, 50, alpha}
		vector.DrawFilledRect(screen, p.X-2, p.Y-2, 4, 4, c, false)
	}

	// Интерфейс
	scoreText := fmt.Sprintf("SCORE: %d", g.score)
	if g.isPaused {
		scoreText += " | PAUSED"
		// Затемнение экрана при паузе
		vector.DrawFilledRect(screen, 0, 0, W, H, color.RGBA{0, 0, 0, 100}, false)
		
		// Надпись PAUSED по центру
		ebitenutil.DebugPrintAt(screen, "PAUSED", W/2-30, H/2)
		ebitenutil.DebugPrintAt(screen, "Press P to Resume", W/2-50, H/2+20)
	}
	
	ebitenutil.DebugPrintAt(screen, scoreText, 10, 10)
}

func (g *Game) Layout(w, h int) (int, int) { return W, H }

func main() {
	rand.Seed(time.Now().UnixNano())
	ebiten.SetWindowTitle("SKY FORCE - REPTILOID HUNTER")
	ebiten.SetWindowSize(W, H)
	if err := ebiten.RunGame(&Game{pX: W / 2, pY: H - 100}); err != nil {
		log.Fatal(err)
	}
}