package main

import (
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func AUtoMeters(au float64) float64 {
	// 1 AU is defined as the average distance from the Earth to the Sun, which is approximately 149.6 million kilometers.
	return au * 149.6e6 * 1000
}
func MetersToAU(meters float64) float64 {
	return meters / (149.6e6 * 1000)
}

func AUDayToMetersPerSecond(auPerDay float64) float64 {
	// 1 AU is defined as the average distance from the Earth to the Sun, which is approximately 149.6 million kilometers.
	// There are 86400 seconds in a day.
	return auPerDay * 149.6e6 * 1000 / 86400
}

// G is the gravitational constant,
const G = 6.67430e-11 // in m^3 kg^-1 s^-2

// Body represents a celestial body in the n-body problem.
type Body struct {
	Label string
	// Mass is the mass of the body.
	Mass float64

	// Position is the current position of the body.
	Position [3]float64

	// Velocity is the current velocity of the body.
	Velocity [3]float64
}

// Distance calculates the distance between two bodies.
func Distance(a, b *Body) float64 {
	// Calculate the distance between the two bodies.
	distance := math.Sqrt(math.Pow(a.Position[0]-b.Position[0], 2) + math.Pow(a.Position[1]-b.Position[1], 2) + math.Pow(a.Position[2]-b.Position[2], 2))

	return distance
}

// Update updates the position and velocity of the body based on the positions and velocities of all the other bodies.
func (b *Body) Update(bodies []*Body, dt float64) {
	// Initialize the acceleration to zero.
	acceleration := [3]float64{0, 0, 0}

	// Calculate the acceleration of the body due to the gravitational forces of the other bodies.
	for _, other := range bodies {
		if b == other {
			// Skip the body itself.
			continue
		}

		// Calculate the distance between the two bodies.
		distance := math.Sqrt(math.Pow(other.Position[0]-b.Position[0], 2) + math.Pow(other.Position[1]-b.Position[1], 2) + math.Pow(other.Position[2]-b.Position[2], 2))

		// Calculate the gravitational force between the two bodies.
		force := (G * /* b.Mass * */ other.Mass) / math.Pow(distance, 2)

		// Calculate the acceleration of the body due to the gravitational force of the other body.
		acceleration[0] += force * (other.Position[0] - b.Position[0]) / distance
		acceleration[1] += force * (other.Position[1] - b.Position[1]) / distance
		acceleration[2] += force * (other.Position[2] - b.Position[2]) / distance
	}

	// Update the velocity of the body using the Euler method.
	b.Velocity[0] += acceleration[0] * dt
	b.Velocity[1] += acceleration[1] * dt
	b.Velocity[2] += acceleration[2] * dt

	// Update the position of the body using the Euler method.
	// TODO split out
	b.Position[0] += b.Velocity[0] * dt
	b.Position[1] += b.Velocity[1] * dt
	b.Position[2] += b.Velocity[2] * dt
}

// Step advances the model by the given time duration.
func Step(bodies []*Body, dt time.Duration) {
	// Convert dt from seconds to days
	days := dt.Seconds()

	// Update the positions and velocities of all the bodies.
	for _, b := range bodies {
		b.Update(bodies, days)
	}
}

type tickMsg time.Time

func main() {

	p := tea.NewProgram(model{}, tea.WithAltScreen())
	go func() {
		astroMain(p)
	}()

	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(tea.EnterAltScreen)
}

func (m model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := message.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.size = msg
		return m, nil
	case bodiesWithToken:
		<-msg.token
		m.bodies = make([]Body, 0, len(msg.bodies))
		for _, b := range msg.bodies {
			m.bodies = append(m.bodies, *b)
		}
		m.bodiesWithToken = msg
		msg.token <- struct{}{}
		return m, nil
	}

	return m, nil
}

