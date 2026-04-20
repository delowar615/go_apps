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
	ShieldDuration = 300 
	
	// Настройки лазера
	LaserMaxEnergy = 180 
	LaserRechargeRate = 12 
	LaserDrainRate = 3     
	LaserWidth = 10        
	
	// Настройки Босса
	BossSpawnScore = 2000 
	BossBaseHP = 500      
	BossSpeed = 2.0       
)

type Object struct {
	X, Y   float32
	VX, VY float32
	Life   int
	Color  color.RGBA 
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
	Type     string 
}

type Boss struct {
	X, Y     float32
	HP       int
	MaxHP    int
	Dir      float32 
	Timer    int     
	Active   bool
}

type Game struct {
	pX, pY    float32
	bullets   []Object
	enemies   []Object
	particles []Object
	stars     []Star
	powerups  []PowerUp
	boss      Boss
	
	timer     int
	score     int
	frame     int
	isPaused  bool
	
	hasShield   bool
	shieldTimer int
	
	laserEnergy   int   
	isLaserActive bool  
	hasLaserUpgrade bool 
	
	nextBossScore int
}

func (g *Game) Init() {
	g.stars = make([]Star, StarCount)
	for i := 0; i < StarCount; i++ {
		g.stars[i] = Star{
			X: rand.Float32() * W, Y: rand.Float32() * H,
			Size: rand.Float32()*1.5 + 0.5, Speed: rand.Float32()*2 + 0.5,
			Bright: uint8(rand.Intn(100) + 155),
		}
	}
	g.powerups = []PowerUp{}
	g.laserEnergy = LaserMaxEnergy
	g.hasLaserUpgrade = true
	g.boss = Boss{Active: false}
	g.nextBossScore = BossSpawnScore
}

