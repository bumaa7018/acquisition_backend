-- Чөлөөлөлтийн төлөв (land_acquisition.status) нь СТАТИК: 1–4 нь кодонд
-- (domain.LandAcquisitionStatus*) тодорхойлогдож, бизнес дүрэм (хээрийн судалгаа,
-- баталгаажуулалт, түгжих) тэдгээрт тулгуурладаг. acquisition_progress_status
-- хүснэгт нь ямар ч FK, query, дүрэмд ашиглагдаагүй (зөвхөн өөрийн CRUD) тул
-- устгав. Багана өөрийн sequence (progress_type_id_seq)-тэй хамт устана.
DROP TABLE IF EXISTS public.acquisition_progress_status;

COMMENT ON COLUMN public.land_acquisition.status IS
    'Чөлөөлөлтийн төлөв (СТАТИК, lookup хүснэгтгүй; кодонд domain.LandAcquisitionStatus*): 1 = Шинэ (нэгж талбарын бүрдэл хийгдэж буй шат), 2 = Хээрийн судалгаа (үнэлгээ/олговор/нэгж талбарын төлөвтэй ажиллах; хоёр нэгж талбар давхцахгүй байх ёстой), 3 = Баталгаажсан (НЗД-ын захирамжийн дугаар/огноо заавал, бүх нэгж талбар эцсийн төлөвт; засвар түгжигдэнэ), 4 = Цуцлагдсан. Шилжилтийн дүрмийг acquisition_workflow хүснэгтээр тохируулна.';
COMMENT ON COLUMN public.acquisition_workflow.from_status IS 'land_acquisition.status-ийн статик утга (1–4); NULL = эхний төлөв';
COMMENT ON COLUMN public.acquisition_workflow.to_status IS 'land_acquisition.status-ийн статик утга (1–4)';
COMMENT ON COLUMN public.acquisition_progress.from_status IS 'land_acquisition.status-ийн статик утга (1–4); 0 = үүсгэгдсэн';
COMMENT ON COLUMN public.acquisition_progress.to_status IS 'land_acquisition.status-ийн статик утга (1–4)';
