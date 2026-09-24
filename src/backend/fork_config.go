package magitrickle

import (
	"errors"
	"fmt"
	"os"

	"github.com/rs/zerolog/log"
	"go.yaml.in/yaml/v2"
)

// Настройки форка лежат в отдельном файле: оригинальная версия его не читает и не перезаписывает,
// поэтому config.yaml остаётся полностью совместимым с оригиналом
const forkCfgFileLocation = cfgFolderLocation + "/fork.yaml"

const defaultForkConfig = `# Настройки форка MagiTrickle. Оригинальная версия этот файл не читает.
# Изменения применяются после перезапуска сервиса.

# Политики доступа Keenetic, трафик устройств которых MagiTrickle не маршрутизирует.
# Указывается название из веб-интерфейса Keenetic (регистр не важен) или системное имя вида Policy4.
# Пустой список [] отключает исключение.
bypassPolicies:
  - noMT
`

type forkConfig struct {
	BypassPolicies []string `yaml:"bypassPolicies"`
}

// loadForkConfig читает настройки форка; если файла нет, создаёт его со значениями по умолчанию
func loadForkConfig(path string) (forkConfig, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		data = []byte(defaultForkConfig)
		if err := os.WriteFile(path, data, 0600); err != nil {
			log.Warn().Err(err).Str("path", path).Msg("failed to create fork config file")
		}
	} else if err != nil {
		return forkConfig{}, fmt.Errorf("failed to read fork config file: %w", err)
	}

	var cfg forkConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return forkConfig{}, fmt.Errorf("failed to parse fork config file: %w", err)
	}
	return cfg, nil
}
