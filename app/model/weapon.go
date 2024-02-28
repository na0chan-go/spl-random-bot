package model

// Weapon 武器情報
type Weapon struct {
	Key        string        `json:"key"`
	Aliases    []string      `json:"aliases"`
	WeaponType WeaponType    `json:"type"`
	WeaponName WeaponName    `json:"name"`
	Main       string        `json:"main"`
	Sub        SubWeapon     `json:"sub"`
	Special    SpecialWeapon `json:"special"`
	Reskin_of  string        `json:"reskin_of"`
}

// WeaponType 武器の種類
type WeaponType struct {
	Key     string     `json:"key"`
	Aliases []string   `json:"aliases"`
	Name    WeaponName `json:"name"`
}

// WeaponName 武器の名前
type WeaponName struct {
	USName string `json:"en_US"`
	JPName string `json:"ja_JP"`
}

// SubWeapon サブウェポン
type SubWeapon struct {
	Key        string     `json:"key"`
	Aliases    []string   `json:"aliases"`
	WeaponName WeaponName `json:"name"`
}

// SpecialWeapon スペシャルウェポン
type SpecialWeapon struct {
	Key        string     `json:"key"`
	Aliases    []string   `json:"aliases"`
	WeaponName WeaponName `json:"name"`
}
