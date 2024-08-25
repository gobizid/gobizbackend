package helpdesk

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/gocroot/config"
	"github.com/gocroot/helper/atapi"
	"github.com/gocroot/helper/atdb"
	"github.com/gocroot/helper/hub"
	"github.com/gocroot/helper/lms"
	"github.com/gocroot/helper/menu"
	"github.com/gocroot/helper/phone"
	"github.com/gocroot/helper/tiket"
	"github.com/gocroot/model"
	"github.com/whatsauth/itmodel"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// helpdesk sudah terintegrasi dengan lms pamong desa backend
func HelpdeskPDLMS(Profile itmodel.Profile, Pesan itmodel.IteungMessage, db *mongo.Database) (reply string) {
	//check apakah tiketnya udah tutup atau belum
	isclosed, stiket, err := tiket.IsTicketClosed("userphone", Pesan.Phone_number, db)
	if err != nil {
		return "IsTicketClosed: " + err.Error()
	}
	if !isclosed { //ada yang belum closed, lanjutkan sesi hub
		//pesan ke user
		reply = GetPrefillMessage("userbantuanadmin", db) //pesan ke user
		reply = fmt.Sprintf(reply, stiket.AdminName)
		hub.CheckHubSession(Pesan.Phone_number, stiket.UserName, stiket.AdminPhone, stiket.AdminName, db)
		//inject menu session untuk menutup tiket
		mn := menu.MenuList{
			No:      0,
			Keyword: stiket.ID.Hex() + "|tutuph3lpdeskt1kcet",
			Konten:  "Akhiri percakapan dan tutup sesi bantuan saat ini",
		}
		err = menu.InjectSessionMenu([]menu.MenuList{mn}, Pesan.Phone_number, db)
		if err != nil {
			return err.Error()
		}
		return
	}
	//jika tiket sudah clear
	statuscode, res, err := atapi.GetStructWithToken[lms.ResponseAPIPD]("token", config.APITOKENPD, config.APIGETPDLMS+Pesan.Phone_number)
	if statuscode != 200 { //404 jika user not found
		msg := "Mohon maaf Bapak/Ibu, nomor anda *belum terdaftar* pada sistem kami.\n" + UserNotFound(Profile, Pesan, db)
		return msg
	}
	if err != nil {
		return err.Error()
	}
	if len(res.Data.ContactAdminProvince) == 0 { //kalo kosong data kontak admin provinsinya maka arahkan ke tim 16 tapi sesuikan dengan provinsinya
		msg := "Mohon maaf Bapak/Ibu " + res.Data.Fullname + " dari desa " + res.Data.Village + ", helpdesk pamongdesa anda.\n" + AdminNotFoundWithProvinsi(Profile, Pesan, res.Data.Province, db)
		return msg
	}
	//jika arraynya ada adminnya maka lanjut ke start session hub
	helpdeskno := res.Data.ContactAdminProvince[0].Phone
	helpdeskname := res.Data.ContactAdminProvince[0].Fullname
	if helpdeskname == "" || helpdeskno == "" {
		return "Nama atau nomor helpdesk tidak ditemukan"
	}
	//pesan ke admin
	msgstr := GetPrefillMessage("adminbantuanadmin", db) //pesan ke admin
	msgstr = fmt.Sprintf(msgstr, res.Data.Fullname, res.Data.Village, res.Data.District, res.Data.Regency)
	dt := &itmodel.TextMessage{
		To:       helpdeskno,
		IsGroup:  false,
		Messages: msgstr,
	}
	go atapi.PostStructWithToken[itmodel.Response]("Token", Profile.Token, dt, Profile.URLAPIText)
	//pesan ke user
	reply = GetPrefillMessage("userbantuanadmin", db) //pesan ke user
	reply = fmt.Sprintf(reply, helpdeskname)
	//insert ke database dan set hub session
	idtiket, err := tiket.InserNewTicket(Pesan.Phone_number, helpdeskname, helpdeskno, db)
	if err != nil {
		return err.Error()
	}
	hub.CheckHubSession(Pesan.Phone_number, res.Data.Fullname, helpdeskno, helpdeskname, db)
	//inject menu session untuk menutup tiket
	mn := menu.MenuList{
		No:      0,
		Keyword: idtiket.Hex() + "|tutuph3lpdeskt1kcet",
		Konten:  "Akhiri percakapan dan tutup sesi bantuan saat ini",
	}
	err = menu.InjectSessionMenu([]menu.MenuList{mn}, Pesan.Phone_number, db)
	if err != nil {
		return err.Error()
	}
	return

}

