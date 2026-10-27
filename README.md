# Dialog Helper Service

API platform for real-time response suggestions for AAC clients. This repo contains two Go services:

- `inference`: public API for `/v1/infer`.
- `storage`: internal/admin API for config, API keys, and persistence.

Docs live in `docs/` and cover setup, deployment, and operations.

## Quick links
- `docs/01-overview.md`
- `docs/02-environment.md`
- `docs/03-yc-setup.md`
- `docs/04-db-schema.md`
- `docs/05-deploy.md`
- `docs/06-api.md`
- `docs/07-local-dev.md`
- `docs/08-testing.md`

## Repository layout
- `cmd/inference`: inference service entrypoint
- `cmd/storage`: storage/admin service entrypoint
- `internal/`: shared packages
- `api/`: OpenAPI and API Gateway config
- `scripts/`: bootstrap and helper scripts
- `testdata/`: small sample files for local tests

## Status
Work in progress.

## Лицензия

Собственный программный код LINKa в этом репозитории распространяется на условиях
GNU General Public License версии 3 или, по выбору получателя, любой более поздней
версии (SPDX: `GPL-3.0-or-later`). Полный неизменённый текст GPLv3: [LICENSE](LICENSE).

Разрешены использование, форки, изменения и коммерческое использование. При
распространении покрываемого GPL кода и производных программ необходимо соблюдать
GPL: сохранять уведомления об авторстве, лицензии и отсутствии гарантий, отмечать
изменения и их даты, предоставлять соответствующий исходный код получателям и
лицензировать покрываемую производную программу на условиях GPL. Личные изменения,
которые не распространяются, публиковать не требуется. Гарантии не предоставляются
в пределах разделов 15–16 GPLv3.

Авторы и правообладатели сохраняют права на свои вклады; существующие имена и годы
авторства сохраняются в исходниках, уведомлениях и истории Git. Поддержка или участие
АНО «Линка» сами по себе не означают передачу ей исключительных прав. Copyright FSF
в тексте GPL относится к тексту лицензии, а не к авторству программы. Ранее
предоставленные права на версии под MIT/BSD/ISC не отзываются.

Эта GPL-декларация относится только к собственному программному коду и не
перелицензирует сторонний код, SDK/DLL Tobii, Unity-компоненты, зависимости,
изображения, звуки, шрифты, бренды и учебные материалы. Их исходные лицензии,
атрибуция и ограничения сохраняются; права на товарные знаки не предоставляются.
Публичная доступность файла не заменяет разрешение правообладателя. Совместимость
всей готовой сборки с GPL и права на материалы без подтверждённой лицензии этим
изменением не устанавливаются.
