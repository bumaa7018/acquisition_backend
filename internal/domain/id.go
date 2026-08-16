package domain

// ID — sdplatform схемийн бүх таних тэмдэг нь serial4 (int4) байдаг:
// sd_user.user_id, sd_role.role_id, sd_permission.permission_id,
// sd_organization.id, hr_department.department_id, hr_position.position_id,
// hr_employee.employee_id, bs_person.person_id.
//
// Тус схемд UUID огт байхгүй тул хэрэглэгч/эрх/HR-ийн ID-ууд бүгд энэ төрлийг
// хэрэглэнэ. Газар чөлөөлөлтийн (main DB) өөрийн мөрүүд uuid хэвээр.
type ID = int32
