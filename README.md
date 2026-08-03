# HSA (Home Secure Auth)
[![codecov](https://codecov.io/gh/whicu/hsa/graph/badge.svg)](https://codecov.io/gh/whicu/hsa)
[![CI](https://github.com/whicu/hsa/actions/workflows/ci.yml/badge.svg)](https://github.com/whicu/hsa/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/whicu/hsa)](https://goreportcard.com/report/github.com/whicu/hsa)
[![Go Version](https://img.shields.io/github/go-mod/go-version/whicu/hsa)](go.mod)
Микросервис аутентификации, управления инвайтами и хранения обёрнутых ключей шифрования для домашнего сервисa.

Сервис реализует подход Zero-Knowledge: он отвечает за проверку подлинности пользователя через Passkey (WebAuthn), но никогда не имеет доступа к необработанным ключам шифрования (DEK).

## Основное назначение

- Аутентификация исключительно по Passkey (WebAuthn) с обязательной поддержкой расширения prf.

- Регистрация по P2P-инвайтам (цепочка доверия от root-пользователя).

- Хранение wrapped_keys (обёрнутых DEK), без возможности их расшифровки на стороне бэкенда.

- Выдача Access JWT для авторизации запросов к storage-svc.

- Управление Refresh-токенами и возможность каскадного отзыва сессий по цепочке приглашений.