func (g *Game) Update() error {
	g.frame++

	if inpututil.IsKeyJustPressed(ebiten.KeyP) || inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		g.isPaused = !g.isPaused
	}

	// Звезды
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

	// --- ЛОГИКА БОССА ---
	if !g.boss.Active && g.score >= g.nextBossScore {
		g.boss.Active = true
		g.boss.MaxHP = BossBaseHP + (g.score / 1000) * 100
		g.boss.HP = g.boss.MaxHP
		g.boss.X = W / 2
		g.boss.Y = 80
		g.boss.Dir = 1
		g.enemies = nil
	}

	if g.boss.Active {
		b := &g.boss
		b.X += BossSpeed * b.Dir
		if b.X > W-60 || b.X < 60 {
			b.Dir *= -1
		}

		b.Timer++
		if b.Timer > 40 {
			for angle := -0.3; angle <= 0.3; angle += 0.3 {
				g.bullets = append(g.bullets, Object{
					X: b.X, Y: b.Y + 40,
					VX: float32(angle) * 6, VY: 7,
					Color: color.RGBA{255, 50, 50, 255},
				})
			}
			b.Timer = 0
		}
	}

	// --- ТАЙМЕР ЩИТА ---
	if g.hasShield {
		g.shieldTimer--
		if g.shieldTimer <= 0 {
			g.hasShield = false
		}
	}

	// --- ЛАЗЕР ---
	mouseBtn := ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)
	keyBtn := ebiten.IsKeyPressed(ebiten.KeySpace)
	
	if g.hasLaserUpgrade && g.laserEnergy > 0 && (mouseBtn || keyBtn) {
		g.isLaserActive = true
		g.laserEnergy -= LaserDrainRate
		if g.laserEnergy < 0 {
			g.laserEnergy = 0
		}
	} else {
		g.isLaserActive = false
		if g.laserEnergy < LaserMaxEnergy {
			g.laserEnergy += LaserRechargeRate
			if g.laserEnergy > LaserMaxEnergy {
				g.laserEnergy = LaserMaxEnergy
			}
		}
	}

	// --- СПАВН ОБЫЧНЫХ ВРАГОВ ---
	if !g.boss.Active {
		if rand.Intn(50) == 0 {
			g.enemies = append(g.enemies, Object{
				X: rand.Float32()*(W-40) + 20, Y: -20, VY: 2 + rand.Float32()*2,
			})
		}
		
		if rand.Intn(600) == 0 {
			typeRoll := rand.Intn(2)
			pType := "shield"
			if typeRoll == 1 {
				pType = "laser"
			}
			g.powerups = append(g.powerups, PowerUp{
				X: rand.Float32()*(W-40) + 20, Y: -20, VY: 2, Type: pType,
			})
		}
	}

	// --- ОБНОВЛЕНИЕ ПУЛЬ ---
	for i := len(g.bullets) - 1; i >= 0; i-- {
		b := &g.bullets[i]
		b.X += b.VX
		b.Y += b.VY
		
		if b.Y > H+20 || b.Y < -20 || b.X < -20 || b.X > W+20 {
			// Проверка на пустой слайс перед удалением
			if len(g.bullets) > 0 {
				g.bullets[i] = g.bullets[len(g.bullets)-1]
				g.bullets = g.bullets[:len(g.bullets)-1]
			}
			continue
		}

		// Пуля игрока vs Босс
		if g.boss.Active && b.Color.R == 255 && b.Color.G == 255 && b.Color.B == 0 {
			dx := b.X - g.boss.X
			dy := b.Y - g.boss.Y
			if dx*dx + dy*dy < 2500 {
				g.boss.HP -= 10
				g.createExplosion(b.X, b.Y, color.RGBA{255, 255, 0, 255})
				
				if len(g.bullets) > 0 {
					g.bullets[i] = g.bullets[len(g.bullets)-1]
					g.bullets = g.bullets[:len(g.bullets)-1]
				}
				
				if g.boss.HP <= 0 {
					g.killBoss()
				}
				continue
			}
		}
		
		// Пуля босса vs Игрок
		if b.Color.R == 255 && b.Color.G < 100 { 
			dx := b.X - g.pX
			dy := b.Y - g.pY
			if dx*dx + dy*dy < 400 {
				if !g.hasShield {
					g.gameOverReset()
				} else {
					g.hasShield = false
					g.createExplosion(g.pX, g.pY, color.RGBA{0, 100, 255, 255})
				}
				
				if len(g.bullets) > 0 {
					g.bullets[i] = g.bullets[len(g.bullets)-1]
					g.bullets = g.bullets[:len(g.bullets)-1]
				}
				continue
			}
		}
	}

	// --- ЛАЗЕР VS БОСС ---
	if g.isLaserActive && g.boss.Active {
		laserLeft := g.pX - float32(LaserWidth)/2
		laserRight := g.pX + float32(LaserWidth)/2
		if g.boss.X > laserLeft && g.boss.X < laserRight {
			g.boss.HP -= 2 
			if g.frame % 5 == 0 {
				g.createExplosion(g.boss.X + (rand.Float32()-0.5)*60, g.boss.Y + (rand.Float32()-0.5)*40, color.RGBA{0, 255, 255, 255})
			}
			if g.boss.HP <= 0 {
				g.killBoss()
			}
		}
	}

	// --- ОБНОВЛЕНИЕ БОНУСОВ (ЗДЕСЬ ЧАСТО БЫВАЕТ ОШИБКА) ---
	for i := len(g.powerups) - 1; i >= 0; i-- {
		// Если слайс уже пуст из-за предыдущих удалений в этом же цикле (редко, но бывает при багах логики)
		if len(g.powerups) == 0 {
			break
		}
		
		// Проверяем, что индекс i все еще валиден
		if i >= len(g.powerups) {
			continue
		}

		p := &g.powerups[i]
		p.Y += p.VY
		dx := p.X - g.pX
		dy := p.Y - g.pY
		
		// Подбор бонуса
		if dx*dx + dy*dy < 900 {
			if p.Type == "shield" {
				g.hasShield = true
				g.shieldTimer = ShieldDuration
			} else if p.Type == "laser" {
				g.hasLaserUpgrade = true
				g.laserEnergy = LaserMaxEnergy
			}
			g.createExplosion(g.pX, g.pY, color.RGBA{0, 255, 255, 255})
			
			// Безопасное удаление
			if len(g.powerups) > 0 {
				g.powerups[i] = g.powerups[len(g.powerups)-1]
				g.powerups = g.powerups[:len(g.powerups)-1]
			}
			continue
		}
		
		// Улет за экран
		if p.Y > H+20 {
			if len(g.powerups) > 0 {
				g.powerups[i] = g.powerups[len(g.powerups)-1]
				g.powerups = g.powerups[:len(g.powerups)-1]
			}
		}
	}

	// --- ИГРОК ---
	mx, my := ebiten.CursorPosition()
	g.pX += (float32(mx) - g.pX) * 0.2
	g.pY += (float32(my) - g.pY) * 0.2
	if g.pX < 0 { g.pX = 0 }
	if g.pX > W { g.pX = W }
	if g.pY < 0 { g.pY = 0 }
	if g.pY > H { g.pY = H }

	// Стрельба игрока
	g.timer++
	if g.timer > 10 {
		g.bullets = append(g.bullets, Object{X: g.pX, Y: g.pY - 20, VX: 0, VY: -10, Color: color.RGBA{255, 255, 0, 255}})
		g.timer = 0
	}

	// --- ВРАГИ ---
	for i := len(g.enemies) - 1; i >= 0; i-- {
		// Защита от выхода за границы
		if i >= len(g.enemies) {
			continue
		}
		
		e := &g.enemies[i]
		e.Y += e.VY

		// Столкновение с игроком
		dx := e.X - g.pX
		dy := e.Y - g.pY
		if dx*dx + dy*dy < 900 {
			if g.hasShield {
				g.createExplosion(e.X, e.Y, color.RGBA{255, 50, 50, 255})
				if len(g.enemies) > 0 {
					g.enemies[i] = g.enemies[len(g.enemies)-1]
					g.enemies = g.enemies[:len(g.enemies)-1]
				}
				continue
			} else {
				g.gameOverReset()
				continue
			}
		}

		// Столкновение с пулями игрока
		hit := false
		for j := len(g.bullets) - 1; j >= 0; j-- {
			b := &g.bullets[j]
			if b.Color.R == 255 && b.Color.G == 255 && b.Color.B == 0 { 
				bdx := e.X - b.X
				bdy := e.Y - b.Y
				if bdx*bdx + bdy*bdy < 400 {
					g.createExplosion(e.X, e.Y, color.RGBA{255, 50, 50, 255})
					
					if len(g.bullets) > 0 {
						g.bullets[j] = g.bullets[len(g.bullets)-1]
						g.bullets = g.bullets[:len(g.bullets)-1]
					}
					
					hit = true
					g.score += 100
					break
				}
			}
		}
		
		if hit {
			if len(g.enemies) > 0 {
				g.enemies[i] = g.enemies[len(g.enemies)-1]
				g.enemies = g.enemies[:len(g.enemies)-1]
			}
			continue
		}
		
		if e.Y > H+20 {
			if len(g.enemies) > 0 {
				g.enemies[i] = g.enemies[len(g.enemies)-1]
				g.enemies = g.enemies[:len(g.enemies)-1]
			}
		}
	}

	// Частицы
	for i := len(g.particles) - 1; i >= 0; i-- {
		if i >= len(g.particles) { continue }
		
		p := &g.particles[i]
		p.X += p.VX
		p.Y += p.VY
		p.Life--
		if p.Life <= 0 {
			if len(g.particles) > 0 {
				g.particles[i] = g.particles[len(g.particles)-1]
				g.particles = g.particles[:len(g.particles)-1]
			}
		}
	}

	return nil
}
func (g *Game) killBoss() {
	g.boss.Active = false
	g.score += 5000
	g.nextBossScore += 3000
	
	for k:=0; k<100; k++ {
		angle := rand.Float32() * 6.28
		speed := 2 + rand.Float32()*8
		c := color.RGBA{255, uint8(rand.Intn(200)), 0, 255}
		g.particles = append(g.particles, Object{
			X: g.boss.X, Y: g.boss.Y,
			VX: float32(math.Cos(float64(angle))) * speed,
			VY: float32(math.Sin(float64(angle))) * speed,
			Life: 60 + rand.Intn(30), Color: c,
		})
	}
}

