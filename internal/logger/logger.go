package logger

import "github.com/sirupsen/logrus"

var Logger logrus.Logger = *logrus.StandardLogger()

func SetLogLevel(rawLevel string) {
	level, err := logrus.ParseLevel(rawLevel)

	if err != nil {
		Logger.Errorf("could not parse log level %v: %v", rawLevel, err)
	}

	Logger.WithFields(logrus.Fields{
		"level": level,
	}).Infoln("setting log level")
	logrus.SetLevel(level)
}
