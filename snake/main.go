package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"image/color"
	"log"
	"math/rand"
	"time"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"fmt"
)


const (
	S = 20 // Размер кубика
	W = 30 // Поле пошире, чтоб было где разгуляться
	H = 20 
)

type Point struct{ X, Y int }

type Game struct {
	snake     []Point
	dir       Point
	nextDir   Point
	apple     Point
	lastUpdate time.Time
	score     int
	isDead    bool
}

func (g *Game) Update() error {
	if g.isDead {
		if ebiten.IsKeyPressed(ebiten.KeyEnter) { g.reset() }
		return nil
	}

	// Оплодотворяем управление без возможности самоубийства
	if ebiten.IsKeyPressed(ebiten.KeyLeft) && g.dir.X == 0 { g.nextDir = Point{-1, 0} }
	if ebiten.IsKeyPressed(ebiten.KeyRight) && g.dir.X == 0 { g.nextDir = Point{1, 0} }
	if ebiten.IsKeyPressed(ebiten.KeyUp) && g.dir.Y == 0 { g.nextDir = Point{0, -1} }
	if ebiten.IsKeyPressed(ebiten.KeyDown) && g.dir.Y == 0 { g.nextDir = Point{0, 1} }

	// Плавный тикер (0.1 сек на шаг)
	if time.Since(g.lastUpdate) > 100*time.Millisecond {
		g.dir = g.nextDir
		head := Point{g.snake[0].X + g.dir.X, g.snake[0].Y + g.dir.Y}

		if head.X < 0 || head.X >= W || head.Y < 0 || head.Y >= H { g.isDead = true }
		for _, p := range g.snake {
			if head == p { g.isDead = true }
		}

		if !g.isDead {
			g.snake = append([]Point{head}, g.snake...)
			if head == g.apple {
				g.score += 10
				g.spawnApple()
			} else {
				g.snake = g.snake[:len(g.snake)-1]
			}
		}
		g.lastUpdate = time.Now()
	}
	return nil
}

func (g *Game) spawnApple() {
	for {
		g.apple = Point{rand.Intn(W), rand.Intn(H)}
		overlap := false
		for _, p := range g.snake {
			if g.apple == p { overlap = true; break }
		}
		if !overlap { break }
	}
}

func (g *Game) reset() {
	g.snake = []Point{{W / 2, H / 2}, {W/2, H/2 + 1}}
	g.dir = Point{0, -1}
	g.nextDir = g.dir
	g.score = 0
	g.isDead = false
	g.spawnApple()
	g.lastUpdate = time.Now()
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{10, 10, 15, 255}) // Темный элитный фон
	
	for _, p := range g.snake {
		vector.DrawFilledRect(screen, float32(p.X*S), float32(p.Y*S), S-1, S-1, color.RGBA{0, 255, 100, 255}, false)
	}
	vector.DrawFilledRect(screen, float32(g.apple.X*S), float32(g.apple.Y*S), S-1, S-1, color.RGBA{255, 50, 100, 255}, false)

	ebitenutil.DebugPrint(screen, fmt.Sprintf("SCORE: %d", g.score))
	if g.isDead {
		ebitenutil.DebugPrint(screen, "\n\n  GAME OVER! PRESS ENTER")
	}
}

func (g *Game) Layout(w, h int) (int, int) { return W * S, H * S }

func main() {
	rand.Seed(time.Now().UnixNano())
	ebiten.SetWindowTitle("SNAKE ELITE - CHELYABINSK")
	ebiten.SetWindowSize(W*S, H*S)
	g := &Game{}
	g.reset()
	if err := ebiten.RunGame(g); err != nil { log.Fatal(err) }
}