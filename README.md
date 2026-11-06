# Shadowsocks GOST + AntiDPI для Android

Отдельный плагин `gost-antidpi` для Shadowsocks Android, на основе соседнего
ShadowsocksGostPlugin. Может устанавливаться рядом с исходным плагином.
Минимальная версия Android — 8.0 (API 26).

AntiDPI работает нативно: ByeDPI 17.3 на C собирается Android NDK и входит в APK.
Python, Windows EXE, отдельное приложение ByeDPI, root и загрузки при запуске
не нужны. Новых Java/Gradle-библиотек интеграция не добавляет. Зависимости
исходного GOST-плагина сохранены; исходники Go/NDK и кеши сборки в APK не входят.
C-движок после strip занимает 58–67 КиБ для одной архитектуры.

## Использование

1. Установите APK и выберите **GOST + AntiDPI** в плагинах Shadowsocks.
2. Настройте `-L` и удалённые узлы `-F`, как в исходном GOST-плагине.
3. В поле AntiDPI выберите `aggressive`, `balanced`, `compat`, `disorder`,
   `fake`, `oob`, `tlsrec` или `off`. Начальное значение — `off`.
4. Сохраните настройки и переподключитесь.

Пример аргументов для редактора плагина:

```text
-L ss://none@${local.host}:${local.port}
-F socks5+wss://${remote.host}:${remote.port}?path=/assets/api/socket
```

Авторизацию и адрес сервера укажите в своём профиле/аргументах GOST.
Действующие пароли из соседних Windows-скриптов в новый проект не перенесены.

Путь трафика: Shadowsocks → GOST → локальный SOCKS5 ByeDPI → сервер GOST.
Локальный порт выбирается при запуске. При VPN-режиме ByeDPI передаёт исходящие
сокеты в `protect_path` Shadowsocks до подключения. При завершении GOST ядро
останавливает дочерний ByeDPI; при аварии ByeDPI завершается GOST.

Стратегии и fallback-группы перенесены из `../antipdi.py`, включая `--proto`
для каждой группы, TTL 8 и timeout 3. Они применяются к видимому оператору
HTTP/TLS TCP-соединению с первым сервером GOST. UDP/QUIC и DNS-обход не
добавлены. Работа конкретной стратегии зависит от сети и сервера.

В этой версии AntiDPI поддерживает конфигурацию GOST через `-L`/`-F`.
При включённом AntiDPI оставьте `config.yaml` пустым и не добавляйте `-C`:
YAML-цепочки отвергаются явно. При `off` исходный механизм YAML сохранён.
Внешние файлы сертификатов можно использовать как раньше.

## Сборка

Нужны Linux x86_64, JDK 17, Android SDK 29, Build Tools 34.0.0 и
NDK 26.2.11394342. Укажите SDK через `ANDROID_HOME` либо `local.properties`. Кеши Go по умолчанию
размещаются в `.go_build/`, вне APK.

```bash
./gradlew assembleDebug
```

Скрипт `build.gost.sh` использует Go 1.25.0, GOST 3.2.4 и исходный патч Go
для защиты сокетов. Недостающие Go/GOST/NDK загружаются только при сборке.
Для release-подписи используются прежние параметры `SignKeyName`,
`SignKeyPass`, `SignKeyStorePath` в `local.properties`; для debug они не нужны.
APK находятся в `app/build/outputs/apk/debug/`. Для каждой архитектуры
создаётся отдельный APK: обычно телефону нужен `arm64-v8a`, старым
устройствам — `armeabi-v7a`, эмуляторам — `x86`/`x86_64`. Каждый APK
содержит только свою пару GOST/ByeDPI, без остальных архитектур.

## Проверки

```bash
# Изолированные проверки генератора аргументов (без загрузки GOST):
.go_build/go/bin/go test gost/gost_helper/antidpi.go gost/gost_helper/antidpi_test.go

# Проверка нативного C-движка на машине сборки; Python нужен только для теста:
make -C antidpi/vendor
python3 antidpi/tests/smoke.py

# Проверка завершения дочернего процесса при остановке GOST:
BYEDPI_TEST_ENGINE="$PWD/antidpi/vendor/ciadpi" .go_build/go/bin/go test \
  gost/gost_helper/antidpi.go gost/gost_helper/antidpi_test.go \
  gost/gost_helper/lifecycle_test.go
```

Smoke-тест проверяет запуск всех стратегий и SOCKS5, затем HTTP-передачу
в режиме off. Это не проверка обхода блокировок у оператора.
Тест VPN-защиты и реального обхода требует Android-устройства с Shadowsocks.

Лицензии и происхождение исходников: [antidpi/NOTICE.md](antidpi/NOTICE.md).
