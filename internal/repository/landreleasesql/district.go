// Package landreleasesql — «Буулгалтын мэдээлэл»-ийн repo болон seeder хоёрын
// ДУНДЫН SQL (seed → postgres импорт нь тестийн import cycle үүсгэдэг тул
// энд тусад нь).
package landreleasesql

// AssignDistrictsSQL — газрын байршлаар сум/дүүрэг (au2) ба аймаг/хотыг (au1)
// тодорхойлж land_release-д хадгална. Газар бүр НЭГ сум/дүүрэгт: доторх цэг
// (ST_PointOnSurface) байрлах сум/дүүрэг, эс бөгөөс давхцсан аль нэг нь.
// ST_Intersection-оор давхцлын талбай харьцуулахаас ~4 дахин хурдан.
//
// $1 — тооцох газрын id-ууд (uuid[]); NULL бол сум/дүүрэггүй БҮХ газар
// (хуучин мөрийг нөхөх seeder).
const AssignDistrictsSQL = `
	UPDATE land_release lr
	   SET au2_code = a.code, au1_code = a.au1_code
	  FROM land_release src
	  JOIN LATERAL (
		SELECT a2.code, a2.au1_code
		  FROM au2 a2
		 WHERE a2.geometry IS NOT NULL AND ST_Intersects(a2.geometry, src.geometry)
		 ORDER BY ST_Intersects(a2.geometry, ST_PointOnSurface(src.geometry)) DESC, a2.code
		 LIMIT 1
	  ) a ON TRUE
	 WHERE lr.id = src.id
	   AND (($1::uuid[] IS NULL AND lr.au2_code IS NULL) OR lr.id = ANY($1::uuid[]))`
