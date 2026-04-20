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
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	W = 400
	H = 600
	StarCount = 100
	ShieldDuration = 300 // 5 секунд при 60 FPS
)

type Object struct {
	X, Y   float32
	VX, VY float32
	Life   int
}

type Star struct {
	X, Y   float32
	Size   float32
	Speed  float32
	Bright uint8
}

// Бонус (Щит)
type PowerUp struct {
	X, Y  float32
	VY    float32
	Active bool
	Type  string // "shield"
}

type Game struct {
	pX, pY    float32
	bullets   []Object
	enemies   []Object
	particles []Object
	stars     []Star
	powerups  []PowerUp // Список активных бонусов
	
	timer     int
	score     int
	frame     int
	isPaused  bool
	
	// Механика щита
	hasShield   bool
	shieldTimer int
}

func (g *Game) Init() {
	g.stars = make([]Star, StarCount)
	for i := 0; i < StarCount; i++ {
		g.stars[i] = Star{
			X:      rand.Float32() * W,
			Y:      rand.Float32() * H,
			Size:   rand.Float32()*1.5 + 0.5,
			Speed:  rand.Float32()*2 + 0.5,
			Bright: uint8(rand.Intn(100) + 155),
		}
	}
	g.powerups = []PowerUp{}
}

func (g *Game) Update() error {
	g.frame++

	// Пауза
	if inpututil.IsKeyJustPressed(ebiten.KeyP) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		g.isPaused = !g.isPaused
	}

	// --- ЛОГИКА ЗВЕЗД ---
	for i := range g.stars {
		s := &g.stars[i]
		s.Y += s.Speed
		if s.Y > H {
			s.Y = -5
			s.X = rand.Float32() * W
		}
	}

	if g.isPaused {
		return nil
	}

	// --- ТАЙМЕР ЩИТА ---
	if g.hasShield {
		g.shieldTimer--
		if g.shieldTimer <= 0 {
			g.hasShield = false
		}
	}

	// --- СПАВН БОНУСОВ ---
	// Шанс 1 к 1000 каждый кадр (примерно раз в 15-20 секунд)
	if rand.Intn(1000) == 0 {
		g.powerups = append(g.powerups, PowerUp{
			X: rand.Float32()*(W-40) + 20,
			Y: -20,
			VY: 2,
			Active: true,
			Type: "shield",
		})
	}

	// --- ОБНОВЛЕНИЕ БОНУСОВ ---
	for i := len(g.powerups) - 1; i >= 0; i-- {
		p := &g.powerups[i]
		p.Y += p.VY
		
		// Проверка подбора бонуса игроком
		dx := p.X - g.pX
		dy := p.Y - g.pY
		dist := dx*dx + dy*dy
		
		if dist < 900 { // Радиус 30 (как у игрока)
			if p.Type == "shield" {
				g.hasShield = true
				g.shieldTimer = ShieldDuration
				g.createExplosion(g.pX, g.pY, color.RGBA{0, 100, 255, 255}) // Эффект получения
			}
			// Удаляем бонус
			g.powerups[i] = g.powerups[len(g.powerups)-1]
			g.powerups = g.powerups[:len(g.powerups)-1]
			continue
		}

		// Удаление если улетел за экран
		if p.Y > H+20 {
			g.powerups[i] = g.powerups[len(g.powerups)-1]
			g.powerups = g.powerups[:len(g.powerups)-1]
		}
	}

	// --- ИГРОВАЯ ЛОГИКА ---

	// 1. Управление
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

	// 4. Пули
	for i := len(g.bullets) - 1; i >= 0; i-- {
		g.bullets[i].Y -= 10
		if g.bullets[i].Y < -10 {
			g.bullets[i] = g.bullets[len(g.bullets)-1]
			g.bullets = g.bullets[:len(g.bullets)-1]
		}
	}

	// 5. Враги и коллизии
	for i := len(g.enemies) - 1; i >= 0; i-- {
		e := &g.enemies[i]
		e.Y += e.VY

		dx := e.X - g.pX
		dy := e.Y - g.pY
		distPlayer := dx*dx + dy*dy

		// Столкновение с игроком
		if distPlayer < 900 {
			if g.hasShield {
				// Щит активен: уничтожаем врага, щит остается (или можно отнимать время)
				g.createExplosion(e.X, e.Y, color.RGBA{255, 50, 50, 255})
				g.enemies[i] = g.enemies[len(g.enemies)-1]
				g.enemies = g.enemies[:len(g.enemies)-1]
				
				// Опционально: щит мигает или тратится. 
				// Давай сделаем так: щит защищает бесконечно долго по времени, но визуально видно.
				continue 
			} else {
				// Щита нет: Game Over / Сброс очков
				g.createExplosion(g.pX, g.pY, color.RGBA{0, 255, 150, 255})
				g.score = 0
				g.enemies = nil
				g.bullets = nil
				continue
			}
		}

		// Столкновение с пулями
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

	// 6. Частицы
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

	// Звезды
	for _, s := range g.stars {
		c := color.RGBA{200, 200, 255, s.Bright}
		vector.DrawFilledRect(screen, s.X, s.Y, s.Size, s.Size, c, false)
	}

	// Бонусы (Щиты)
	for _, p := range g.powerups {
		if p.Type == "shield" {
			vector.DrawFilledCircle(screen, p.X, p.Y, 10, color.RGBA{0, 100, 255, 255}, false)
			vector.StrokeCircle(screen, p.X, p.Y, 14, 2, color.RGBA{0, 200, 255, 255}, false)
		}
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
	playerColor := color.RGBA{0, 255, 150, 255}
	
	// Если есть щит, рисуем защитное поле и бар
	if g.hasShield {
		// Прозрачный синий круг вокруг игрока
		vector.StrokeCircle(screen, g.pX, g.pY, 25, 3, color.RGBA{0, 150, 255, 200}, false)
		
		// --- ИСПРАВЛЕНИЕ ТИПОВ ---
		var barWidth float32 = 40.0
		var barHeight float32 = 4.0
		ratio := float32(g.shieldTimer) / float32(ShieldDuration)
		
		// Фон бара (серый)
		vector.DrawFilledRect(screen, g.pX - barWidth/2, g.pY - 35, barWidth, barHeight, color.RGBA{50, 50, 50, 200}, false)
		// Заполнение бара (синее)
		vector.DrawFilledRect(screen, g.pX - barWidth/2, g.pY - 35, barWidth*ratio, barHeight, color.RGBA{0, 200, 255, 255}, false)
	}

	vector.DrawFilledCircle(screen, g.pX, g.pY, 15, playerColor, false)
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
		vector.DrawFilledRect(screen, 0, 0, W, H, color.RGBA{0, 0, 0, 100}, false)
		ebitenutil.DebugPrintAt(screen, "PAUSED", W/2-30, H/2)
		ebitenutil.DebugPrintAt(screen, "Press P to Resume", W/2-50, H/2+20)
	}
	
	ebitenutil.DebugPrintAt(screen, scoreText, 10, 10)
	ebitenutil.DebugPrintAt(screen, "Collect Blue Orbs for Shield!", 10, 30)
}

func (g *Game) Layout(w, h int) (int, int) { return W, H }

func main() {
	rand.Seed(time.Now().UnixNano())
	ebiten.SetWindowTitle("SKY FORCE - SHIELD UPDATE")
	ebiten.SetWindowSize(W, H)
	
	game := &Game{pX: W / 2, pY: H - 100}
	game.Init()
	
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}