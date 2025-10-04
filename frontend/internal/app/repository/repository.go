package repository

import (
	"errors"
    "fmt"
    "strings"
)

type Repository struct {
}

func NewRepository() (*Repository, error) {
	return &Repository{}, nil
}

type Observation struct {
	ID_observation int
	Description    string
	Result         float64  // результат заявки в км
}
type Day struct {
	ID          int
	Date        string
	Description string
	FullInfo    string
	Image       string
    EarthRA     float64
    EarthDEC    float64
    BodiesText  string
    AsteroidRA  float64  // координаты астероида
    AsteroidDEC float64
}

type AsteroidData struct {
    RA  float64
    DEC float64
}

var ErrDayNotFound = errors.New("day not found")

var days = []Day{
	{
		ID:          1,
		Date:        "21.02.2025",
		Description: "Позиции небесных тел на 21 февраля 2025 года",
		FullInfo:    "Астрономические координаты Земли, астероидов Bennu, Eros и Vesta на указанную дату. Координаты представлены в экваториальной системе координат.",
		Image:       "http://localhost:9000/pictures/bennu.jpeg",
		EarthRA:     154.2308,
		EarthDEC:    10.6731,
		AsteroidRA:  133.5752,
		AsteroidDEC: 22.4028,
		BodiesText:  "Bennu:\nRA: 133.5752°\nDEC: 22.4028°\nEros:\nRA: 290.8563°\nDEC: -24.9965°\nVesta:\nRA: 200.9185°\nDEC: -1.1438°",
	},
	{
		ID:          2,
		Date:        "20.03.2025",
		Description: "Позиции небесных тел на 20 марта 2025 года",
		FullInfo:    "Астрономические координаты Земли, астероидов Bennu, Eros и Vesta на указанную дату. Координаты представлены в экваториальной системе координат.",
		Image:       "http://localhost:9000/pictures/eros.jpg",
		EarthRA:     179.3340,
		EarthDEC:    0.2892,
		AsteroidRA:  158.8320,
		AsteroidDEC: 11.7454,
		BodiesText:  "Bennu:\nRA: 158.8320°\nDEC: 11.7454°\nEros:\nRA: 300.9850°\nDEC: -21.4905°\nVesta:\nRA: 208.5728°\nDEC: -4.3249°",
	},
	{
		ID:          3,
		Date:        "16.04.2025",
		Description: "Позиции небесных тел на 16 апреля 2025 года",
		FullInfo:    "Астрономические координаты Земли, астероидов Bennu, Eros и Vesta на указанную дату. Координаты представлены в экваториальной системе координат.",
		Image:       "http://localhost:9000/pictures/final.png",
		EarthRA:     204.0161,
		EarthDEC:    -10.0047,
		AsteroidRA:  178.5265,
		AsteroidDEC: 1.0704,
		BodiesText:  "Bennu:\nRA: 178.5265°\nDEC: 1.0704°\nEros:\nRA: 310.5254°\nDEC: -17.4401°\nVesta:\nRA: 216.4333°\nDEC: -7.4845°",
	},
	{
		ID:          4,
		Date:        "13.05.2025",
		Description: "Позиции небесных тел на 13 мая 2025 года",
		FullInfo:    "Астрономические координаты Земли, астероидов Bennu, Eros и Vesta на указанную дату. Координаты представлены в экваториальной системе координат.",
		Image:       "http://localhost:9000/pictures/galileo.jpg",
		EarthRA:     229.7279,
		EarthDEC:    -18.3008,
		AsteroidRA:  195.3633,
		AsteroidDEC: -8.2895,
		BodiesText:  "Bennu:\nRA: 195.3633°\nDEC: -8.2895°\nEros:\nRA: 319.7108°\nDEC: -12.9146°\nVesta:\nRA: 224.5325°\nDEC: -10.5492°",
	},
	{
		ID:          5,
		Date:        "09.06.2025",
		Description: "Позиции небесных тел на 9 июня 2025 года",
		FullInfo:    "Астрономические координаты Земли, астероидов Bennu, Eros и Vesta на указанную дату. Координаты представлены в экваториальной системе координат.",
		Image:       "http://localhost:9000/pictures/idle.jpg",
		EarthRA:     257.0373,
		EarthDEC:    -22.9007,
		AsteroidRA:  211.0160,
		AsteroidDEC: -16.0451,
		BodiesText:  "Bennu:\nRA: 211.0160°\nDEC: -16.0451°\nEros:\nRA: 328.7896°\nDEC: -7.9499°\nVesta:\nRA: 232.8950°\nDEC: -13.4388°",
	},
	{
		ID:          6,
		Date:        "06.07.2025",
		Description: "Позиции небесных тел на 6 июля 2025 года",
		FullInfo:    "Астрономические координаты Земли, астероидов Bennu, Eros и Vesta на указанную дату. Координаты представлены в экваториальной системе координат.",
		Image:       "http://localhost:9000/pictures/lutec.jpg",
		EarthRA:     285.0358,
		EarthDEC:    -22.7161,
		AsteroidRA:  226.5102,
		AsteroidDEC: -22.1504,
		BodiesText:  "Bennu:\nRA: 226.5102°\nDEC: -22.1504°\nEros:\nRA: 338.0380°\nDEC: -2.5617°\nVesta:\nRA: 241.5318°\nDEC: -16.0694°",
	},
	{
		ID:          7,
		Date:        "02.08.2025",
		Description: "Позиции небесных тел на 2 августа 2025 года",
		FullInfo:    "Астрономические координаты Земли, астероидов Bennu, Eros и Vesta на указанную дату. Координаты представлены в экваториальной системе координат.",
			Image:       "http://localhost:9000/pictures/vesta.jpeg",
		EarthRA:     312.0604,
		EarthDEC:    -17.8398,
		AsteroidRA:  242.4963,
		AsteroidDEC: -26.5315,
		BodiesText:  "Bennu:\nRA: 242.4963°\nDEC: -26.5315°\nEros:\nRA: 347.7894°\nDEC: 3.2361°\nVesta:\nRA: 250.4343°\nDEC: -18.3572°",
	},
}

