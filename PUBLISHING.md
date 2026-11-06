# Публикация

Корень публикуемого проекта — эта папка, без соседних Windows-скриптов.
Сохранена история исходного ShadowsocksGostPlugin; origin указывает на
https://github.com/nafe93/ShadowsocksGostPluginANTIDPI.git.
Исходный репозиторий сохранён как upstream; отправляйте изменения в origin.

В Git входят исходники, Gradle Wrapper, лицензии и документация.
APK, кеши, NDK, Go toolchain, local.properties и ключи подписи исключены.
Готовые APK и SHA256SUMS.txt собраны в локальной папке release-artifacts/.
Загрузите их как assets GitHub Release; текст релиза — RELEASE_NOTES.md.

Workflow .github/workflows/android.yml собирает проект на JDK 17 для четырёх
архитектур, запускает тесты и сохраняет APK как Actions artifacts.
Это development-сборки с временным debug-ключом runner: они не публикуются
автоматически в Releases и не предназначены для обновления установленного
плагина, подписанного другим ключом.

Для постоянных релизов создайте и сохраните release-ключ вне репозитория.
Параметры SignKeyName, SignKeyPass и SignKeyStorePath задаются в исключённом
из Git local.properties. После этого используйте ./gradlew assembleRelease.
