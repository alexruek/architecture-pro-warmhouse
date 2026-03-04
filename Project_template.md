# Задание 1. Анализ и планирование

### 1. Описание функциональности монолитного приложения

**Управление отоплением:**

- Пользователи могут включать и отключать отопление в конкретном помещении
- Пользователи могут задавать целевую температуру для помещения
- Система применяет управляющие команды к физическим устройствам отопления

**Мониторинг температуры:**

- Пользователи могут просматривать текущие показания датчиков в разрезе комнат
- Система принимает показания от IoT-датчиков (Living Room, Bedroom, Kitchen и др.)
- Система хранит историю показаний в базе данных

**Управление датчиками (CRUD):**

- Регистрация нового датчика (POST /api/v1/sensors)
- Получение списка всех датчиков и конкретного датчика по ID
- Обновление конфигурации датчика и его текущего значения/статуса
- Удаление датчика из системы

### 2. Анализ архитектуры монолитного приложения

- Язык программирования: Go
- База данных: PostgreSQL (единая, общая для всей бизнес-логики)
- Взаимодействие с клиентом: синхронное, REST HTTP API (порт 8080)
- Взаимодействие с устройствами: синхронные HTTP-запросы (pull/push от датчиков)
- Деплой: Docker-контейнер + docker-compose
- Структура: весь код (HTTP-обработчики, бизнес-логика, доступ к БД) находится
  в едином Go-приложении без разделения на независимые модули

### 3. Определение доменов и границы контекстов


| Домен        | Ответственность                                                                            | Ключевые сущности              |
| ----------------- | --------------------------------------------------------------------------------------------------------- | ---------------------------------------------- |
| Device Management | Регистрация, конфигурация и жизненный цикл устройств        | Device, DeviceType, Location                   |
| Telemetry         | Сбор, хранение и предоставление показаний датчиков            | TelemetryRecord, Sensor, SensorValue           |
| Heating Control   | Бизнес-логика управления отоплением                                       | HeatingZone, TargetTemperature, HeatingCommand |
| User & Auth       | Аутентификация, авторизация, управление доступом к домам | User, House, Permission                        |
| Notifications     | Оповещения об аномалиях, событиях и статусах устройств     | Alert, Notification, Threshold                 |

### 4. Проблемы монолитного решения

- Невозможно независимо масштабировать отдельные части —
  например, модуль телеметрии при росте числа датчиков тянет за собой всё приложение
- Сбой в любом модуле (например, в обработчике отопления)
  останавливает работу всей системы, включая несвязанные функции
- Добавление новых типов устройств (свет, ворота, камеры)
  требует изменений во всём монолите, высок риск регрессий
- Все домены работают с одной БД и одними структурами данных,
  изменение схемы для одного домена затрагивает остальные, что говорит о высокой связности
- Любое изменение требует пересборки и перезапуска
  всего приложения, что увеличивает риски и downtime

### 5. Визуализация контекста системы — диаграмма С4

![C4 Context Diagram](schemas/c4-context.svg)

# Задание 2. Проектирование микросервисной архитектуры

**Декомпозиция монолита на микросервисы:**


| Домен (As-Is монолит)           | Микросервис (To-Be) | Источник                                               |
| ------------------------------------------- | ------------------------------ | -------------------------------------------------------------- |
| Управление отоплением   | `heating-service`              | Из монолита                                          |
| Мониторинг температуры | `telemetry-service`            | Из монолита                                          |
| —                                          | `user-service`                 | Новый (SaaS, самообслуживание)            |
| —                                          | `device-service`               | Новый (самоподключение устройств) |
| —                                          | `lighting-service`             | Новый (управление светом)                 |
| —                                          | `gate-service`                 | Новый (управление воротами)             |
| —                                          | `camera-service`               | Новый (видеонаблюдение)                    |
| —                                          | `scenario-service`             | Новый (пользовательские сценарии) |
| —                                          | `notification-service`         | Новый (уведомления)                            |

![C4 Containers Diagram](schemas/c4-containers.svg)

Компоненты device-service — самого центрального сервиса, без которого SaaS невозможен:

![C4 Components Diagram](schemas/c4-components.svg)

Критичный поток: получение телеметрии → автоматическое управление отоплением. Это ключевая бизнес-логика, объединяющая несколько сервисов:

![C4 Code Diagram](schemas/c4-code.svg)

# Задание 3. Разработка ER-диаграммы

![ER-diagram](schemas/er-diagram.svg)