// Jika user tidak terdaftar maka akan mengeluarkan list operator pusat
func UserNotFound(Profile itmodel.Profile, Pesan itmodel.IteungMessage, db *mongo.Database) (reply string) {
	//check apakah ada session, klo ga ada kasih reply menu
	Sesdoc, _, err := menu.CheckSession(Pesan.Phone_number, db)
	if err != nil {
		return err.Error()
	}

	msg, err := menu.GetMenuFromKeywordAndSetSession("adminpusat", Sesdoc, db)
	if err != nil {
		return err.Error()
	}
	return msg
}

// penugasan helpdeskpusat jika user belum terdaftar, ini limpahan dari pilihan func UserNotFound
func HelpdeskPusat(Profile itmodel.Profile, Pesan itmodel.IteungMessage, db *mongo.Database) (reply string) {
	Pesan.Message = strings.ReplaceAll(Pesan.Message, "adminpusat", "")
	Pesan.Message = strings.TrimSpace(Pesan.Message)
	op, err := GetOperatorFromSection(Pesan.Message, db)
	if err != nil {
		return err.Error()
	}
	res := lms.GetDataFromAPI(Pesan.Phone_number)
	msgstr := GetPrefillMessage("adminbantuanadmin", db) //pesan untuk admin
	msgstr = fmt.Sprintf(msgstr, res.Data.Fullname, res.Data.Village, res.Data.District, res.Data.Regency)
	dt := &itmodel.TextMessage{
		To:       op.PhoneNumber,
		IsGroup:  false,
		Messages: msgstr,
	}
	go atapi.PostStructWithToken[itmodel.Response]("Token", Profile.Token, dt, Profile.URLAPIText)
	reply = GetPrefillMessage("userbantuanadmin", db) //pesan untuk user
	reply = fmt.Sprintf(reply, op.Name)
	//insert ke database dan set hub session
	idtiket, err := tiket.InserNewTicket(Pesan.Phone_number, op.Name, op.PhoneNumber, db)
	if err != nil {
		return err.Error()
	}
	hub.CheckHubSession(Pesan.Phone_number, phone.MaskPhoneNumber(Pesan.Phone_number)+" ~ "+Pesan.Alias_name, op.PhoneNumber, op.Name, db)
	//inject menu session untuk menutup tiket
	mn := menu.MenuList{
		No:      0,
		Keyword: idtiket.Hex() + "|tutuph3lpdeskt1kcet",
		Konten:  "Akhiri percakapan dan tutup sesi bantuan saat ini",
	}
	err = menu.InjectSessionMenu([]menu.MenuList{mn}, Pesan.Phone_number, db)
	if err != nil {
		return err.Error()
	}
	return

}

// Jika user terdaftar tapi belum ada operator provinsi maka akan mengeluarkan list operator pusat
func AdminNotFoundWithProvinsi(Profile itmodel.Profile, Pesan itmodel.IteungMessage, provinsi string, db *mongo.Database) (reply string) {
	//tambah lojik query ke provinsi
	sec, err := GetSectionFromProvinsiRegex(db, provinsi)
	if err != nil {
		return err.Error()
	}
	op, err := GetOperatorFromSection(sec, db)
	if err != nil {
		return err.Error()
	}
	res := lms.GetDataFromAPI(Pesan.Phone_number)
	msgstr := GetPrefillMessage("adminbantuanadmin", db) //pesan untuk admin
	msgstr = fmt.Sprintf(msgstr, res.Data.Fullname, res.Data.Village, res.Data.District, res.Data.Regency)
	dt := &itmodel.TextMessage{
		To:       op.PhoneNumber,
		IsGroup:  false,
		Messages: msgstr,
	}
	go atapi.PostStructWithToken[itmodel.Response]("Token", Profile.Token, dt, Profile.URLAPIText)
	reply = GetPrefillMessage("userbantuanadmin", db) //pesan untuk user
	reply = fmt.Sprintf(reply, op.Name)
	//insert ke database dan set hub session
	idtiket, err := tiket.InserNewTicket(Pesan.Phone_number, op.Name, op.PhoneNumber, db)
	if err != nil {
		return err.Error()
	}
	hub.CheckHubSession(Pesan.Phone_number, res.Data.Fullname, op.PhoneNumber, op.Name, db)
	//inject menu session untuk menutup tiket
	mn := menu.MenuList{
		No:      0,
		Keyword: idtiket.Hex() + "|tutuph3lpdeskt1kcet",
		Konten:  "Akhiri percakapan dan tutup sesi bantuan saat ini",
	}
	err = menu.InjectSessionMenu([]menu.MenuList{mn}, Pesan.Phone_number, db)
	if err != nil {
		return err.Error()
	}
	return
}

