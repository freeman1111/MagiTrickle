# Форк MagiTrickle: как обновляться из официального репозитория

Этот репозиторий — форк официального MagiTrickle:

| | |
|---|---|
| Официальный репозиторий (основной) | https://gitlab.com/magitrickle/magitrickle |
| Официальное зеркало на GitHub | https://github.com/MagiTrickle/MagiTrickle |
| Официальный `develop` | тест-сборки (каждый push собирает пакеты в CI) |
| Официальный `main` | релизные сборки (теги `0.8.2`, `0.8.2-rev2` и т.д.) |

GitLab и GitHub-зеркало содержат одинаковые коммиты, поэтому синхронизация идёт с GitLab (первоисточник).

## Схема веток форка

```
GitLab develop ──(зеркало)──▶ upstream ──(merge)──▶ develop ──▶ ваши изменения
```

| Ветка форка | Что в ней | Правило |
|---|---|---|
| `upstream` | Точная копия официального `develop` | **Никогда не коммитить.** Обновляется принудительно (force) |
| `develop` | Официальный код + изменения форка | Основная ветка. Обновляется через `git merge upstream` |
| `feature/*`, `fix/*` | Ваши изменения | Ветвятся от `develop`, вливаются в `develop` |

Ключевая идея: все свои изменения держим только в `develop` (и feature-ветках), а официальный код приезжает
обычным `git merge`. Git сам помнит, что уже сливалось, поэтому при каждом обновлении конфликты возможны
только в тех файлах, которые менялись **и** у нас, **и** в официальном репозитории.

Чтобы конфликтов было меньше:
- не переформатируйте чужой код и не делайте «косметических» правок в файлах upstream;
- изменения форка по возможности выносите в отдельные файлы/пакеты (как `scripts/`, `docs/FORK.md`, `sync-upstream.yml`);
- не трогайте `README.md` и `.gitlab-ci.yml` upstream без необходимости.

## Автоматическое обновление (GitHub Actions)

Workflow `.github/workflows/sync-upstream.yml`:

1. Запускается **ежедневно** (04:17 UTC) и вручную через *Actions → Sync with upstream → Run workflow*.
2. Скачивает официальный `develop` и теги, принудительно обновляет ветку `upstream` в форке.
3. Пытается слить `upstream` в `develop`:
   - конфликтов нет → merge-коммит пушится в `develop` автоматически;
   - есть конфликты → открывается Pull Request `upstream → develop`, в нём видно, что конфликтует.

Ручной запуск позволяет выбрать ветку upstream (`develop` или `main`) и отключить merge (только обновить зеркало).

### Настройка (один раз)

1. *Settings → Actions → General*: убедиться, что Actions включены и *Workflow permissions* = **Read and write**
   (нужно для push и создания PR).
2. *(рекомендуется)* Добавить секрет `SYNC_TOKEN` — Personal Access Token со scope `repo`
   (*Settings → Secrets and variables → Actions*).
   Без него push от `GITHUB_TOKEN` по правилам GitHub **не запускает** `build.yml`, то есть после автосинхронизации
   тест-пакеты не соберутся, пока вы не запушите что-то сами. С `SYNC_TOKEN` сборка стартует автоматически.
3. *(для релизов)* Секрет `RELEASE_TOKEN` используется `build.yml`, чтобы прикреплять пакеты к GitHub Release.

## Ручное обновление (локально)

```sh
scripts/sync-upstream.sh               # upstream develop → ветка upstream → merge в текущую ветку
scripts/sync-upstream.sh --no-merge    # только обновить зеркало
UPSTREAM_REF=main scripts/sync-upstream.sh   # взять официальный релизный main вместо develop
```

Скрипт сам добавит remote `upstream`, если его нет. После успешного merge:

```sh
git push origin develop upstream --tags
```

Если merge упал с конфликтами — разрешить их, `git add`, `git commit`, `git push`.
Отменить: `git merge --abort`.

