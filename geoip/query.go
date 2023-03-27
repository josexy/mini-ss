package geoip

import (
	"net/netip"

	"github.com/oschwald/geoip2-golang"
)

var db *geoip2.Reader

func OpenDB(path string) (err error) {
	db, err = geoip2.Open(path)
	if err != nil {
		return err
	}
	return nil
}

func CloseDB() error {
	if db == nil {
		return nil
	}
	return db.Close()
}

func QueryCountryByIP(ip netip.Addr) string {
	if !ip.IsValid() {
		return ""
	}
	country, err := db.Country(ip.AsSlice())
	if err != nil {
		return ""
	}
	return country.Country.IsoCode
}

func QueryCountryByString(s string) string {
	ip, err := netip.ParseAddr(s)
	if err != nil {
		return ""
	}
	return QueryCountryByIP(ip)
}
