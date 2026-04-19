package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"image/color"
	"log"
	"math/rand"
	"time"
	"fmt"

)

const (
	W = 400
	H = 600
)

type Object struct{ X, Y float32 }

type Game struct {
	pX, pY  float32
	bullets []Object
	enemies []Object
	timer   int
	score   int
}

func (g *Game) Update() error {
	// 1. Управление (самолет за мышкой)
	mx, my := ebiten.CursorPosition()
	g.pX, g.pY = float32(mx), float32(my)

	// 2. Оплодотворение неба пулями (каждые 12 кадров)
	g.timer++
	if g.timer > 12 {
		g.bullets = append(g.bullets, Object{g.pX, g.pY - 20})
		g.timer = 0
	}

	// 3. Спавн рептилоидов (наглый вброс сверху)
	if rand.Intn(40) == 0 {
		g.enemies = append(g.enemies, Object{rand.Float32() * W, -20})
	}

	// 4. Движение пуль вверх "в натяг"
	for i := 0; i < len(g.bullets); i++ {
		g.bullets[i].Y -= 8
		if g.bullets[i].Y < -10 {
			g.bullets = append(g.bullets[:i], g.bullets[i+1:]...)
			i--
		}
	}

	// 5. Движение врагов вниз и ПРОВЕРКА НА ГИБЕЛЬ
	for i := 0; i < len(g.enemies); i++ {
		g.enemies[i].Y += 3 // Скорость врага
		
		// Коллизия: пуля попала в рептилоида?
		for j := 0; j < len(g.bullets); j++ {
			dx := g.enemies[i].X - g.bullets[j].X
			dy := g.enemies[i].Y - g.bullets[j].Y
			if dx*dx+dy*dy < 400 { // Если бахнуло
				g.enemies = append(g.enemies[:i], g.enemies[i+1:]...)
				g.bullets = append(g.bullets[:j], g.bullets[j+1:]...)
				g.score += 100
				i--
				break
			}
		}

		if i >= 0 && i < len(g.enemies) && g.enemies[i].Y > H+20 {
			g.enemies = append(g.enemies[:i], g.enemies[i+1:]...)
			i--
		}
	}

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{5, 5, 20, 255}) // Ночное небо Челябинска

	// Рисуем пули (трассеры)
	for _, b := range g.bullets {
		vector.DrawFilledRect(screen, b.X-2, b.Y, 4, 12, color.RGBA{255, 255, 0, 255}, false)
	}

	// Рисуем врагов-рептилоидов (красные окурки)
	for _, e := range g.enemies {
		vector.DrawFilledCircle(screen, e.X, e.Y, 12, color.RGBA{255, 50, 50, 255}, false)
	}

	// Твой титановый штурмовик
	vector.DrawFilledCircle(screen, g.pX, g.pY, 15, color.RGBA{0, 255, 150, 255}, false)

	// Статистика побед
	ebitenutil.DebugPrint(screen, fmt.Sprintf("REPTILOIDS DESTROYED: %d", g.score))
}

func (g *Game) Layout(w, h int) (int, int) { return W, H }

func main() {
	rand.Seed(time.Now().UnixNano())
	ebiten.SetWindowTitle("SKY FORCE - REPTILOID HUNTER")
	ebiten.SetWindowSize(W, H)
	if err := ebiten.RunGame(&Game{pX: W / 2, pY: H - 50}); err != nil {
		log.Fatal(err)
	}
}