// penutupan helpdesk dari pilihan menu objectid|tutuph3lpdeskt1kcet
func EndHelpdesk(Profile itmodel.Profile, Pesan itmodel.IteungMessage, db *mongo.Database) (reply string) {
	msgs := strings.Split(Pesan.Message, "|")
	id := msgs[0]
	// Mengonversi id string ke primitive.ObjectID
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		reply = "Invalid ID format: " + err.Error()
		return
	}
	helpdeskuser, err := atdb.GetOneLatestDoc[tiket.Bantuan](db, "tiket", bson.M{"_id": objectID})
	if err != nil {
		reply = err.Error()
		return
	}
	//helpdeskuser.Solusi = strings.Split(msgs[1], ":")[1]
	helpdeskuser.Terlayani = true
	_, err = atdb.ReplaceOneDoc(db, "tiket", bson.M{"_id": objectID}, helpdeskuser)
	if err != nil {
		reply = err.Error()
		return
	}
	//hapus hub
	atdb.DeleteOneDoc(db, "hub", bson.M{"userphone": helpdeskuser.UserPhone, "adminphone": helpdeskuser.AdminPhone})
	//prefill message admin dan user
	msgstradmin := GetPrefillMessage("admintutuphelpdesk", db) //pesan untuk admin
	msgstradmin = fmt.Sprintf(msgstradmin, helpdeskuser.UserName, helpdeskuser.Desa)

	msgstruser := GetPrefillMessage("usertutuphelpdesk", db) //pesan untuk user
	msgstruser = fmt.Sprintf(msgstruser, helpdeskuser.AdminName, helpdeskuser.UserName, helpdeskuser.ID.Hex())
	//pembagian yg dikirim dan reply
	var sendmsg string
	if Pesan.Phone_number == helpdeskuser.UserPhone {
		reply = msgstruser
		sendmsg = msgstradmin
	} else {
		reply = msgstradmin
		sendmsg = msgstruser
	}
	dt := &itmodel.TextMessage{
		To:       helpdeskuser.UserPhone,
		IsGroup:  false,
		Messages: sendmsg,
	}
	go atapi.PostStructWithToken[itmodel.Response]("Token", Profile.Token, dt, Profile.URLAPIText)

	return
}

// admin terkoneksi dengan user tiket terakhir yang belum terlayani
func AdminOpenSessionCurrentUserTiket(Profile itmodel.Profile, Pesan itmodel.IteungMessage, db *mongo.Database) (reply string) {
	//check apakah tiketnya udah tutup atau belum
	isclosed, stiket, err := tiket.IsTicketClosed("adminphone", Pesan.Phone_number, db)
	if err != nil {
		return "IsTicketClosed: " + err.Error()
	}
	if !isclosed { //ada yang belum closed, lanjutkan sesi hub
		//pesan ke admin
		reply = GetPrefillMessage("adminadasesitiket", db) //pesan ke user
		reply = fmt.Sprintf(reply, stiket.UserName, stiket.Desa, stiket.Kec, stiket.KabKot)
		hub.CheckHubSession(stiket.UserPhone, stiket.UserName, stiket.AdminPhone, stiket.AdminName, db)
		//inject menu session untuk menutup tiket
		mn := menu.MenuList{
			No:      0,
			Keyword: stiket.ID.Hex() + "|tutuph3lpdeskt1kcet",
			Konten:  "Akhiri percakapan dan tutup sesi bantuan saat ini",
		}
		err = menu.InjectSessionMenu([]menu.MenuList{mn}, Pesan.Phone_number, db)
		if err != nil {
			return err.Error()
		}
		return
	}

	reply = GetPrefillMessage("adminkosongsesitiket", db) //pesan ke user
	reply = fmt.Sprintf(reply, stiket.AdminName)
	return
}