func (g *Game) gameOverReset() {
	g.createExplosion(g.pX, g.pY, color.RGBA{0, 255, 150, 255})
	g.score = 0
	g.nextBossScore = BossSpawnScore
	g.enemies = nil
	g.bullets = nil
	g.boss.Active = false
	g.hasShield = false
}

func (g *Game) createExplosion(x, y float32, c color.RGBA) {
	for k := 0; k < 8; k++ {
		angle := rand.Float32() * 6.28
		speed := 2 + rand.Float32()*3
		g.particles = append(g.particles, Object{
			X: x, Y: y,
			VX: float32(math.Cos(float64(angle))) * speed,
			VY: float32(math.Sin(float64(angle))) * speed,
			Life: 20 + rand.Intn(10), Color: c,
		})
	}
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{5, 5, 20, 255})

	// Звезды
	for _, s := range g.stars {
		vector.DrawFilledRect(screen, s.X, s.Y, s.Size, s.Size, color.RGBA{200, 200, 255, s.Bright}, false)
	}

	// Лазер
	if g.isLaserActive {
		jitter := float32(rand.Intn(3)) - 1.5
		currentWidth := float32(LaserWidth) + jitter
		vector.DrawFilledRect(screen, g.pX - currentWidth*1.5, 0, currentWidth*3, float32(H), color.RGBA{0, 255, 255, 100}, false)
		vector.DrawFilledRect(screen, g.pX - currentWidth/2, 0, currentWidth, float32(H), color.RGBA{255, 255, 255, 255}, false)
	}

	// Босс
	if g.boss.Active {
		vector.DrawFilledCircle(screen, g.boss.X, g.boss.Y, 50, color.RGBA{80, 0, 80, 255}, false)
		vector.StrokeCircle(screen, g.boss.X, g.boss.Y, 50, 4, color.RGBA{255, 0, 255, 255}, false)
		vector.DrawFilledRect(screen, g.boss.X-30, g.boss.Y, 60, 20, color.RGBA{40, 0, 40, 255}, false)
		
		var barW float32 = 120
		ratio := float32(g.boss.HP) / float32(g.boss.MaxHP)
		vector.DrawFilledRect(screen, g.boss.X - barW/2, g.boss.Y - 70, barW, 10, color.RGBA{50, 50, 50, 255}, false)
		vector.DrawFilledRect(screen, g.boss.X - barW/2, g.boss.Y - 70, barW*ratio, 10, color.RGBA{255, 0, 0, 255}, false)
	}

	// Пули
	for _, b := range g.bullets {
		vector.DrawFilledRect(screen, b.X-3, b.Y-3, 6, 6, b.Color, false)
	}

	// Враги
	for _, e := range g.enemies {
		vector.DrawFilledCircle(screen, e.X, e.Y, 12, color.RGBA{255, 50, 50, 255}, false)
	}

	// Бонусы
	for _, p := range g.powerups {
		if p.Type == "shield" {
			vector.DrawFilledCircle(screen, p.X, p.Y, 10, color.RGBA{0, 100, 255, 255}, false)
			vector.StrokeCircle(screen, p.X, p.Y, 14, 2, color.RGBA{255, 255, 255, 255}, false)
		} else {
			// ИСПРАВЛЕНО: используем RGBA вместо color.Cyan
			vector.DrawFilledRect(screen, p.X-5, p.Y-10, 10, 20, color.RGBA{0, 255, 255, 255}, false)
			vector.StrokeRect(screen, p.X-7, p.Y-12, 14, 24, 2, color.RGBA{255, 255, 255, 255}, false)
		}
	}

	// Игрок
	playerColor := color.RGBA{0, 255, 150, 255}
	if g.hasShield {
		vector.StrokeCircle(screen, g.pX, g.pY, 25, 3, color.RGBA{0, 150, 255, 200}, false)
		var barWidth float32 = 40.0
		var barHeight float32 = 4.0
		ratio := float32(g.shieldTimer) / float32(ShieldDuration)
		vector.DrawFilledRect(screen, g.pX - barWidth/2, g.pY - 35, barWidth, barHeight, color.RGBA{50, 50, 50, 255}, false)
		vector.DrawFilledRect(screen, g.pX - barWidth/2, g.pY - 35, barWidth*ratio, barHeight, color.RGBA{0, 200, 255, 255}, false)
	}

	if g.hasLaserUpgrade {
		var barWidth float32 = 60.0
		var barHeight float32 = 6.0
		ratio := float32(g.laserEnergy) / float32(LaserMaxEnergy)
		
		// ИСПРАВЛЕНО: определяем цвет через RGBA
		var barColor color.RGBA
		if g.isLaserActive {
			barColor = color.RGBA{255, 255, 255, 255} // White
		} else {
			barColor = color.RGBA{0, 255, 255, 255} // Cyan
		}
		
		vector.DrawFilledRect(screen, g.pX - barWidth/2, g.pY + 25, barWidth, barHeight, color.RGBA{50, 50, 50, 255}, false)
		vector.DrawFilledRect(screen, g.pX - barWidth/2, g.pY + 25, barWidth*ratio, barHeight, barColor, false)
	}

	vector.DrawFilledCircle(screen, g.pX, g.pY, 15, playerColor, false)

	// Частицы
	for _, p := range g.particles {
		alpha := uint8(float32(p.Life) / 60.0 * 255.0)
		c := color.RGBA{p.Color.R, p.Color.G, p.Color.B, alpha}
		vector.DrawFilledRect(screen, p.X-2, p.Y-2, 4, 4, c, false)
	}

	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("SCORE: %d", g.score), 10, 10)
	if g.boss.Active {
		ebitenutil.DebugPrintAt(screen, "WARNING: BOSS APPROACHING", W/2-80, 10)
	}
}

func (g *Game) Layout(w, h int) (int, int) { return W, H }

func main() {
	rand.Seed(time.Now().UnixNano())
	ebiten.SetWindowTitle("SKY FORCE - BOSS BATTLE")
	ebiten.SetWindowSize(W, H)
	game := &Game{pX: W / 2, pY: H - 100}
	game.Init()
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}