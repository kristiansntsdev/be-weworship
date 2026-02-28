package models

type Pengurus struct {
	ID         int    `db:"id_pengurus"`
	Nama       string `db:"nama"`
	Username   string `db:"username"`
	Password   string `db:"password"`
	LevelAdmin string `db:"leveladmin"`
	Nowa       string `db:"nowa"`
	Kota       string `db:"kotalevelup"`
}

type Peserta struct {
	ID         int    `db:"id_peserta"`
	Nama       string `db:"nama"`
	Email      string `db:"email"`
	Password   string `db:"password"`
	UserCode   string `db:"usercode"`
	UserLevel  string `db:"userlevel"`
	Verifikasi string `db:"verifikasi"`
	Status     string `db:"status"`
	Role       string `db:"role"`
}

type PesertaBasic struct {
	ID    int    `db:"id_peserta"`
	Nama  string `db:"nama"`
	Email string `db:"email"`
}
