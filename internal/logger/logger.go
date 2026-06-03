package logger

import (
	"os"

	"github.com/sirupsen/logrus"
	"rss-aggregator/internal/config"
)

// Logger est une instance globale de logrus.Logger pour être utilisée dans tout le projet.
var Logger *logrus.Logger

func init() {
	// Initialiser le logger avec les paramètres par défaut
	Logger = logrus.New()

	// Définir le niveau de log (peut être modifié via l'environnement ou la config)
	// Priorité: ENV > Config > Défaut
	logLevel := os.Getenv("LOG_LEVEL")

	// Si non défini dans l'environnement, essayer de charger depuis la config
	if logLevel == "" {
		cfg, err := config.LoadConfig("data/config.yaml")
		if err == nil && cfg.LogLevel != "" {
			logLevel = cfg.LogLevel
		}
	}

	switch logLevel {
	case "debug":
		Logger.SetLevel(logrus.DebugLevel)
	case "info":
		Logger.SetLevel(logrus.InfoLevel)
	case "warn":
		Logger.SetLevel(logrus.WarnLevel)
	case "error":
		Logger.SetLevel(logrus.ErrorLevel)
	case "fatal":
		Logger.SetLevel(logrus.FatalLevel)
	case "panic":
		Logger.SetLevel(logrus.PanicLevel)
	default:
		Logger.SetLevel(logrus.InfoLevel)
	}

	Logger.Infof("Log level set to: %s", logLevel)

	// Configurer le format du log
	Logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
		DisableColors:   false,
	})

	// Définir la sortie (peut être redirigée vers un fichier ou un autre writer)
	Logger.SetOutput(os.Stdout)

	// Configurer le hook pour les erreurs fatales (ex: envoyer une notification)
	Logger.AddHook(&fatalHook{})
}

// fatalHook est un hook personnalisé pour gérer les logs de niveau Fatal.
type fatalHook struct{}

func (hook *fatalHook) Fire(entry *logrus.Entry) error {
	// Ici, tu pourrais ajouter une logique pour notifier les erreurs fatales (ex: Sentry, email, etc.)
	return nil
}

func (hook *fatalHook) Levels() []logrus.Level {
	return []logrus.Level{
		logrus.FatalLevel,
		logrus.PanicLevel,
	}
}

// Debugf utilise le logger global pour afficher un message de debug.
func Debugf(format string, args ...interface{}) {
	Logger.Debugf(format, args...)
}

// Infof utilise le logger global pour afficher un message d'information.
func Infof(format string, args ...interface{}) {
	Logger.Infof(format, args...)
}

// Warnf utilise le logger global pour afficher un message d'avertissement.
func Warnf(format string, args ...interface{}) {
	Logger.Warnf(format, args...)
}

// Errorf utilise le logger global pour afficher un message d'erreur.
func Errorf(format string, args ...interface{}) {
	Logger.Errorf(format, args...)
}

// Fatalf utilise le logger global pour afficher un message fatal et quitter l'application.
func Fatalf(format string, args ...interface{}) {
	Logger.Fatalf(format, args...)
}

// Panicf utilise le logger global pour afficher un message fatal et paniquer.
func Panicf(format string, args ...interface{}) {
	Logger.Panicf(format, args...)
}

// Debugln utilise le logger global pour afficher un message de debug sans formatage.
func Debugln(args ...interface{}) {
	Logger.Debugln(args...)
}

// Infoln utilise le logger global pour afficher un message d'information sans formatage.
func Infoln(args ...interface{}) {
	Logger.Infoln(args...)
}

// Warnln utilise le logger global pour afficher un message d'avertissement sans formatage.
func Warnln(args ...interface{}) {
	Logger.Warnln(args...)
}

// Errorln utilise le logger global pour afficher un message d'erreur sans formatage.
func Errorln(args ...interface{}) {
	Logger.Errorln(args...)
}

// Fatalln utilise le logger global pour afficher un message fatal et quitter l'application sans formatage.
func Fatalln(args ...interface{}) {
	Logger.Fatalln(args...)
}

// Panicln utilise le logger global pour afficher un message fatal et paniquer sans formatage.
func Panicln(args ...interface{}) {
	Logger.Panicln(args...)
}
