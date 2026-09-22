package web

import (
	"html/template"
	"io/fs"
	"net/http"
	"sync"
	"time"

	"github.com/Metal-Bat/sidequest/internal/quest"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	IsAdmin  bool
	ID       int64
	Username string
	XP       int
}

type key int

const userKey key = 0
const csrfKey key = 1

type Data struct {
	User                                     User
	CSRF, Page, Error, Message, Username     string
	Title, Description, Difficulty, Estimate string
	Quests                                   []quest.Quest
	Selected                                 *quest.Quest
	Minutes                                  int
	OOB                                      bool
	Changed                                  quest.Quest
	Action                                   string
}

type attempt struct {
	count int
	until time.Time
}

type App struct {
	pool      *pgxpool.Pool
	quests    quest.Store
	templates *template.Template
	secure    bool
	dummy     []byte
	mu        sync.Mutex
	attempts  map[string]attempt
	hashing   chan struct{}
}

func New(pool *pgxpool.Pool, files fs.FS, secure bool) (http.Handler, error) {
	t, err := template.New("").Funcs(templateFunctions()).ParseFS(files, "templates/*.html")
	if err != nil {
		return nil, err
	}
	dummy, err := bcrypt.GenerateFromPassword([]byte(randomToken()), 12)
	if err != nil {
		return nil, err
	}
	a := &App{
		pool:      pool,
		quests:    quest.Store{Pool: pool},
		templates: t,
		secure:    secure,
		dummy:     dummy,
		attempts:  make(map[string]attempt),
		hashing:   make(chan struct{}, 4),
	}
	return a.routes(files)
}
