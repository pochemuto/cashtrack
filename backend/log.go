package cashtrack

import (
	"os"
	"time"

	"github.com/rs/zerolog"
)

var log zerolog.Logger

func init() {
	log = zerolog.New(
		zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.DateTime},
	).Level(zerolog.TraceLevel).With().Timestamp().Caller().Logger()
}
