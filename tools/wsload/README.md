# WebSocket Yük Testi — 10.000 Eşzamanlı Kullanıcı Simülasyonu

`wsload`, canlı fiyat akışına (`/ws`) binlerce mobil istemciyi taklit ederek
bağlanır, bağlantıları açık tutar ve gecikme/throughput ölçer.

## Neyi ölçer?

- **Bağlantı başarısı / hatası / reconnect** — accept kuyruğu, dosya
  tanıtıcısı (fd) limitleri veya backend doygunluğu sorunlarını yakalar.
- **İlk mesaj süresi (time-to-first-message)** — bağlantı kurulduktan sonra
  ilk fiyat karesinin gelme süresi.
- **Uçtan uca teslim gecikmesi** — `now - envelope.timestamp`. Sunucunun
  broadcast fan-out gecikmesini ölçer. Doğru olması için istemci ve sunucu
  saatleri NTP ile senkron olmalı; en temiz sonuç için **sunucuda** çalıştırın.
- **Mesaj / bayt throughput**.

## Çalıştırma

```bash
# Yerelde backend'e karşı
go run ./tools/wsload -url ws://127.0.0.1:8080/ws -conns 2000 -rate 500 -duration 30s

# Sunucuda NodePort'a karşı (SADECE backend kapasitesi — Cloudflare/nginx hariç)
./wsload -url ws://127.0.0.1:4700/ws -conns 10000 -rate 1000 -duration 120s

# Public uçtan uca (Cloudflare + aaPanel nginx + NodePort dahil)
./wsload -url wss://tsc-api.talhacan.com/ws -conns 10000 -rate 500 -duration 120s
```

Bayraklar: `-url`, `-conns`, `-rate` (saniyede açılan bağlantı — rampa),
`-duration` (rampadan sonra tutma süresi), `-token` (opsiyonel JWT).

## Metodoloji (10k plan)

1. **İzole backend testi:** Yükü sunucuda, NodePort'a (`127.0.0.1:4700`) karşı
   çalıştır. Böylece yalnızca Go WebSocket hub'ının fan-out kapasitesini
   ölçersin (ağ/CDN değişkenleri devre dışı).
2. **Rampa:** 1000 bağ/sn ile 10.000'e çık (10 sn), sonra 120 sn tut.
3. **Public uçtan uca test:** Aynı testi `wss://tsc-api.talhacan.com/ws`
   üzerinden çalıştır; Cloudflare + aaPanel nginx + NodePort yolunu doğrula.
4. **Kabul kriterleri:** dial hatası ≈ 0, read hatası ≈ 0, teslim gecikmesi
   p99 < 100ms (backend), broadcast düşmesi (dropped) = 0.

## Ölçekleme için gereken ayarlar (10k+)

- **Yük üreteci host'u:** `ulimit -n 200000` (her bağlantı 1 fd).
- **Backend pod'ları:** HPA zaten 3→20 replica (CPU %70). Her replica binlerce
  bağlantıyı taşır; hub broadcast'i O(bağlantı) olduğundan replica sayısı
  fan-out'u paralelleştirir. Redis pub/sub ile tüm replica'lar aynı fiyatı
  yayınlar (multi-cluster).
- **aaPanel nginx:** `worker_connections` yüksek olmalı (ör. 65535),
  `worker_rlimit_nofile` artırılmalı; WS için `proxy_read_timeout 3600s`.
- **Cloudflare:** WebSocket açık olmalı; free planda eşzamanlı WS sınırlarına
  dikkat — gerçek 10k için Cloudflare'ı bypass edip (gri bulut) doğrudan
  NodePort'a proxy yapılan bir alt alan adı tercih edilebilir.
- **k3s node:** `net.core.somaxconn`, `net.ipv4.ip_local_port_range` ve dosya
  tanıtıcısı limitleri (`fs.file-max`) artırılmalı.

## Gerçek sonuçlar (13 Tem 2026, sunucu: 40 çekirdek / 96GB)

NodePort'a (`127.0.0.1:4700`) karşı, yük üreteci aynı node'da:

| Senaryo | Pod | Rampa | Zirve bağlantı | dial hata | read hata | Teslim gecikmesi |
|---|---|---|---|---|---|---|
| İlk deneme | 3 (HPA min) | 1000/sn | 8975 → 5896 | 329 | 3775 | p50 <2s, p99 >5s |
| Ölçekli deneme | 12 (ön-ölçek) | 500/sn | **10000/10000** | **0** | **0** | p50 <200ms, p99 <2s |

Çıkarımlar:
- Backend, yeterli replica ile **10.000 eşzamanlı bağlantıyı 0 hata ile** taşır;
  ilk mesaj p50 <1ms.
- İlk denemedeki düşüşün nedeni **bağlantı fırtınası**: rampa sırasında sadece 3
  pod vardı ve HPA (CPU tabanlı) ~15-30 sn sonra tepki verir. Bu süre zarfında
  pod'lar doydu. Çözüm: HPA `minReplicas: 5` + hızlı `scaleUp` davranışı
  (uygulandı) ve kampanya öncesi manuel ön-ölçekleme.
- Gecikme rakamları, yük üretecinin **aynı node'da** CPU çalmasından etkilenir;
  gerçek üretimde (istemciler harici) daha düşüktür.

## Yorumlama

- `dialErr` yüksekse → accept kuyruğu / fd limiti / nginx worker_connections.
- `readErr` yüksek + `live` düşüyorsa → backend bağlantıları düşürüyor
  (write buffer dolması, ping/pong timeout). Hub `send_buffer_size` artırılır.
- Teslim gecikmesi p99 büyükse → broadcast fan-out CPU-bound; replica ekle
  (HPA maxReplicas) veya broadcast'i shard'la.