То же самое вручную, без скрипта:

```sh
git remote add upstream https://gitlab.com/magitrickle/magitrickle.git   # один раз
git fetch --tags upstream develop
git branch -f upstream upstream/develop
git checkout develop
git merge upstream/develop
git push origin develop upstream --tags
```

## Где брать тест-сборки

- **Форк**: `build.yml` собирает пакеты для всех конфигов из `config/*/` на каждый push в любую ветку.
  Готовые `.ipk`/`.apk` лежат в *Actions → Build and Package OPKG → (нужный запуск) → Artifacts*.
- **Официальные**: GitLab-пайплайн ветки `develop`, job `build`. Артефакты живут 1 неделю:
  `https://gitlab.com/magitrickle/magitrickle/-/jobs/artifacts/develop/download?job=build`
  (если ссылка отдаёт 404 — срок истёк, нужно собрать самим или дождаться нового коммита).
- **Официальные релизы**: https://gitlab.com/magitrickle/magitrickle/-/releases

## Отличия форка от официальной версии

### Маршрутизация только с интерфейсов из `link`

В официальной версии `app.link` определяет только, на каких интерфейсах перехватывается DNS, а маркировка
трафика (`mangle PREROUTING`) действует на пакеты с **любого** интерфейса. В форке переход в цепочку группы
добавляется отдельно для каждого интерфейса из `link` (`-i br0 -j MT_...`), поэтому маршрутизируется только
трафик, вошедший через перечисленные интерфейсы. Остальные сегменты (гостевая сеть, VPN-сервер и т.п.)
ходят в интернет напрямую.

```yaml
app:
  link:
    - br0        # домашний сегмент
    - br1        # ещё один сегмент, если нужен
```

### Исключение политик доступа Keenetic (`bypassPolicies`)

Keenetic помечает трафик устройств, привязанных к политике доступа, своей fwmark-меткой ещё до правил
MagiTrickle (цепочка `_NDM_HOTSPOT_PRERT`). Форк умеет запрашивать метку политики по имени через RCI
(`GET /rci/show/ip/policy/<имя>/mark`, как это делает HydraRoute) и ставит в начало цепочки группы
правило `-m mark --mark <метка> -j RETURN`. Устройства из такой политики не получают маршрутизацию
MagiTrickle и ходят «как при голом интернете» (по правилам самой политики).

```yaml
app:
  bypassPolicies:
    - noMT
```

Имя указывается так, как оно отображается в веб-интерфейсе Keenetic (*Приоритеты подключений → Политики
доступа*). Метки запрашиваются при старте сервиса: если политика не найдена или RCI недоступен, в лог
пишется предупреждение (`failed to get policy mark`), а трафик этой политики маршрутизируется как обычно.
После создания или переименования политики нужен `restart` сервиса. На платформах без RCI (OpenWrt,
не-Keenetic Entware) параметр игнорируется с тем же предупреждением.

Проверить, что правила применились:

```sh
iptables -t mangle -S | grep MT_
# -A PREROUTING -i br0 -j MT_xxxxxxxx
# -A MT_xxxxxxxx -m mark --mark 0xffffaa2 -j RETURN
```

## Выпуск релиза форка

1. Обновиться из upstream (см. выше), убедиться, что `develop` собирается.
2. Создать тег и GitHub Release (например `0.8.2-fork1`) из `develop`.
3. `build.yml` на событие `release: published` соберёт пакеты и прикрепит их к релизу (нужен `RELEASE_TOKEN`).

Свои теги называйте с суффиксом (`-fork1`), чтобы они не пересекались с официальными тегами,
которые синхронизация копирует в форк.

## Сборка вручную

См. корневой `README.md` (раздел «Сборка») и `CLAUDE.md`: `cp .config.example .config`, отредактировать
`PLATFORM`/`TARGET`, затем `make`.
