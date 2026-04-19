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
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	S = 20 // Размер клетки
	W = 30 // Ширина поля
	H = 20 // Высота поля
	TickRate = 80 * time.Millisecond // Скорость игры
)

type Point struct {
	X, Y int
}

type Game struct {
	snake      []Point
	dir        Point
	nextDir    Point
	inputQueue []Point // Очередь нажатий для защиты от самоубийства
	apple      Point
	lastUpdate time.Time
	score      int
	isDead     bool
	tick       float64 // Для анимации
}

func (g *Game) Update() error {
	g.tick += 0.1

	if g.isDead {
		if ebiten.IsKeyPressed(ebiten.KeyEnter) || ebiten.IsKeyPressed(ebiten.KeySpace) {
			g.reset()
		}
		return nil
	}

	// --- Обработка ввода (с буфером) ---
	if ebiten.IsKeyPressed(ebiten.KeyLeft) && len(g.inputQueue) < 2 {
		g.queueInput(Point{-1, 0})
	}
	if ebiten.IsKeyPressed(ebiten.KeyRight) && len(g.inputQueue) < 2 {
		g.queueInput(Point{1, 0})
	}
	if ebiten.IsKeyPressed(ebiten.KeyUp) && len(g.inputQueue) < 2 {
		g.queueInput(Point{0, -1})
	}
	if ebiten.IsKeyPressed(ebiten.KeyDown) && len(g.inputQueue) < 2 {
		g.queueInput(Point{0, 1})
	}

	// --- Логика движения ---
	if time.Since(g.lastUpdate) > TickRate {
		g.processInput()
		
		head := Point{g.snake[0].X + g.dir.X, g.snake[0].Y + g.dir.Y}

		// Проверка столкновений со стенами
		if head.X < 0 || head.X >= W || head.Y < 0 || head.Y >= H {
			g.isDead = true
		}

		// Проверка столкновений с хвостом
		if !g.isDead {
			for _, p := range g.snake {
				if head == p {
					g.isDead = true
					break
				}
			}
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

// Добавляем направление в очередь, если оно не противоположно текущему
func (g *Game) queueInput(newDir Point) {
	lastDir := g.dir
	if len(g.inputQueue) > 0 {
		lastDir = g.inputQueue[len(g.inputQueue)-1]
	}

	// Нельзя развернуться на 180 градусов
	if newDir.X != -lastDir.X || newDir.Y != -lastDir.Y {
		g.inputQueue = append(g.inputQueue, newDir)
	}
}

// Берем первое направление из очереди
func (g *Game) processInput() {
	if len(g.inputQueue) > 0 {
		g.dir = g.inputQueue[0]
		g.inputQueue = g.inputQueue[1:]
	}
}

func (g *Game) spawnApple() {
	for {
		g.apple = Point{rand.Intn(W), rand.Intn(H)}
		overlap := false
		for _, p := range g.snake {
			if g.apple == p {
				overlap = true
				break
			}
		}
		if !overlap {
			break
		}
	}
}

func (g *Game) reset() {
	g.snake = []Point{{W / 2, H / 2}, {W / 2, H / 2 + 1}, {W / 2, H / 2 + 2}}
	g.dir = Point{0, -1}
	g.nextDir = g.dir
	g.inputQueue = nil
	g.score = 0
	g.isDead = false
	g.spawnApple()
	g.lastUpdate = time.Now()
}

func (g *Game) Draw(screen *ebiten.Image) {
	// Фон
	screen.Fill(color.RGBA{15, 15, 20, 255})

	// Рисуем сетку (опционально, для стиля)
	for x := 0; x < W; x++ {
		for y := 0; y < H; y++ {
			vector.StrokeRect(screen, float32(x*S), float32(y*S), S, S, 1, color.RGBA{30, 30, 40, 255}, false)
		}
	}

	// яблоко с легкой пульсацией
	scale := 1.0 + math.Sin(g.tick)*0.1
	offset := float32((1.0 - scale) * S / 2)
	vector.DrawFilledRect(
		screen, 
		float32(g.apple.X*S)+offset, 
		float32(g.apple.Y*S)+offset, 
		float32(S)*float32(scale)-2*offset, 
		float32(S)*float32(scale)-2*offset, 
		color.RGBA{255, 50, 80, 255}, 
		false,
	)

	// Рисуем змейку
	for i, p := range g.snake {
		c := color.RGBA{0, 255, 100, 255}
		if i == 0 {
			c = color.RGBA{100, 255, 150, 255} // Голова светлее
		}
		vector.DrawFilledRect(screen, float32(p.X*S)+1, float32(p.Y*S)+1, S-2, S-2, c, false)
	}

	// UI
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("SCORE: %d", g.score), 10, 10)
	
	if g.isDead {
		// Полупрозрачный фон для Game Over
		vector.DrawFilledRect(screen, 0, float32(H*S)/2-40, float32(W*S), 80, color.RGBA{0, 0, 0, 150}, false)
		ebitenutil.DebugPrintAt(screen, "GAME OVER", int(float32(W*S)/2-40), int(float32(H*S)/2-10))
		ebitenutil.DebugPrintAt(screen, "Press ENTER to Restart", int(float32(W*S)/2-70), int(float32(H*S)/2+10))
	}
}

func (g *Game) Layout(w, h int) (int, int) {
	return W * S, H * S
}

func main() {
	rand.Seed(time.Now().UnixNano())
	ebiten.SetWindowTitle("SNAKE")
	ebiten.SetWindowSize(W*S, H*S)
	ebiten.SetWindowResizable(false)
	
	g := &Game{}
	g.reset()
	
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}

