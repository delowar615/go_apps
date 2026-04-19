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
	S = 25 // Размер чутка побольше я слепошарый
	W = 10 // Ширина норм
	H = 20 // Высота годится
)

// Фигуры: I, O, T, S, Z, J, L
var shapes = [][][]int{
	{{0, 0}, {1, 0}, {2, 0}, {3, 0}}, // I (Палка)
	{{0, 0}, {1, 0}, {0, 1}, {1, 1}}, // O (Квадрат)
	{{0, 0}, {1, 0}, {2, 0}, {1, 1}}, // T и прочая поебень
	{{1, 0}, {2, 0}, {0, 1}, {1, 1}}, // S
	{{0, 0}, {1, 0}, {1, 1}, {2, 1}}, // Z
	{{0, 0}, {0, 1}, {1, 1}, {2, 1}}, // J
	{{2, 0}, {0, 1}, {1, 1}, {2, 1}}, // L
}

// Цвета для фигур
var colors = []color.RGBA{
	{0, 255, 255, 255},   // I - Cyan
	{255, 255, 0, 255},   // O - Yellow
	{128, 0, 128, 255},   // T - Purple
	{0, 255, 0, 255},     // S - Green
	{255, 0, 0, 255},     // Z - Red
	{0, 0, 255, 255},     // J - Blue
	{255, 165, 0, 255},   // L - Orange
}

type Game struct {
	board      [W][H]color.RGBA // Храним цвет занятой клетки
	posX, posY int
	timer      int
	current    [][]int
	currentIdx int // Индекс текущей фигуры для цвета
	gameOver   bool
}

func (g *Game) Update() error {
	if g.gameOver {
		if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
			g.reset()
		}
		return nil
	}

	g.timer++
	// Скорость падения фигур
	speed := 40 
	if g.timer > speed {
		g.posY++
		g.timer = 0
	}

	// --- руль ---

	// Влево
	if inpututil.IsKeyJustPressed(ebiten.KeyLeft) {
		g.move(-1, 0)
	}
	// Вправо
	if inpututil.IsKeyJustPressed(ebiten.KeyRight) {
		g.move(1, 0)
	}
	// Вниз (ускорение)
	if ebiten.IsKeyPressed(ebiten.KeyDown) {
		g.timer += 5 // Пропускаем кадры таймера
	}

	// Поворот
	if inpututil.IsKeyJustPressed(ebiten.KeyUp) {
		g.rotate()
	}

	// --- здесь жопами толкаемся ---
	
	if g.checkCollision(0, 1) {
		g.fixPiece()
		g.clearLines()
		g.spawnPiece()
		
		// Проверка на пиздец
		if g.checkCollision(0, 0) {
			g.gameOver = true
		}
	}

	return nil
}

// Движение фигуры
func (g *Game) move(dx, dy int) {
	if !g.checkCollision(dx, dy) {
		g.posX += dx
		g.posY += dy
	}
}

// Поворот фигуры
func (g *Game) rotate() {
	newShape := make([][]int, len(g.current))
	for i, p := range g.current {
		// здесь крутим вертим
		// x' = -y, y' = x
		newShape[i] = []int{-p[1], p[0]}
	}
	
	// щупаем есть ли место для разворота
	// Если если нет то летим как летим
	tempShape := g.current
	g.current = newShape
	
	if g.checkCollision(0, 0) {
		// Если задел отмена поворота
		g.current = tempShape
	}
}

// Проверка столкновений
func (g *Game) checkCollision(dx, dy int) bool {
	for _, p := range g.current {
		nx := g.posX + p[0] + dx
		ny := g.posY + p[1] + dy

		// Выход за границы по X
		if nx < 0 || nx >= W {
			return true
		}
		// Выход за границы по Y (пол)
		if ny >= H {
			return true
		}
		// Столкновение с другой фигурой (проверяем только если ny >= 0)
		if ny >= 0 && g.board[nx][ny].A != 0 {
			return true
		}
	}
	return false
}

// Фиксация фигуры на поле
func (g *Game) fixPiece() {
	c := colors[g.currentIdx]
	for _, p := range g.current {
		bx := g.posX + p[0]
		by := g.posY + p[1]
		if bx >= 0 && bx < W && by >= 0 && by < H {
			g.board[bx][by] = c
		}
	}
}

// Очистка линий
func (g *Game) clearLines() {
	for y := H - 1; y >= 0; y-- {
		full := true
		for x := 0; x < W; x++ {
			if g.board[x][y].A == 0 { // Если прозрачный, значит пустой
				full = false
				break
			}
		}
		if full {
			// Сдвигаем всё вниз
			for row := y; row > 0; row-- {
				for x := 0; x < W; x++ {
					g.board[x][row] = g.board[x][row-1]
				}
			}
			// Очищаем верхнюю строку
			for x := 0; x < W; x++ {
				g.board[x][0] = color.RGBA{0, 0, 0, 0}
			}
			y++ // Проверяем эту же строку снова, так как всё сдвинулось
		}
	}
}

// варганим новую фигуру
func (g *Game) spawnPiece() {
	g.currentIdx = rand.Intn(len(shapes))
	g.current = shapes[g.currentIdx]
	g.posX = W/2 - 1
	g.posY = 0
}

func (g *Game) reset() {
	g.board = [W][H]color.RGBA{}
	g.gameOver = false
	g.spawnPiece()
}

func (g *Game) Draw(screen *ebiten.Image) {
	// Фон
	vector.DrawFilledRect(screen, 0, 0, float32(W*S), float32(H*S), color.RGBA{20, 20, 30, 255}, false)

	// Рисуем стакан (сетку)
	for x := 0; x < W; x++ {
		for y := 0; y < H; y++ {
			// Рамка ячейки
			vector.StrokeRect(screen, float32(x*S), float32(y*S), float32(S), float32(S), 1, color.RGBA{50, 50, 60, 255}, false)
			
			// Заполненные ячейки
			if g.board[x][y].A != 0 {
				vector.DrawFilledRect(screen, float32(x*S)+1, float32(y*S)+1, float32(S-2), float32(S-2), g.board[x][y], false)
			}
		}
	}

	// варганим активную фигуру
	if !g.gameOver {
		c := colors[g.currentIdx]
		for _, p := range g.current {
			vector.DrawFilledRect(screen, float32((g.posX+p[0])*S)+1, float32((g.posY+p[1])*S)+1, float32(S-2), float32(S-2), c, false)
		}
	} else {
		// Надпись Game Over
		//просто мигаем экраном или рисуем прямоугольник
		vector.DrawFilledRect(screen, 50, 100, float32(W*S-100), 50, color.RGBA{0,0,0,200}, false)
		// Здесь можно было бы добавить текст, если подключить font package
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return W * S, H * S
}

func main() {
	rand.Seed(42) 
	ebiten.SetWindowTitle("TETRIS")
	ebiten.SetWindowSize(W*S*2, H*S*2)
	
	game := &Game{}
	game.reset()
	
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}