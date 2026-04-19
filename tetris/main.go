package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"image/color"
	"log"
	"math/rand"
)


const (
	S = 20 // Кубик
	W = 10 // Стакан
	H = 20 // Высота
)

// Сегрегация фигур: массив фигур -> массив точек -> [x, y]
var shapes = [][][]int{
	{{0, 0}, {1, 0}, {0, 1}, {1, 1}}, // Квадрат
	{{0, 0}, {1, 0}, {2, 0}, {3, 0}}, // Палка
	{{0, 1}, {1, 1}, {2, 1}, {1, 0}}, // Т-шка
	{{0, 0}, {0, 1}, {1, 1}, {2, 1}}, // Г-шка
}

type Game struct {
	board      [W][H]bool
	posX, posY int
	timer      int
	current    [][]int
}

func (g *Game) Update() error {
	g.timer++
	if g.timer > 30 {
		g.posY++
		g.timer = 0
	}

	// 1. УПРАВЛЕНИЕ ВЛЕВО (Защита от Паники [-1])
	if inpututil.IsKeyJustPressed(ebiten.KeyLeft) {
		canMove := true
		for _, p := range g.current {
			if g.posX+p[0] <= 0 { canMove = false; break }
		}
		if canMove { g.posX-- }
	}

	// 2. УПРАВЛЕНИЕ ВПРАВО
	if inpututil.IsKeyJustPressed(ebiten.KeyRight) {
		canMove := true
		for _, p := range g.current {
			if g.posX+p[0] >= W-1 { canMove = false; break }
		}
		if canMove { g.posX++ }
	}

	// 3. ПОВОРОТ ФИГУРЫ (Магия ЧТЗ)
	if inpututil.IsKeyJustPressed(ebiten.KeyUp) {
		newShape := make([][]int, len(g.current))
		for i, p := range g.current {
			// Крутим: x = -y, y = x
			newShape[i] = []int{-p[1], p[0]}
		}
		// Проверка: не влетаем ли в стену или в кости после поворота
		canRotate := true
		for _, p := range newShape {
			nx, ny := g.posX+p[0], g.posY+p[1]
			if nx < 0 || nx >= W || ny < 0 || ny >= H || (ny >= 0 && g.board[nx][ny]) {
				canRotate = false; break
			}
		}
		if canRotate { g.current = newShape }
	}

	// 4. ПРИЗЕМЛЕНИЕ
	hitBottom := false
	for _, p := range g.current {
		nextY := g.posY + p[1] + 1
		if nextY >= H { hitBottom = true; break }
		if g.posX+p[0] >= 0 && g.posX+p[0] < W && g.board[g.posX+p[0]][nextY] {
			hitBottom = true; break
		}
	}

	if hitBottom {
		for _, p := range g.current {
			bx, by := g.posX+p[0], g.posY+p[1]
			if bx >= 0 && bx < W && by >= 0 && by < H { g.board[bx][by] = true }
		}
		// СЖИГАЕМ ЛИНИИ
		for y := 0; y < H; y++ {
			full := true
			for x := 0; x < W; x++ {
				if !g.board[x][y] { full = false; break }
			}
			if full {
				for row := y; row > 0; row-- {
					for x := 0; x < W; x++ { g.board[x][row] = g.board[x][row-1] }
				}
				for x := 0; x < W; x++ { g.board[x][0] = false }
			}
		}
		g.posY, g.posX = 0, 4
		g.current = shapes[rand.Intn(len(shapes))]
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	// Рисуем стакан
	for x := 0; x < W; x++ {
		for y := 0; y < H; y++ {
			vector.StrokeRect(screen, float32(x*S), float32(y*S), S, S, 1, color.White, false)
			if g.board[x][y] {
				vector.DrawFilledRect(screen, float32(x*S), float32(y*S), S, S, color.RGBA{235, 6, 255, 255}, false)
			}
		}
	}
	// РИСУЕМ АКТИВНЫЙ СТВОЛ
	for _, p := range g.current {
		vector.DrawFilledRect(screen, float32((g.posX+p[0])*S), float32((g.posY+p[1])*S), S, S, color.RGBA{0, 255, 0, 255}, false)
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) { return W * S, H * S }

func main() {
	ebiten.SetWindowTitle("VANAHEIM TETRIS - CHAMPION EDITION")
	ebiten.SetWindowSize(W*S*2, H*S*2)
	game := &Game{posX: 4, current: shapes[rand.Intn(len(shapes))]}
	if err := ebiten.RunGame(game); err != nil { log.Fatal(err) }
}