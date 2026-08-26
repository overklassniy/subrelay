// Package i18n provides minimal internationalization for the tray and
// dialog UI with Russian and English dictionaries.
//
// The active language is selected once at startup from settings and can
// be switched at runtime via SetLanguage, which is used to rebuild the
// tray menu after a language change.
package i18n

import "sync"

// Language identifiers supported by the application.
const (
	LangRU = "ru"
	LangEN = "en"
)

// Lang is the active language code.
var (
	lang   = LangRU
	langMu sync.RWMutex
)

// dict maps a translation key to its text per language.
var dict = map[string]map[string]string{
	"app.name": {
		LangRU: "Subrelay",
		LangEN: "Subrelay",
	},
	"tray.status.idle": {
		LangRU: "Ожидание",
		LangEN: "Idle",
	},
	"tray.status.running": {
		LangRU: "Работает: %d узлов, обновлено %s",
		LangEN: "Running: %d nodes, updated %s",
	},
	"tray.status.error": {
		LangRU: "Ошибка: %s",
		LangEN: "Error: %s",
	},
	"tray.nodes": {
		LangRU: "Узлы",
		LangEN: "Nodes",
	},
	"tray.balancers": {
		LangRU: "Балансировщики",
		LangEN: "Balancers",
	},
	"tray.balancer.ru.socks": {
		LangRU: "RU SOCKS",
		LangEN: "RU SOCKS",
	},
	"tray.balancer.ru.http": {
		LangRU: "RU HTTP",
		LangEN: "RU HTTP",
	},
	"tray.balancer.nonru.socks": {
		LangRU: "Не-RU SOCKS",
		LangEN: "Non-RU SOCKS",
	},
	"tray.balancer.nonru.http": {
		LangRU: "Не-RU HTTP",
		LangEN: "Non-RU HTTP",
	},
	"tray.refresh": {
		LangRU: "Обновить сейчас",
		LangEN: "Refresh now",
	},
	"tray.nodes_window": {
		LangRU: "Список узлов",
		LangEN: "Nodes list",
	},
	"tray.settings": {
		LangRU: "Настройки",
		LangEN: "Settings",
	},
	"tray.logs": {
		LangRU: "Журнал",
		LangEN: "Logs",
	},
	"tray.exit": {
		LangRU: "Выход",
		LangEN: "Exit",
	},
	"tray.copied": {
		LangRU: "Скопировано: %s",
		LangEN: "Copied: %s",
	},
	"settings.title": {
		LangRU: "Настройки",
		LangEN: "Settings",
	},
	"settings.subscription.url": {
		LangRU: "URL подписки",
		LangEN: "Subscription URL",
	},
	"settings.update.interval": {
		LangRU: "Интервал обновления (мин)",
		LangEN: "Update interval (min)",
	},
	"settings.autostart": {
		LangRU: "Запускать с системой",
		LangEN: "Start with system",
	},
	"settings.language": {
		LangRU: "Язык",
		LangEN: "Language",
	},
	"settings.ports": {
		LangRU: "Порты",
		LangEN: "Ports",
	},
	"settings.ru.socks": {
		LangRU: "RU SOCKS порт",
		LangEN: "RU SOCKS port",
	},
	"settings.ru.http": {
		LangRU: "RU HTTP порт",
		LangEN: "RU HTTP port",
	},
	"settings.nonru.socks": {
		LangRU: "Не-RU SOCKS порт",
		LangEN: "Non-RU SOCKS port",
	},
	"settings.nonru.http": {
		LangRU: "Не-RU HTTP порт",
		LangEN: "Non-RU HTTP port",
	},
	"settings.socks.start": {
		LangRU: "Начало диапазона SOCKS",
		LangEN: "SOCKS range start",
	},
	"settings.http.start": {
		LangRU: "Начало диапазона HTTP",
		LangEN: "HTTP range start",
	},
	"settings.advanced": {
		LangRU: "Дополнительно",
		LangEN: "Advanced",
	},
	"settings.urltest.interval": {
		LangRU: "Интервал urltest (сек)",
		LangEN: "urltest interval (sec)",
	},
	"settings.urltest.tolerance": {
		LangRU: "Допуск urltest (мс)",
		LangEN: "urltest tolerance (ms)",
	},
	"settings.urltest.url": {
		LangRU: "URL теста urltest",
		LangEN: "urltest URL",
	},
	"settings.headers": {
		LangRU: "Заголовки запроса",
		LangEN: "Request headers",
	},
	"settings.connect": {
		LangRU: "Подключиться",
		LangEN: "Connect",
	},
	"settings.save": {
		LangRU: "Сохранить",
		LangEN: "Save",
	},
	"settings.cancel": {
		LangRU: "Отмена",
		LangEN: "Cancel",
	},
	"nodes.title": {
		LangRU: "Узлы",
		LangEN: "Nodes",
	},
	"nodes.name": {
		LangRU: "Имя",
		LangEN: "Name",
	},
	"nodes.transport": {
		LangRU: "Транспорт",
		LangEN: "Transport",
	},
	"nodes.ru": {
		LangRU: "RU",
		LangEN: "RU",
	},
	"nodes.nonru": {
		LangRU: "не RU",
		LangEN: "non-RU",
	},
	"nodes.socks": {
		LangRU: "SOCKS",
		LangEN: "SOCKS",
	},
	"nodes.http": {
		LangRU: "HTTP",
		LangEN: "HTTP",
	},
	"nodes.copy": {
		LangRU: "Копировать",
		LangEN: "Copy",
	},
	"nodes.search": {
		LangRU: "Поиск",
		LangEN: "Search",
	},
	"logs.title": {
		LangRU: "Журнал",
		LangEN: "Logs",
	},
	"logs.open.file": {
		LangRU: "Открыть файл",
		LangEN: "Open file",
	},
	"logs.dump.config": {
		LangRU: "Дамп конфигурации",
		LangEN: "Dump config",
	},
	"logs.clear": {
		LangRU: "Очистить",
		LangEN: "Clear",
	},
	"notify.update.success": {
		LangRU: "Подписка обновлена: %d узлов",
		LangEN: "Subscription updated: %d nodes",
	},
	"notify.update.error": {
		LangRU: "Ошибка обновления подписки: %s",
		LangEN: "Subscription update error: %s",
	},
	"notify.fetch.error": {
		LangRU: "Ошибка получения подписки: %s",
		LangEN: "Subscription fetch error: %s",
	},
	"wizard.title": {
		LangRU: "Добро пожаловать в Subrelay",
		LangEN: "Welcome to Subrelay",
	},
	"wizard.prompt": {
		LangRU: "Введите URL подписки для начала",
		LangEN: "Enter the subscription URL to begin",
	},
	"settings.tab.general": {
		LangRU: "Общие",
		LangEN: "General",
	},
	"settings.tab.ports": {
		LangRU: "Порты",
		LangEN: "Ports",
	},
	"settings.tab.urltest": {
		LangRU: "Проверка URL",
		LangEN: "URL test",
	},
	"settings.tab.headers": {
		LangRU: "Заголовки",
		LangEN: "Headers",
	},
	"settings.section.balancer.ru": {
		LangRU: "Балансировщик RU",
		LangEN: "RU balancer",
	},
	"settings.section.balancer.nonru": {
		LangRU: "Балансировщик не-RU",
		LangEN: "Non-RU balancer",
	},
	"settings.section.perNode": {
		LangRU: "Диапазоны портов узлов",
		LangEN: "Per-node port ranges",
	},
	"validation.required": {
		LangRU: "Поле обязательно для заполнения",
		LangEN: "This field is required",
	},
	"validation.int": {
		LangRU: "Введите целое число",
		LangEN: "Enter a whole number",
	},
	"validation.range": {
		LangRU: "Значение должно быть от %d до %d",
		LangEN: "Value must be between %d and %d",
	},
	"validation.url": {
		LangRU: "Введите корректный URL (http:// или https://)",
		LangEN: "Enter a valid URL (http:// or https://)",
	},
	"nodes.refresh": {
		LangRU: "Обновить",
		LangEN: "Refresh",
	},
	"nodes.empty": {
		LangRU: "Узлы не найдены",
		LangEN: "No nodes found",
	},
	"logs.refresh": {
		LangRU: "Обновить",
		LangEN: "Refresh",
	},
	"logs.autorefresh": {
		LangRU: "Автообновление",
		LangEN: "Auto-refresh",
	},
	"logs.level.all": {
		LangRU: "Все уровни",
		LangEN: "All levels",
	},
	"logs.dump.saved": {
		LangRU: "Конфигурация сохранена: %s",
		LangEN: "Configuration saved: %s",
	},
	"logs.dump.empty": {
		LangRU: "Конфигурация еще не построена",
		LangEN: "No configuration has been built yet",
	},
	"logs.dump.error": {
		LangRU: "Не удалось сохранить конфигурацию: %s",
		LangEN: "Failed to save configuration: %s",
	},
	"logs.open.error": {
		LangRU: "Не удалось открыть файл: %s",
		LangEN: "Failed to open file: %s",
	},
	"logs.cleared": {
		LangRU: "Журнал очищен",
		LangEN: "Log cleared",
	},
	"settings.check_updates": {
		LangRU: "Проверить обновления",
		LangEN: "Check for updates",
	},
	"update.checking": {
		LangRU: "Проверка обновлений...",
		LangEN: "Checking for updates...",
	},
	"update.available.verdict": {
		LangRU: "Доступна новая версия %s (текущая %s).",
		LangEN: "New version %s available (current %s).",
	},
	"update.open_page": {
		LangRU: "Открыть страницу релиза",
		LangEN: "Open release page",
	},
	"update.uptodate.verdict": {
		LangRU: "Установлена последняя версия (%s).",
		LangEN: "Latest version is installed (%s).",
	},
	"update.no_releases.verdict": {
		LangRU: "Релизы еще не опубликованы.",
		LangEN: "No releases have been published yet.",
	},
	"update.error.verdict": {
		LangRU: "Ошибка проверки: %s",
		LangEN: "Check failed: %s",
	},
	"update.available.title": {
		LangRU: "Доступна новая версия",
		LangEN: "New version available",
	},
	"update.available.notify": {
		LangRU: "Доступен Subrelay %s. Откройте настройки, чтобы обновить.",
		LangEN: "Subrelay %s is available. Open settings to update.",
	},
}

// SetLanguage sets the active language. Unknown codes fall back to
// English.
func SetLanguage(code string) {
	langMu.Lock()
	defer langMu.Unlock()
	if code == LangRU || code == LangEN {
		lang = code
		return
	}
	lang = LangEN
}

// Language returns the active language code.
func Language() string {
	langMu.RLock()
	defer langMu.RUnlock()
	return lang
}

// T returns the translated text for key in the active language. When the
// key is unknown or the active language lacks a translation, the English
// text is returned; when no translation exists at all, the key itself is
// returned.
func T(key string) string {
	langMu.RLock()
	defer langMu.RUnlock()
	entry, ok := dict[key]
	if !ok {
		return key
	}
	if text, ok := entry[lang]; ok && text != "" {
		return text
	}
	if text, ok := entry[LangEN]; ok && text != "" {
		return text
	}
	return key
}