// legacy
// handling key word, keyword :bantuan operator
func StartHelpdesk(Profile itmodel.Profile, Pesan itmodel.IteungMessage, db *mongo.Database) (reply string) {
	//check apakah tiket dari user sudah di tutup atau belum
	user, err := atdb.GetOneLatestDoc[model.Laporan](db, "helpdeskuser", bson.M{"terlayani": bson.M{"$exists": false}, "phone": Pesan.Phone_number})
	if err != nil {
		if err != mongo.ErrNoDocuments {
			return err.Error()
		}
		//berarti tiket udah close semua
	} else { //ada tiket yang belum close
		msgstr := "*Permintaan bantuan dari Pengguna " + user.Nama + " (" + user.Phone + ")*\n\nMohon dapat segera menghubungi beliau melalui WhatsApp di nomor wa.me/" + user.Phone + " untuk memberikan solusi terkait masalah yang sedang dialami:\n\n" + user.Masalah
		msgstr += "\n\nSetelah masalah teratasi, dimohon untuk menginputkan solusi yang telah diberikan ke dalam sistem melalui tautan berikut:\nwa.me/" + Profile.Phonenumber + "?text=" + user.ID.Hex() + "|+solusi+dari+operator+helpdesk+:+"
		dt := &itmodel.TextMessage{
			To:       user.User.PhoneNumber,
			IsGroup:  false,
			Messages: msgstr,
		}
		go atapi.PostStructWithToken[itmodel.Response]("Token", Profile.Token, dt, Profile.URLAPIText)
		reply = "Segera, Bapak/Ibu akan dihubungkan dengan salah satu Admin kami, *" + user.User.Name + "*.\n\n Mohon tunggu sebentar, kami akan menghubungi Anda melalui WhatsApp di nomor wa.me/" + user.User.PhoneNumber + "\nTerima kasih atas kesabaran Bapak/Ibu"
		//reply = "Kakak kami hubungkan dengan operator kami yang bernama *" + user.User.Name + "* di nomor wa.me/" + user.User.PhoneNumber + "\nMohon tunggu sebentar kami akan kontak kakak melalui nomor tersebut.\n_Terima kasih_"
		return
	}
	//mendapatkan semua nama team dari db
	namateam, helpdeskslist, err := GetNamaTeamFromPesan(Pesan, db)
	if err != nil {
		return err.Error()
	}

	//suruh pilih nama team kalo tidak ada
	if namateam == "" {
		reply = "Selamat datang Bapak/Ibu " + Pesan.Alias_name + "\n\nTerima kasih telah menghubungi kami *Helpdesk LMS Pamong Desa*\n\n"
		reply += "Untuk mendapatkan layanan yang lebih baik, mohon bantuan Bapak/Ibu *untuk memilih regional* tujuan Anda terlebih dahulu:\n"
		for i, helpdesk := range helpdeskslist {
			no := strconv.Itoa(i + 1)
			teamurl := strings.ReplaceAll(helpdesk, " ", "+")
			reply += no + ". Regional " + helpdesk + "\n" + "wa.me/" + Profile.Phonenumber + "?text=bantuan+operator+" + teamurl + "\n"
		}
		return
	}
	//suruh pilih scope dari bantuan team
	scope, scopelist, err := GetScopeFromTeam(Pesan, namateam, db)
	if err != nil {
		return err.Error()
	}
	//pilih scope jika belum
	if scope == "" {
		reply = "Terima kasih.\nSekarang, mohon pilih provinsi asal Bapak/Ibu dari daftar berikut:\n" // " + namateam + " :\n"
		for i, scope := range scopelist {
			no := strconv.Itoa(i + 1)
			scurl := strings.ReplaceAll(scope, " ", "+")
			reply += no + ". " + scope + "\n" + "wa.me/" + Profile.Phonenumber + "?text=bantuan+operator+" + namateam + "+" + scurl + "\n"
		}
		return
	}
	//menuliskan pertanyaan bantuan
	user = model.Laporan{
		Scope: scope,
		Team:  namateam,
		Nama:  Pesan.Alias_name,
		Phone: Pesan.Phone_number,
	}
	_, err = atdb.InsertOneDoc(db, "helpdeskuser", user)
	if err != nil {
		return err.Error()
	}
	reply = "Silakan ketik pertanyaan atau masalah yang ingin Bapak/Ibu " + Pesan.Alias_name + " sampaikan. Kami siap membantu Anda" // + " mengetik pertanyaan atau bantuan yang ingin dijawab oleh operator: "

	return
}

