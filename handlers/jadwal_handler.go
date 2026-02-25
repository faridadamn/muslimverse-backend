package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type JadwalResponse struct {
	Status string `json:"status"`
	Data   struct {
		Kota    string `json:"kota"`
		Tanggal string `json:"tanggal"`
		Jadwal  struct {
			Imsyak string `json:"imsyak"`
			Shubuh string `json:"shubuh"`
			Terbit string `json:"terbit"`
			Dhuha  string `json:"dhuha"`
			Dzuhur string `json:"dzuhur"`
			Ashr   string `json:"ashr"`
			Magrib string `json:"magrib"`
			Isya   string `json:"isya"`
		} `json:"jadwal"`
	} `json:"data"`
}

type KotaItem struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type DaftarKotaResponse struct {
	Status string     `json:"status"`
	Total  int        `json:"total"`
	Data   []KotaItem `json:"data"`
}

const (
	jadwalAPIBase = "https://script.google.com/macros/s/AKfycbx8CtuEFQrYxM5sF2pZYvjrcIQa4Mj25lO6BUVqFHrhURw05bg06dBtpeYtvax5NIi1/exec"
)

var daftarKotaCache []KotaItem

func GetJadwalShalat(c *gin.Context) {
	kotaInput := strings.ToLower(c.Param("kota"))
	if kotaInput == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Kota harus diisi"})
		return
	}

	log.Println("========== DEBUG JADWAL ==========")
	log.Println("1. Input kota dari user:", kotaInput)

	// Ambil daftar kota
	kotaList, err := getDaftarKota()
	if err != nil {
		log.Println("2. ERROR ambil daftar kota:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal ambil daftar kota"})
		return
	}
	log.Println("2. Total kota dalam cache:", len(kotaList))

	// Cari ID dan nama dari API pertama
	var localID string
	var kotaName string

	if isNumeric(kotaInput) {
		localID = kotaInput
		for _, kota := range kotaList {
			if kota.ID == localID {
				kotaName = kota.Name
				log.Println("   - Ditemukan! Local ID:", localID, "Nama:", kotaName)
				break
			}
		}
	} else {
		for _, kota := range kotaList {
			if strings.Contains(kota.Name, kotaInput) {
				localID = kota.ID
				kotaName = kota.Name
				log.Println("   - Ditemukan! Local ID:", localID, "Nama:", kotaName)
				break
			}
		}
	}

	if localID == "" {
		log.Println("4. ERROR: Kota tidak ditemukan di database")
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": "Kota tidak ditemukan",
		})
		return
	}

	// ===== MAPPING ID KHUSUS UNTUK myQuran =====
	var myQuranID string

	// Mapping manual untuk kota-kota besar
	switch kotaName {
	case "jakartabarat", "jakartapusat", "jakartaselatan", "jakartatimur", "jakartautara":
		myQuranID = "1301" // Semua Jakarta pake ID yang sama
	case "bandung":
		myQuranID = "0314" // ID Bandung di myQuran
	case "surabaya":
		myQuranID = "0505" // ID Surabaya di myQuran
	case "medan":
		myQuranID = "1207" // ID Medan di myQuran
	case "semarang":
		myQuranID = "1406" // ID Semarang di myQuran
	case "palembang":
		myQuranID = "1674" // ID Palembang di myQuran
	case "makassar":
		myQuranID = "7601" // ID Makassar di myQuran
	case "tangerang":
		myQuranID = "3671" // ID Tangerang di myQuran
	case "bekasi":
		myQuranID = "3216" // ID Bekasi di myQuran
	case "depok":
		myQuranID = "3276" // ID Depok di myQuran
	default:
		// Fallback: pake ID yang sama (tapi kemungkinan besar error)
		myQuranID = localID
		log.Println("⚠️ Peringatan: Tidak ada mapping untuk", kotaName, ", pakai ID:", localID)
	}

	log.Println("4a. Mapping - Kota:", kotaName, "| Local ID:", localID, "| myQuran ID:", myQuranID)

	// PAKAI API myQuran dengan ID yang sudah dimapping
	tahun := time.Now().Format("2006")
	bulan := time.Now().Format("01")
	tanggal := time.Now().Format("02")

	apiURL := fmt.Sprintf("https://api.myquran.com/v1/sholat/jadwal/%s/%s/%s/%s",
		myQuranID, tahun, bulan, tanggal)

	log.Println("5. Memanggil myQuran API:", apiURL)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(apiURL)
	if err != nil {
		log.Println("6. ERROR network:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Network error: " + err.Error()})
		return
	}
	defer resp.Body.Close()

	log.Println("6. Response status code:", resp.StatusCode)

	// Baca response body untuk log
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("7. ERROR baca body:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal baca response"})
		return
	}

	log.Println("7. Raw response dari myQuran:", string(bodyBytes))

	// Parse response myQuran
	var myQuranResp struct {
		Status bool `json:"status"`
		Data   struct {
			Lokasi string `json:"lokasi"`
			Daerah string `json:"daerah"`
			Jadwal struct {
				Imsak   string `json:"imsak"`
				Subuh   string `json:"subuh"`
				Terbit  string `json:"terbit"`
				Dhuha   string `json:"dhuha"`
				Dzuhur  string `json:"dzuhur"`
				Ashar   string `json:"ashar"`
				Maghrib string `json:"maghrib"`
				Isya    string `json:"isya"`
				Tanggal string `json:"tanggal"`
			} `json:"jadwal"`
		} `json:"data"`
		Message string `json:"message"`
	}

	err = json.Unmarshal(bodyBytes, &myQuranResp)
	if err != nil {
		log.Println("8. ERROR parsing JSON:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Parse error: " + err.Error()})
		return
	}

	log.Println("8. Parsed response - Status:", myQuranResp.Status)

	if !myQuranResp.Status {
		log.Println("9. ERROR: myQuran return status false")
		log.Println("9. Message dari myQuran:", myQuranResp.Message)
		c.JSON(http.StatusOK, gin.H{
			"status":  "error",
			"message": fmt.Sprintf("Jadwal untuk %s belum tersedia", kotaName),
		})
		return
	}

	log.Println("9. Data jadwal - Subuh:", myQuranResp.Data.Jadwal.Subuh)
	log.Println("10. ✅ SEMUA BERHASIL! Mengirim response ke Flutter")

	// Response untuk Flutter - PASTIKAN KOTA TERKIRIM!
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"kota":    kotaName, // Nama kota asli dari input
			"tanggal": myQuranResp.Data.Jadwal.Tanggal,
			"jadwal": gin.H{
				"imsyak": myQuranResp.Data.Jadwal.Imsak,
				"shubuh": myQuranResp.Data.Jadwal.Subuh,
				"terbit": myQuranResp.Data.Jadwal.Terbit,
				"dhuha":  myQuranResp.Data.Jadwal.Dhuha,
				"dzuhur": myQuranResp.Data.Jadwal.Dzuhur,
				"ashr":   myQuranResp.Data.Jadwal.Ashar,
				"magrib": myQuranResp.Data.Jadwal.Maghrib,
				"isya":   myQuranResp.Data.Jadwal.Isya,
			},
		},
	})
}

func GetDaftarKota(c *gin.Context) {
	kotaList, err := getDaftarKota()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Filter pencarian kalau ada query
	search := c.Query("search")
	if search != "" {
		filtered := []KotaItem{}

		for _, kota := range kotaList {
			if strings.Contains(strings.ToLower(kota.Name), strings.ToLower(search)) {
				filtered = append(filtered, kota)
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"status": "success",
			"total":  len(filtered),
			"data":   filtered,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"total":  len(kotaList),
		"data":   kotaList,
	})
}

func getDaftarKota() ([]KotaItem, error) {
	if daftarKotaCache != nil {
		return daftarKotaCache, nil
	}

	apiURL := jadwalAPIBase + "?action=daftar-kota"
	resp, err := http.Get(apiURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var kotaResp DaftarKotaResponse
	err = json.Unmarshal(body, &kotaResp)
	if err != nil {
		return nil, err
	}

	daftarKotaCache = kotaResp.Data
	return daftarKotaCache, nil
}

func isNumeric(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func getCurrentDate() string {
	return time.Now().Format("02 January 2006")
}