// Простая конфигурация заявки: список ID дней, которые нужно отобразить
var observationDayIDs = map[int][]int{
	1: {1, 3}, // заявка 1 содержит дни: 21.02.2025 (ID=1) и 16.04.2025 (ID=3)
}

func (r *Repository) GetDays() ([]Day, error) {
	// обязательно проверяем ошибки, и если они появились - передаем выше, то есть хендлеру
	if len(days) == 0 {
		return nil, fmt.Errorf("массив пустой")
	}

	return days, nil
}

func (r *Repository) GetDay(id int) (Day, error) {
	// тут у вас будет логика получения нужного дня, тоже наверное через цикл в первой лабе, и через запрос к БД начиная со второй
	days, err := r.GetDays()
	if err != nil {
		return Day{}, err // тут у нас уже есть кастомная ошибка из нашего метода, поэтому мы можем просто вернуть ее
	}

    for _, day := range days {
        if day.ID == id {
            return day, nil // если нашли, то просто возвращаем найденный день без ошибок
        }
    }
    return Day{}, ErrDayNotFound
}

func (r *Repository) GetDaysByDate(date string) ([]Day, error) {
	days, err := r.GetDays()
	if err != nil {
		return []Day{}, err
	}

	var result []Day
	for _, day := range days {
		if strings.Contains(strings.ToLower(day.Date), strings.ToLower(date)) {
			result = append(result, day)
		}
	}

	return result, nil
}


// GetObservationDays возвращает список дней для заявки по массиву ID
func (r *Repository) GetObservationDays(observationID int) ([]Day, error) {
	ids, ok := observationDayIDs[observationID]
    if !ok {
		return []Day{}, nil
    }

    allDays, err := r.GetDays()
    if err != nil {
		return nil, err
    }

    // индекс по ID дня для быстрого доступа
    dayByID := make(map[int]Day, len(allDays))
    for _, d := range allDays {
        dayByID[d.ID] = d
    }

    result := make([]Day, 0, len(ids))
    for _, id := range ids {
        if day, ok := dayByID[id]; ok {
            result = append(result, day)
        }
    }
    return result, nil
}

func (r *Repository) GetObservation(id int) (Observation, error) {
	return observationList[id], nil
}




var observationList = map[int]Observation{
	1: {
		ID_observation: 1,
		Description:    "Астрономическая обсерватория. Наблюдение №1.",
		Result:         58800000.0, // результат в км
	},
}