// handling key word
func FeedbackHelpdesk(Profile itmodel.Profile, Pesan itmodel.IteungMessage, db *mongo.Database) (reply string) {
	msgs := strings.Split(Pesan.Message, "|")
	id := msgs[0]
	// Mengonversi id string ke primitive.ObjectID
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		reply = "Invalid ID format: " + err.Error()
		return
	}
	helpdeskuser, err := atdb.GetOneLatestDoc[model.Laporan](db, "helpdeskuser", bson.M{"_id": objectID, "phone": Pesan.Phone_number})
	if err != nil {
		reply = err.Error()
		return
	}
	strrate := strings.Split(msgs[1], ":")[1]
	rate := strings.TrimSpace(strrate)
	rt, err := strconv.Atoi(rate)
	if err != nil {
		reply = err.Error()
		return
	}
	helpdeskuser.RateLayanan = rt
	_, err = atdb.ReplaceOneDoc(db, "helpdeskuser", bson.M{"_id": objectID}, helpdeskuser)
	if err != nil {
		reply = err.Error()
		return
	}

	reply = "Terima kasih banyak atas waktu Bapak/Ibu untuk memberikan penilaian terhadap pelayanan Admin " + helpdeskuser.User.Name + "\n\nApresiasi Bapak/Ibu sangat berarti bagi kami untuk terus memberikan yang terbaik.."

	msgstr := "*Feedback Diterima*\n*" + helpdeskuser.Nama + "*\n*" + helpdeskuser.Phone + "*\nMemberikan rating " + rate + " bintang"
	dt := &itmodel.TextMessage{
		To:       helpdeskuser.User.PhoneNumber,
		IsGroup:  false,
		Messages: msgstr,
	}
	go atapi.PostStructWithToken[itmodel.Response]("Token", Profile.Token, dt, Profile.URLAPIText)

	return
}

// handling non key word
func PenugasanOperator(Profile itmodel.Profile, Pesan itmodel.IteungMessage, db *mongo.Database) (reply string, err error) {
	//check apakah tiket dari user sudah di tutup atau belum
	user, err := atdb.GetOneLatestDoc[model.Laporan](db, "helpdeskuser", bson.M{"phone": Pesan.Phone_number})
	if err != nil {
		if err == mongo.ErrNoDocuments {
			//check apakah dia operator yang belum tutup tiketnya
			user, err = atdb.GetOneLatestDoc[model.Laporan](db, "helpdeskuser", bson.M{"terlayani": bson.M{"$exists": false}, "user.phonenumber": Pesan.Phone_number})
			if err != nil {
				if err == mongo.ErrNoDocuments {
					err = nil
					reply = ""
					return
				}
				err = errors.New("galat di collection helpdeskuser operator: " + err.Error())
				return
			}
			//jika ada tiket yang statusnya belum closed
			reply = "*Permintaan bantuan dari Pengguna " + user.Nama + " (" + user.Phone + ")*\n\nMohon dapat segera menghubungi beliau melalui WhatsApp di nomor wa.me/" + user.Phone + " untuk memberikan solusi terkait masalah yang sedang dialami:\n\n" + user.Masalah
			reply += "\n\nSetelah masalah teratasi, dimohon untuk menginputkan solusi yang telah diberikan ke dalam sistem melalui tautan berikut:\nwa.me/" + Profile.Phonenumber + "?text=" + user.ID.Hex() + "|+solusi+dari+operator+helpdesk+:+"
			return

		}
		err = errors.New("galat di collection helpdeskuser user: " + err.Error())
		return
	}
	if !user.Terlayani {
		user.Masalah += "\n" + Pesan.Message
		if user.User.Name == "" || user.User.PhoneNumber == "" {
			var op model.Userdomyikado
			op, err = GetOperatorFromScopeandTeam(user.Scope, user.Team, db)
			if err != nil {
				return
			}
			user.User = op
		}
		_, err = atdb.ReplaceOneDoc(db, "helpdeskuser", bson.M{"_id": user.ID}, user)
		if err != nil {
			return
		}

		msgstr := "*Permintaan bantuan dari Pengguna " + user.Nama + " (" + user.Phone + ")*\n\nMohon dapat segera menghubungi beliau melalui WhatsApp di nomor wa.me/" + user.Phone + " untuk memberikan solusi terkait masalah yang sedang dialami:\n\n" + user.Masalah
		msgstr += "\n\nSetelah masalah teratasi, dimohon untuk menginputkan solusi yang telah diberikan ke dalam sistem melalui tautan berikut:\nwa.me/" + Profile.Phonenumber + "?text=" + user.ID.Hex() + "|+solusi+dari+operator+helpdesk+:+"
		dt := &itmodel.TextMessage{
			To:       user.User.PhoneNumber,
			IsGroup:  false,
			Messages: msgstr,
		}
		go atapi.PostStructWithToken[itmodel.Response]("Token", Profile.Token, dt, Profile.URLAPIText)

		reply = "Segera, Bapak/Ibu akan dihubungkan dengan salah satu Admin kami, *" + user.User.Name + "*.\n\n Mohon tunggu sebentar, kami akan menghubungi Anda melalui WhatsApp di nomor wa.me/" + user.User.PhoneNumber + "\nTerima kasih atas kesabaran Bapak/Ibu"

	}
	return

}