# Задание 4. Создание и документирование API

### 1. Тип API

Для синхронных операций (управление устройствами, получение телеметрии) используется REST API (OpenAPI 3.0) — предсказуемое взаимодействие с немедленным ответом.

Для асинхронных событий (поступление телеметрии, смена статусов устройств) используется AsyncAPI 2.6 поверх Kafka — это позволяет сервисам быть независимыми и не блокировать друг друга при высокой нагрузке.

### 2. Документация API

- REST API (Device Service + Telemetry Service): [api/openapi.yaml](./api/openapi.yaml)
- Async Events (Kafka topics): [api/asyncapi.yaml](./api/asyncapi.yaml)

# Задание 5. Работа с docker и docker-compose

Перейдите в apps.

Там находится приложение-монолит для работы с датчиками температуры. В README.md описано как запустить решение.

Вам нужно:

1) сделать простое приложение temperature-api на любом удобном для вас языке программирования, которое при запросе /temperature?location= будет отдавать рандомное значение температуры.

Locations - название комнаты, sensorId - идентификатор названия комнаты

```
	// If no location is provided, use a default based on sensor ID
	if location == "" {
		switch sensorID {
		case "1":
			location = "Living Room"
		case "2":
			location = "Bedroom"
		case "3":
			location = "Kitchen"
		default:
			location = "Unknown"
		}
	}

	// If no sensor ID is provided, generate one based on location
	if sensorID == "" {
		switch location {
		case "Living Room":
			sensorID = "1"
		case "Bedroom":
			sensorID = "2"
		case "Kitchen":
			sensorID = "3"
		default:
			sensorID = "0"
		}
	}
```

2) Приложение следует упаковать в Docker и добавить в docker-compose. Порт по умолчанию должен быть 8081
3) Кроме того для smart_home приложения требуется база данных - добавьте в docker-compose файл настройки для запуска postgres с указанием скрипта инициализации ./smart_home/init.sql

Для проверки можно использовать Postman коллекцию smarthome-api.postman_collection.json и вызвать:

- Create Sensor
- Get All Sensors

Должно при каждом вызове отображаться разное значение температуры

Ревьюер будет проверять точно так же.

### Задание 6. MVP с новыми микросервисами

### Новые микросервисы


| Микросервис | Язык         | Порт | Описание                                                                                             |
| ---------------------- | ---------------- | -------- | ------------------------------------------------------------------------------------------------------------ |
| `device-service`       | Go               | 8082     | Регистрация и управление устройствами, хранение состояний |
| `telemetry-service`    | Python (FastAPI) | 8083     | Прием и хранение телеметрии, интеграция с монолитом              |

### Интеграция с монолитом

`telemetry-service` при получении новой телеметрии обновляет значение датчика в монолите
через `PATCH /api/v1/sensors/:id/value` — синхронный HTTP-вызов. Если монолит недоступен,
запись телеметрии не блокируется.

### Запуск

```bash
cd apps
docker-compose up --build


```


| Эндпоинт                                                                                            | Сервис                |
| ----------------------------------------------------------------------------------------------------------- | --------------------------- |
| GET[http://localhost:8080/api/v1/sensors](http://localhost:8080/api/v1/sensors)                             | Smart Home (монолит) |
| GET[http://localhost:8081/temperature?location=Kitchen](http://localhost:8081/temperature?location=Kitchen) | Temperature API             |
| GET[http://localhost:8082/devices](http://localhost:8082/devices)                                           | Device Service              |
| POST[http://localhost:8083/telemetry](http://localhost:8083/telemetry)                                      | Telemetry Service           |
| GET[http://localhost:8083/docs](http://localhost:8083/docs)                                                 | Swagger UI (FastAPI)        |

<pre class="not-prose w-full rounded font-mono text-sm font-extralight"><div class="codeWrapper bg-subtle text-light selection:text-super selection:bg-super/10 my-md relative flex flex-col rounded-lg font-mono text-sm font-medium"><div class="translate-y-xs -translate-x-xs bottom-xl mb-xl flex h-0 items-start justify-end sm:sticky sm:top-xs"><div class="overflow-hidden border-subtlest ring-subtlest divide-subtlest bg-base rounded-full"><div class="border-subtlest ring-subtlest divide-subtlest bg-subtle"></div></div></div><div class="-mt-xl"><br class="Apple-interchange-newline"/></div></div></pre>

Swagger UI для `telemetry-service` будет доступен по адресу `http://localhost:8083/docs`