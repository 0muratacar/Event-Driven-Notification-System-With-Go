  Proje Ne Yapıyor?

  Günde milyonlarca SMS, Email ve Push bildirim gönderebilen bir sistem. Düşün ki bir e-ticaret sitenin var — kullanıcıya sipariş onayı emaili, SMS ile kargo kodu, telefona push bildirim göndermen gerekiyor. Bu
  sistem tam olarak bunu yapıyor.

  Node.js dünyasındaki karşılığı: Express + BullMQ + Prisma + Redis stack'i ile yapacağın bir bildirim microservice'i.

  ---
  Proje Yapısı (Node.js Karşılaştırmalı)

  Go Projesi                          Node.js Karşılığı
  ─────────────────────────────────────────────────────
  cmd/notifier/main.go              → index.js / server.js (giriş noktası)
  internal/config/                  → dotenv + config dosyası
  internal/domain/                  → models/ veya types/ (TypeScript tipleri)
  internal/repository/              → prisma/queries veya dal/ (veritabanı katmanı)
  internal/service/                 → services/ (iş mantığı)
  internal/validator/               → express-validator / joi / zod
  internal/queue/                   → BullMQ producer/consumer
  internal/worker/                  → BullMQ worker process
  internal/delivery/                → nodemailer, twilio SDK, firebase-admin
  internal/ratelimit/               → rate-limiter-flexible
  internal/api/handler/             → route handler'lar (req, res) => {}
  internal/api/middleware/          → express middleware'ler
  internal/api/websocket/           → socket.io veya ws kütüphanesi
  internal/api/router.go            → express.Router()
  internal/api/server.go            → app.listen()
  internal/tracing/                 → @opentelemetry/node
  migrations/                       → prisma migrate / knex migrations

  ---
  Kullanılan Teknolojiler (Node.js Karşılıkları)

  ┌──────────────────────────┬───────────────────────────────┬─────────────────────────────────┐
  │      Go Kütüphanesi      │       Node.js Karşılığı       │         Ne İşe Yarıyor          │
  ├──────────────────────────┼───────────────────────────────┼─────────────────────────────────┤
  │ net/http (stdlib)        │ Express.js                    │ HTTP sunucu ve route'lar        │
  ├──────────────────────────┼───────────────────────────────┼─────────────────────────────────┤
  │ jackc/pgx/v5             │ pg veya Prisma                │ PostgreSQL bağlantısı           │
  ├──────────────────────────┼───────────────────────────────┼─────────────────────────────────┤
  │ redis/go-redis/v9        │ ioredis                       │ Redis bağlantısı                │
  ├──────────────────────────┼───────────────────────────────┼─────────────────────────────────┤
  │ golang-migrate/migrate   │ prisma migrate / knex migrate │ DB migration'ları               │
  ├──────────────────────────┼───────────────────────────────┼─────────────────────────────────┤
  │ caarlos0/env             │ dotenv + envalid              │ Environment variable'ları okuma │
  ├──────────────────────────┼───────────────────────────────┼─────────────────────────────────┤
  │ log/slog (stdlib)        │ winston / pino                │ Loglama                         │
  ├──────────────────────────┼───────────────────────────────┼─────────────────────────────────┤
  │ go-playground/validator  │ zod / joi / express-validator │ Veri doğrulama                  │
  ├──────────────────────────┼───────────────────────────────┼─────────────────────────────────┤
  │ prometheus/client_golang │ prom-client                   │ Metrik toplama                  │
  ├──────────────────────────┼───────────────────────────────┼─────────────────────────────────┤
  │ go.opentelemetry.io/otel │ @opentelemetry/node           │ Distributed tracing             │
  ├──────────────────────────┼───────────────────────────────┼─────────────────────────────────┤
  │ coder/websocket          │ ws / socket.io                │ WebSocket                       │
  ├──────────────────────────┼───────────────────────────────┼─────────────────────────────────┤
  │ stretchr/testify         │ jest / mocha                  │ Test framework                  │
  ├──────────────────────────┼───────────────────────────────┼─────────────────────────────────┤
  │ google/uuid              │ uuid npm paketi               │ Unique ID üretme                │
  └──────────────────────────┴───────────────────────────────┴─────────────────────────────────┘

  ---
  Adım Adım Ne Yaptım

  ADIM 1: go.mod — Proje Tanımı

  Node.js'teki package.json dosyasının karşılığı. Projenin adını, Go versiyonunu ve tüm bağımlılıkları tanımlar.

  Node.js → package.json + npm install
  Go      → go.mod + go mod tidy

  ADIM 2: internal/config/config.go — Konfigürasyon

  Environment variable'ları okuyup struct'lara (Node.js'teki obje) dolduran dosya.

  Node.js'te şöyle yaparsın:
  // Node.js
  require('dotenv').config()
  const config = {
    port: process.env.PORT || 8080,
    dbUrl: process.env.DATABASE_URL,
    redisUrl: process.env.REDIS_URL
  }

  Go'da aynı şey:
  // Go
  type ServerConfig struct {
      Host string `env:"SERVER_HOST" envDefault:"0.0.0.0"`
      Port int    `env:"SERVER_PORT" envDefault:"8080"`
  }

  env:"SERVER_PORT" → "SERVER_PORT adlı environment variable'ı oku"
  envDefault:"8080" → "yoksa 8080 kullan"

  Go'da struct = Node.js'te interface/type. Go'da class yok, struct var — düz veri taşıyan objeler gibi düşün.

  ---
  ADIM 3: internal/domain/ — Veri Modelleri

  3 dosya var:

  notification.go — Ana veri modeli. TypeScript'te şöyle yazardın:

  // Node.js/TypeScript karşılığı
  type Channel = 'sms' | 'email' | 'push'
  type Priority = 'high' | 'normal' | 'low'
  type Status = 'pending' | 'scheduled' | 'queued' | 'processing' | 'delivered' | 'failed' | 'cancelled'

  interface Notification {
    id: string
    idempotencyKey?: string
    channel: Channel
    priority: Priority
    recipient: string
    subject?: string
    body: string
    status: Status
    scheduledAt?: Date
    attemptCount: number
    maxRetries: number
    // ... devamı
  }

  Go'da type Channel string diyip ardından const ile sabit değerler tanımlıyoruz. TypeScript enum'larının karşılığı.

  template.go — Bildirim şablonları. Mesela "Merhaba {{.Name}}, kodunuz: {{.Code}}" gibi şablonlar tanımlayıp, her bildirimde değişkenleri dolduruyoruz. Node.js'te Handlebars veya EJS ne ise, Go'da text/template o.

  errors.go — Tüm hata tipleri. Node.js'te custom Error class'ları yaparsın ya, burada var ErrNotFound = errors.New("not found") şeklinde tanımlanıyor.

  ---
  ADIM 4: migrations/ — Veritabanı Tabloları

  Prisma migrate veya Knex migration'larının karşılığı. 3 tablo oluşturuyoruz:

  1) notifications tablosu — Ana tablo
  - id → UUID (her bildirimin benzersiz kimliği)
  - idempotency_key → Aynı bildirimi iki kez göndermemek için. Mesela kullanıcı butona iki kez basarsa, aynı key ile ikinci istek reddedilir
  - channel → sms/email/push
  - priority → high/normal/low (yüksek öncelikli bildirimler önce gönderilir)
  - status → pending → queued → processing → delivered/failed
  - scheduled_at → İleri tarihli gönderim için

  2) templates tablosu — Bildirim şablonları
  - Mesela "Hoşgeldin {{.Name}}" şablonu bir kez kaydedilir, her seferinde sadece değişkenler değişir

  3) delivery_attempts tablosu — Gönderim denemeleri
  - Her denemenin sonucu kaydedilir (başarılı mı, hata kodu ne, kaç ms sürdü)

  Index'ler — Veritabanı sorgularını hızlandırmak için. Node.js'te Prisma'da @@index([status]) yazarsın, SQL'de CREATE INDEX ile aynı şey.

  ---
  ADIM 5: internal/repository/ — Veritabanı Katmanı

  Node.js'te Prisma veya raw SQL query'ler yazarsın. Go'da aynısı:

  // Node.js + Prisma
  const notification = await prisma.notification.create({ data: { ... } })
  const found = await prisma.notification.findUnique({ where: { id } })

  // Go
  func (r *NotificationRepository) Create(ctx context.Context, n *domain.Notification) error {
      _, err := r.pool.Exec(ctx, "INSERT INTO notifications (...) VALUES ($1, $2, ...)", ...)
      return err
  }

  Önemli fonksiyonlar:

  - Create — Tek bildirim ekle
  - CreateBatch — Transaction içinde toplu ekleme (ya hepsi eklenir ya hiçbiri)
  - GetByID — ID ile getir
  - List — Cursor pagination ile listele
  - UpdateStatus — Durum güncelle
  - Cancel — İptal et (sadece pending/scheduled/queued durumundakiler iptal edilebilir)
  - FetchScheduledDue — Zamanı gelmiş planlanmış bildirimleri getir (FOR UPDATE SKIP LOCKED — birden fazla worker aynı bildirimi almasın diye)

  Cursor Pagination nedir? — Normal pagination'da "sayfa 500'ü getir" dersen veritabanı önce 499 sayfayı atlayıp 500'e gitmek zorunda. Cursor pagination'da "şu tarih ve ID'den sonrakileri getir" dersin, çok daha
  hızlı. Node.js'te Prisma'nın cursor pagination'ı ile aynı mantık.

  ---
  ADIM 6: internal/validator/validator.go — Doğrulama

  Her kanal için farklı doğrulama kuralları:

  - Email → Geçerli email adresi mi? Subject var mı?
  - SMS → E.164 formatında telefon numarası mı? (+905551234567 gibi). Body 1600 karakterden kısa mı?
  - Push → Device token boş mu? Body 4096 karakterden kısa mı?

  Node.js'te Zod ile şöyle yazardın:
  const smsSchema = z.object({
    recipient: z.string().regex(/^\+[1-9]\d{6,14}$/),
    body: z.string().max(1600)
  })

  ---
  ADIM 7: internal/service/ — İş Mantığı Katmanı

  Handler'lar (controller) ile repository (veritabanı) arasındaki katman. Tüm iş kuralları burada.

  notification.go — NotificationService:

  Create fonksiyonu şunu yapar:
  1. Gelen veriyi doğrula
  2. Template varsa, template'i veritabanından getir ve değişkenleri doldur
  3. Kanal bazlı içerik doğrulaması yap
  4. scheduled_at varsa → status = "scheduled", yoksa → status = "pending"
  5. Veritabanına kaydet
  6. Pending ise → Redis Stream'e gönder (async işlensin)

  template.go — TemplateService:

  Go'nun built-in text/template paketi ile şablon render:
  // Şablon: "Merhaba {{.Name}}, kodunuz: {{.Code}}"
  // Değişkenler: {"Name": "Ali", "Code": "1234"}
  // Sonuç: "Merhaba Ali, kodunuz: 1234"

  QueueProducer interface — Burada önemli bir Go konsepti var:

  type QueueProducer interface {
      Enqueue(ctx context.Context, notificationID uuid.UUID, channel domain.Channel, priority domain.Priority) error
  }

  Node.js/TypeScript'te:
  interface QueueProducer {
    enqueue(notificationId: string, channel: Channel, priority: Priority): Promise<void>
  }

  Interface sayesinde servis katmanı, kuyruğun Redis mi, RabbitMQ mu, yoksa test mock'u mu olduğunu bilmez. Dependency Injection — Node.js'te awilix veya tsyringe ile yaptığın şey.

  ---
  ADIM 8: internal/api/middleware/ — Middleware'ler

  Express middleware'lerinin birebir karşılığı:

  requestid.go — Her isteğe benzersiz ID atar:
  // Node.js karşılığı
  app.use((req, res, next) => {
    req.id = req.headers['x-request-id'] || uuid()
    res.setHeader('X-Request-ID', req.id)
    next()
  })

  logging.go — Her isteği loglar (morgan middleware gibi):
  {"method":"POST","path":"/api/v1/notifications","status":201,"duration_ms":15}

  recovery.go — Panic'leri yakalar (Go'da throw/catch yok, panic/recover var):
  // Node.js karşılığı
  app.use((err, req, res, next) => {
    console.error(err.stack)
    res.status(500).json({ error: 'internal server error' })
  })

  ---
  ADIM 9: internal/api/handler/ — Route Handler'lar

  Express route handler'larının karşılığı.

  // Node.js
  app.post('/api/v1/notifications', async (req, res) => {
    const notification = await notificationService.create(req.body)
    res.status(201).json(notification)
  })

  app.get('/api/v1/notifications/:id', async (req, res) => {
    const n = await notificationService.getById(req.params.id)
    if (!n) return res.status(404).json({ error: 'not found' })
    res.json(n)
  })

  // Go
  func (h *NotificationHandler) Create(w http.ResponseWriter, r *http.Request) {
      var req domain.CreateNotificationRequest
      json.NewDecoder(r.Body).Decode(&req)   // req.body parse
      n, err := h.svc.Create(r.Context(), req)
      writeJSON(w, http.StatusCreated, n)     // res.status(201).json(n)
  }

  response.go — Hata yönetimi helper'ı. Domain hatalarını HTTP status kodlarına çeviriyor:
  - ErrNotFound → 404
  - ErrDuplicate → 409
  - ErrValidation → 400

  health.go — /health ve /ready endpoint'leri:
  - /health → "Uygulama ayakta mı?" (Kubernetes liveness probe)
  - /ready → "Postgres ve Redis bağlı mı?" (Kubernetes readiness probe)

  ---
  ADIM 10: internal/api/router.go — Route Tanımları

  Go 1.22 ile gelen method routing:

  mux.HandleFunc("POST /api/v1/notifications", notificationHandler.Create)
  mux.HandleFunc("GET /api/v1/notifications/{id}", notificationHandler.Get)

  Node.js karşılığı:
  router.post('/api/v1/notifications', notificationHandler.create)
  router.get('/api/v1/notifications/:id', notificationHandler.get)

  {id} → Go 1.22'de path parameter, Express'teki :id ile aynı. r.PathValue("id") ile okunuyor.

  ---
  ADIM 11: internal/queue/ — Redis Streams (Mesaj Kuyruğu)

  Bu en kritik kısım. Node.js'te BullMQ kullanırsın, burada Redis Streams kullanıyoruz.

  Neden kuyruk lazım?

  Kullanıcı POST /notifications dediğinde bildirimi hemen göndermiyoruz. Veritabanına kaydedip kuyruğa atıyoruz. Arka planda worker'lar kuyruktan alıp gönderiyor. Bu sayede:
  - API anında cevap verir (kullanıcı beklemez)
  - Binlerce bildirim aynı anda gelirse sistem çökmez
  - Bir bildirim başarısız olursa tekrar denenir

  3 öncelik kuyruğu:
  notifications:high    → Önce bunlar işlenir
  notifications:normal  → Sonra bunlar
  notifications:low     → En son bunlar

  producer.go — Kuyruğa mesaj ekler:
  // Node.js + BullMQ karşılığı
  await highPriorityQueue.add('notify', { notificationId: '...' })

  // Go + Redis Streams
  client.XAdd(ctx, &redis.XAddArgs{
      Stream: "notifications:high",
      Values: map[string]any{"notification_id": "..."},
  })

  consumer.go — Kuyruktan mesaj okur:

  Consumer Group kavramı — birden fazla worker aynı kuyruğu dinler ama her mesajı sadece bir worker alır. BullMQ'da bu otomatik, Redis Streams'de XREADGROUP komutu ile yapılıyor.

  XAUTOCLAIM — Bir worker çökerse, işlediği mesajları başka worker devralır. At-least-once delivery garantisi.

  dlq.go — Dead Letter Queue (Ölü Mektup Kuyruğu):
  Tüm denemeler başarısız olursa bildirim buraya düşer. İnsan müdahalesi gerekir.

  ---
  ADIM 12: internal/delivery/ — Bildirim Gönderim Katmanı

  provider.go — Interface tanımı:
  type Provider interface {
      Send(ctx context.Context, notification *domain.Notification) Result
      Channel() domain.Channel
  }

  Node.js'te:
  interface Provider {
    send(notification: Notification): Promise<Result>
    channel(): Channel
  }

  email.go, sms.go, push.go — Her kanal için ayrı provider. Hepsi webhook.site'a HTTP POST yapıyor (gerçek projede nodemailer/twilio/firebase olurdu):

  // Node.js karşılığı
  async send(notification) {
    const response = await fetch('https://webhook.site/...', {
      method: 'POST',
      body: JSON.stringify({ to: notification.recipient, body: notification.body })
    })
    return { statusCode: response.status }
  }

  retry.go — Exponential backoff hesaplama:
  Deneme 0 → 1 saniye bekle
  Deneme 1 → 2 saniye
  Deneme 2 → 4 saniye
  Deneme 3 → 8 saniye
  Deneme 4 → 16 saniye
  Deneme 5+ → max 5 dakika

  Her başarısız denemede bekleme süresi ikiye katlanır. API'yi flood etmemek için.

  ---
  ADIM 13: internal/ratelimit/limiter.go — Rate Limiting

  Sliding window algoritması, Lua script ile Redis'te atomik çalışır.

  Neden Lua? Redis tek thread'li. Lua script'i Redis sunucusunda çalışır, bu sayede "kontrol et + ekle" işlemi atomik olur. Node.js'ten iki ayrı komut gönderirsen arada race condition olabilir.

  Kural: Her kanal için saniyede max 100 mesaj. SMS kanalı saniyede 100'den fazla göndermez.

  // Node.js + rate-limiter-flexible karşılığı
  const limiter = new RateLimiterRedis({ points: 100, duration: 1 })
  await limiter.consume('sms') // 101. istekte hata fırlatır

  ---
  ADIM 14: internal/worker/ — Worker Sistemi

  pool.go — Goroutine havuzu. Go'da goroutine = hafif thread. Node.js tek thread'li ama Go'da gerçek paralel çalışma var.

  // Node.js karşılığı — 10 ayrı BullMQ worker
  for (let i = 0; i < 10; i++) {
    new Worker('notifications', processJob)
  }

  // Go — 10 goroutine başlat
  pool.Start(ctx, dispatcher.Work)  // 10 goroutine paralel çalışır

  dispatcher.go — Her worker'ın ana döngüsü:

  Sonsuz döngü:
    1. Redis'ten mesaj oku (XREADGROUP)
    2. Rate limit kontrolü yap (izin yoksa 50ms bekle, tekrar dene)
    3. Bildirimi PostgreSQL'den getir
    4. Status = "processing" yap
    5. Provider ile gönder (HTTP POST webhook.site)
    6. Başarılı → status = "delivered", WebSocket broadcast
    7. Başarısız → retry kaldı mı?
       - Evet → delayed sorted set'e ekle (backoff süresinde tekrar dene)
       - Hayır → status = "failed", DLQ'ya gönder
    8. Mesajı ACK et (kuyruktan sil)

  Prometheus metrikleri de burada toplanıyor:
  - notifications_processed_total — Toplam işlenen bildirim (kanal ve durum bazında)
  - notification_delivery_duration_seconds — Gönderim süresi histogramı
  - notifications_retried_total — Toplam tekrar deneme
  - notifications_dlq_total — DLQ'ya düşen bildirim sayısı

  scheduler.go — Her 5 saniyede Postgres'e bakıp zamanı gelmiş planlanmış bildirimleri kuyruğa atar:

  SELECT * FROM notifications
  WHERE status = 'scheduled' AND scheduled_at <= NOW()
  FOR UPDATE SKIP LOCKED  -- Başka worker aynı satırı almasın

  ---
  ADIM 15: internal/api/websocket/hub.go — Gerçek Zamanlı Güncellemeler

  Client WebSocket ile bağlanıp bir bildirimin durumunu canlı takip edebilir:

  // Client tarafı
  const ws = new WebSocket('ws://localhost:8080/ws/notifications/abc-123')
  ws.onmessage = (e) => {
    console.log(JSON.parse(e.data))
    // { notification_id: "abc-123", status: "delivered" }
  }

  Hub, notification_id → [bağlı client'lar] map'i tutar. Bildirim durumu değiştiğinde tüm bağlı client'lara mesaj gönderir.

  Node.js'te socket.io room'ları ile aynı mantık:
  io.to('notification:abc-123').emit('status', { status: 'delivered' })

  ---
  ADIM 16: internal/tracing/tracing.go — Distributed Tracing

  OpenTelemetry ile bir isteğin tüm yolculuğunu izleyebilirsin:
  API isteği → Service → Queue → Worker → Delivery

  Jaeger veya Grafana Tempo'da görselleştirilir. Node.js'teki @opentelemetry/node ile aynı.

  ---
  ADIM 17: cmd/notifier/main.go — Ana Giriş Noktası

  Node.js'teki index.js / server.js. Her şeyi birbirine bağlar:

  1. Config yükle
  2. Tracing başlat
  3. PostgreSQL'e bağlan
  4. Migration'ları çalıştır
  5. Redis'e bağlan
  6. Repository'leri oluştur
  7. Queue bileşenlerini oluştur
  8. Service'leri oluştur
  9. Delivery provider'ları oluştur
  10. Rate limiter oluştur
  11. WebSocket hub oluştur
  12. HTTP handler'ları oluştur
  13. Router'ı kur
  14. Worker'ları başlat
  15. Scheduler'ı başlat
  16. HTTP server'ı başlat
  17. SIGINT/SIGTERM bekle → graceful shutdown

  Graceful shutdown — Ctrl+C yapınca anında kapanmaz:
  1. Yeni HTTP isteklerini reddet
  2. Mevcut isteklerin bitmesini bekle
  3. Worker'ların mevcut işlerini bitirmesini bekle
  4. DB ve Redis bağlantılarını kapat

  Node.js'te:
  process.on('SIGTERM', async () => {
    await server.close()
    await pool.end()
  })

  ---
  ADIM 18: docker-compose.yml — Altyapı

  3 servis:
  - postgres — PostgreSQL 16 veritabanı
  - redis — Redis 7 (kuyruk + rate limit + cache)
  - notifier — Bizim uygulama

  depends_on + healthcheck ile postgres ve redis hazır olmadan uygulama başlamaz.

  ADIM 19: Dockerfile — Container İmajı

  Multi-stage build:
  # 1. Aşama: Go ile derle (büyük imaj, compiler var)
  FROM golang:1.22-alpine AS builder
  RUN go build -o /notifier ./cmd/notifier

  # 2. Aşama: Sadece binary'yi al (küçük imaj, ~20MB)
  FROM gcr.io/distroless/static-debian12
  COPY --from=builder /notifier /notifier

  Node.js'te node:alpine kullanırsın, ama Go'da derlenen binary tek başına çalışır — Node.js runtime'a bile gerek yok. Bu yüzden final imaj çok küçük.

  ADIM 20: CI Pipeline (.github/workflows/ci.yml)

  GitHub Actions ile:
  1. Lint — Kod kalitesi kontrolü (ESLint karşılığı)
  2. Test — Postgres ve Redis service container'ları ile testler
  3. Build — Binary derleme
  4. Docker — Docker imaj derleme

  ---
  ADIM 21: Testler

  4 test dosyası:

  - domain/notification_test.go — Enum'lar doğru çalışıyor mu?
  - validator/validator_test.go — Email, telefon, push token doğrulaması
  - delivery/retry_test.go — Backoff hesaplaması doğru mu?
  - service/template_test.go — Template render düzgün çalışıyor mu?

  Go'da test dosyaları aynı klasörde _test.go suffix'i ile durur. Jest'teki .test.js gibi.

  ---
  Veri Akışı (Büyük Resim)

  Kullanıcı POST /api/v1/notifications gönderir
      │
      ▼
  Handler: JSON parse + ilk doğrulama
      │
      ▼
  Service: Template render + kanal doğrulama + PostgreSQL'e kaydet (status: pending)
      │
      ▼
  Producer: Redis Stream'e XADD (notifications:high/normal/low)
      │                                          (status: queued)
      ▼
  Worker (goroutine): XREADGROUP ile mesajı al
      │
      ├─ Rate limit kontrolü (Redis Lua script)
      │
      ├─ PostgreSQL'den bildirimi getir         (status: processing)
      │
      ├─ Provider ile HTTP POST (webhook.site)
      │
      ├─ Başarılı? → status: delivered → WebSocket broadcast ✓
      │
      └─ Başarısız? → Retry kaldı mı?
           ├─ Evet → Redis sorted set'e ekle (backoff süresi ile)
           │         → Backoff goroutine tekrar kuyruğa atar
           └─ Hayır → status: failed → DLQ'ya gönder


  Scheduler (her 5 saniye):
      PostgreSQL'de scheduled_at <= NOW() olanları bul
      → Status: pending yap
      → Redis Stream'e XADD

  ---
  Çalıştırmak İçin

  # 1. Go'yu kur (https://go.dev/dl/)
  brew install go

  # 2. Bağımlılıkları indir (npm install karşılığı)
  go mod tidy

  # 3. Docker ile her şeyi ayağa kaldır
  docker compose up --build

  # 4. Test et
  curl localhost:8080/health        # → {"status":"ok"}
  curl localhost:8080/ready          # → postgres: ok, redis: ok

  # 5. Bildirim gönder
  curl -X POST localhost:8080/api/v1/notifications \
    -H "Content-Type: application/json" \
    -d '{"channel":"email","priority":"high","recipient":"test@example.com","subject":"Merhaba","body":"Test bildirimi"}'