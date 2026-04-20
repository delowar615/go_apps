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
	
	// Настройки щита
	ShieldDuration = 300 // 5 сек при 60 FPS
	
	// Настройки лазера
	LaserMaxEnergy = 180 // 3 секунды энергии (60 * 3)
	LaserRechargeRate = 12 // Скорость восстановления (полная зарядка за ~15 сек)
	LaserDrainRate = 3     // Скорость траты энергии
	LaserWidth = 10        // Базовая ширина луча
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

type PowerUp struct {
	X, Y     float32
	VY       float32
	Active   bool
	Type     string // "shield" или "laser"
}

type Game struct {
	pX, pY    float32
	bullets   []Object
	enemies   []Object
	particles []Object
	stars     []Star
	powerups  []PowerUp
	
	timer     int
	score     int
	frame     int
	isPaused  bool
	
	// Щит
	hasShield   bool
	shieldTimer int
	
	// Лазер
	laserEnergy   int   // Текущая энергия (0 - LaserMaxEnergy)
	isLaserActive bool  // Активен ли луч прямо сейчас
	hasLaserUpgrade bool // Есть ли вообще улучшение лазера
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
	g.laserEnergy = LaserMaxEnergy // Начинаем с полным зарядом для теста, или 0
	g.hasLaserUpgrade = true // Для теста сразу дадим лазер, или можно спавнить бонус
}