func (m model) View() string {

	height := m.size.Height - 2
	width := m.size.Width
	type coord struct {
		x, y int
	}
	bm := map[coord]int{}

	mapScale := AUtoMeters(4)

	metersPerCell := mapScale / float64(height)

	for i, b := range m.bodies {

		c := coord{int(b.Position[0] / metersPerCell), int(b.Position[1] / metersPerCell)}
		c.y += height / 4
		c.x += width / 2
		bm[c] = i
	}

	var s strings.Builder
	s.WriteString(fmt.Sprintf("%d objects (%dx%d): %v %.0f(i/s) \n", len(m.bodies), m.size.Height, m.size.Width, m.bodiesWithToken.currentTime, m.bodiesWithToken.itersSec))
	s.Grow((height + 2) * (width + 1))

	runes := []rune{'☉', '♁', '♂'}

	for i := 0; i < height; i++ {
		for j := 0; j < width; j++ {
			if b, ok := bm[coord{x: j, y: i}]; ok {
				s.WriteRune(runes[b])
				_ = b
			} else {
				s.WriteRune(' ')
			}
		}
		s.WriteRune('\n')
	}

	return s.String()
}

type model struct {
	bodies          []Body
	bodiesWithToken bodiesWithToken
	size            tea.WindowSizeMsg
}

type bodiesWithToken struct {
	bodies      []*Body
	currentTime time.Time
	itersSec    float64
	token       chan struct{}
}

func astroMain(p *tea.Program) {
	startTime := time.Date(2022, time.January, 1, 0, 0, 0, 0, time.UTC)
	bodies := []*Body{
		{
			Label: "Sun",
			Mass:  1.989e30, // Mass of the Sun
			Position: [3]float64{
				AUtoMeters(0), // x-position (AU)
				0,             // y-position (AU)
				0,             // z-position (AU)
			},
			Velocity: [3]float64{
				0, // x-velocity (AU/day)
				0, // y-velocity (AU/day)
				0, // z-velocity (AU/day)
			},
		},
		{
			Label: "Earth",
			Mass:  5.972e24, // Mass of Earth
			Position: [3]float64{
				AUtoMeters(-1.01673977e-01), // x-position (AU)
				AUtoMeters(7.00034986e-01),  // y-position (AU)
				AUtoMeters(-1.85435480e-06), // z-position (AU)
			},
			Velocity: [3]float64{
				AUDayToMetersPerSecond(-1.42987359e-02), // x-velocity (AU/day)
				AUDayToMetersPerSecond(-1.00797828e-02), // y-velocity (AU/day)
				AUDayToMetersPerSecond(2.24008069e-07),  // z-velocity (AU/day)
			},
		},
		{
			Label: "Mars",
			Mass:  6.39e23, // Mass of Mars
			Position: [3]float64{
				AUtoMeters(1.38708645),      // x-position (AU)
				AUtoMeters(-9.63136861e-01), // y-position (AU)
				AUtoMeters(3.79103570e-02),  // z-position (AU)
			},
			Velocity: [3]float64{
				AUDayToMetersPerSecond(7.20279246e-03),     // x-velocity (AU/day)
				AUDayToMetersPerSecond(1.67110509e-02) / 2, // y-velocity (AU/day) // FIXME ?
				AUDayToMetersPerSecond(-1.70863874e-03),    // z-velocity (AU/day)
			},
		},
		// TODO: Add more bodies as needed.
	}

	const advTime = time.Second

	currentTime := startTime

	printInterval := time.Hour * 7 * 24

	var (
		lastPrint         time.Time
		lastPrintRealtime time.Time
		iterCount         int
	)

	payload := bodiesWithToken{
		bodies: bodies,
		token:  make(chan struct{}),
	}

	for {
		if lastPrint.Add(printInterval).Before(currentTime) /*|| true*/ {
			if !lastPrintRealtime.IsZero() {
				dur := time.Since(lastPrintRealtime)
				// fmt.Println("iters/sec: ", float64(iterCount)/(dur.Seconds()))
				payload.itersSec = float64(iterCount) / (dur.Seconds())
				_ = dur
			}
			lastPrint = currentTime
			// fmt.Println(currentTime)
			for i, body := range bodies {
				// fmt.Println(body.Label, body.Position)
				for _, otherBody := range bodies[i:] {
					if body == otherBody {
						continue
					}
					// fmt.Println("Distance to ", otherBody.Label, MetersToAU(Distance(body, otherBody)))
				}
			}
			payload.currentTime = currentTime
			p.Send(payload)
			lastPrintRealtime = time.Now()
			iterCount = 0
			payload.token <- struct{}{}
			<-payload.token
		}
		Step(bodies, advTime)
		currentTime = currentTime.Add(advTime)

		iterCount++
	}
}