func (g *Game) Update() error {
	g.frame++

	// Пауза (P или Esc)
	if inpututil.IsKeyJustPressed(ebiten.KeyP) || inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
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

	// --- УПРАВЛЕНИЕ ЛАЗЕРОМ ---
	// Лазер работает, если есть энергия И игрок держит кнопку (ЛКМ или Пробел)
	mouseBtn := ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)
	keyBtn := ebiten.IsKeyPressed(ebiten.KeySpace)
	
	if g.hasLaserUpgrade && g.laserEnergy > 0 && (mouseBtn || keyBtn) {
		g.isLaserActive = true
		g.laserEnergy -= LaserDrainRate
		if g.laserEnergy < 0 { g.laserEnergy = 0 }
	} else {
		g.isLaserActive = false
		// Перезарядка
		if g.laserEnergy < LaserMaxEnergy {
			g.laserEnergy += LaserRechargeRate
			if g.laserEnergy > LaserMaxEnergy { g.laserEnergy = LaserMaxEnergy }
		}
	}

	// --- СПАВН БОНУСОВ ---
	// Шанс спавна бонуса (раз в ~10-15 секунд)
	if rand.Intn(600) == 0 {
		typeRoll := rand.Intn(2)
		pType := "shield"
		if typeRoll == 1 { pType = "laser" }
		
		g.powerups = append(g.powerups, PowerUp{
			X: rand.Float32()*(W-40) + 20,
			Y: -20,
			VY: 2,
			Active: true,
			Type: pType,
		})
	}

	// --- ОБНОВЛЕНИЕ БОНУСОВ ---
	for i := len(g.powerups) - 1; i >= 0; i-- {
		p := &g.powerups[i]
		p.Y += p.VY
		
		dx := p.X - g.pX
		dy := p.Y - g.pY
		dist := dx*dx + dy*dy
		
		if dist < 900 { // Подбор
			if p.Type == "shield" {
				g.hasShield = true
				g.shieldTimer = ShieldDuration
				g.createExplosion(g.pX, g.pY, color.RGBA{0, 100, 255, 255})
			} else if p.Type == "laser" {
				g.hasLaserUpgrade = true
				g.laserEnergy = LaserMaxEnergy // Полная зарядка при подборе
				g.createExplosion(g.pX, g.pY, color.RGBA{0, 255, 255, 255})
			}
			
			g.powerups[i] = g.powerups[len(g.powerups)-1]
			g.powerups = g.powerups[:len(g.powerups)-1]
			continue
		}

		if p.Y > H+20 {
			g.powerups[i] = g.powerups[len(g.powerups)-1]
			g.powerups = g.powerups[:len(g.powerups)-1]
		}
	}

	// --- ИГРОВАЯ ЛОГИКА ---

	// 1. Управление кораблем
	mx, my := ebiten.CursorPosition()
	targetX, targetY := float32(mx), float32(my)
	g.pX += (targetX - g.pX) * 0.2
	g.pY += (targetY - g.pY) * 0.2

	if g.pX < 0 { g.pX = 0 }
	if g.pX > W { g.pX = W }
	if g.pY < 0 { g.pY = 0 }
	if g.pY > H { g.pY = H }

	// 2. Стрельба обычными пулями (автоматическая)
	// Если лазер активен, обычные пули можно отключить или оставить. Оставим для плотности огня.
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

	// 5. Враги и коллизии
	// Сначала проверяем ЛАЗЕР, так как он приоритетнее (мгновенное убийство)
	if g.isLaserActive {
		laserLeft := g.pX - float32(LaserWidth)/2
		laserRight := g.pX + float32(LaserWidth)/2
		
		for i := len(g.enemies) - 1; i >= 0; i-- {
			e := &g.enemies[i]
			// Простая проверка: центр врага внутри ширины лазера
			if e.X > laserLeft && e.X < laserRight {
				g.createExplosion(e.X, e.Y, color.RGBA{0, 255, 255, 255}) // Циановый взрыв
				g.score += 50 // Меньше очков за лазер, так как легко
				g.enemies[i] = g.enemies[len(g.enemies)-1]
				g.enemies = g.enemies[:len(g.enemies)-1]
			}
		}
	}

	// Затем обычные столкновения
	for i := len(g.enemies) - 1; i >= 0; i-- {
		e := &g.enemies[i]
		e.Y += e.VY

		dx := e.X - g.pX
		dy := e.Y - g.pY
		distPlayer := dx*dx + dy*dy

		// Столкновение с игроком
		if distPlayer < 900 {
			if g.hasShield {
				g.createExplosion(e.X, e.Y, color.RGBA{255, 50, 50, 255})
				g.enemies[i] = g.enemies[len(g.enemies)-1]
				g.enemies = g.enemies[:len(g.enemies)-1]
				continue 
			} else {
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

	// --- РИСУЕМ ЛАЗЕР (ЕСЛИ АКТИВЕН) ---
	if g.isLaserActive {
		// Эффект дрожания ширины
		jitter := float32(rand.Intn(3)) - 1.5 // от -1.5 до 1.5
		currentWidth := float32(LaserWidth) + jitter
		
		// 1. Аура (Широкая, полупрозрачная, бирюзовая)
		auraWidth := currentWidth * 3.0
		auraColor := color.RGBA{0, 255, 255, 100} // Cyan transparent
		vector.DrawFilledRect(screen, g.pX - auraWidth/2, 0, auraWidth, float32(H), auraColor, false)
		
		// 2. Основной луч (Белый, каленый)
		coreColor := color.RGBA{255, 255, 255, 255}
		vector.DrawFilledRect(screen, g.pX - currentWidth/2, 0, currentWidth, float32(H), coreColor, false)
	}

	// Бонусы
	for _, p := range g.powerups {
		if p.Type == "shield" {
			vector.DrawFilledCircle(screen, p.X, p.Y, 10, color.RGBA{0, 100, 255, 255}, false)
			vector.StrokeCircle(screen, p.X, p.Y, 14, 2, color.RGBA{0, 200, 255, 255}, false)
		} else if p.Type == "laser" {
			// Иконка лазера (молния или кристалл)
			vector.DrawFilledRect(screen, p.X-5, p.Y-10, 10, 20, color.RGBA{0, 255, 255, 255}, false)
			vector.StrokeRect(screen, p.X-7, p.Y-12, 14, 24, 2, color.White, false)
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
	
	// Щит
	if g.hasShield {
		vector.StrokeCircle(screen, g.pX, g.pY, 25, 3, color.RGBA{0, 150, 255, 200}, false)
		
		var barWidth float32 = 40.0
		var barHeight float32 = 4.0
		ratio := float32(g.shieldTimer) / float32(ShieldDuration)
		
		vector.DrawFilledRect(screen, g.pX - barWidth/2, g.pY - 35, barWidth, barHeight, color.RGBA{50, 50, 50, 200}, false)
		vector.DrawFilledRect(screen, g.pX - barWidth/2, g.pY - 35, barWidth*ratio, barHeight, color.RGBA{0, 200, 255, 255}, false)
	}

	// Энергия Лазера (UI под игроком)
	if g.hasLaserUpgrade {
		var barWidth float32 = 60.0
		var barHeight float32 = 6.0
		ratio := float32(g.laserEnergy) / float32(LaserMaxEnergy)
		
		// Фон бара энергии
		vector.DrawFilledRect(screen, g.pX - barWidth/2, g.pY + 25, barWidth, barHeight, color.RGBA{50, 50, 50, 200}, false)
		
		// Цвет бара зависит от состояния
		barColor := color.RGBA{0, 255, 255, 255} // Cyan
		if g.isLaserActive {
			barColor = color.RGBA{255, 255, 255, 255} // White hot when firing
		} else if ratio < 0.2 {
			barColor = color.RGBA{255, 50, 50, 255} // Red if low
		}
		
		vector.DrawFilledRect(screen, g.pX - barWidth/2, g.pY + 25, barWidth*ratio, barHeight, barColor, false)
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
	
	// Подсказки
	if !g.hasLaserUpgrade {
		ebitenutil.DebugPrintAt(screen, "Find Laser Upgrade!", 10, 30)
	} else {
		ebitenutil.DebugPrintAt(screen, "Hold LMB/Space for LASER", 10, 30)
	}
}

func (g *Game) Layout(w, h int) (int, int) { return W, H }

func main() {
	rand.Seed(time.Now().UnixNano())
	ebiten.SetWindowTitle("SKY FORCE - LASER BEAM")
	ebiten.SetWindowSize(W, H)
	
	game := &Game{pX: W / 2, pY: H - 100}
	game.Init()
	
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}