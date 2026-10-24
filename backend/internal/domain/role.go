package domain

type Role string

const (
	RoleUser       Role = "USER"
	RoleAdmin      Role = "ADMIN"
	RoleSuperAdmin Role = "SUPERADMIN"
)

func (r Role) Valid() bool {
	_, ok := rolePermissions[r]
	return ok
}

type Permission string

const (
	PermProductsWrite   Permission = "products.write"
	PermCategoriesWrite Permission = "categories.write"
	PermProductsImport  Permission = "products.import"
	PermOrdersCreate    Permission = "orders.create"
	PermOrdersReadOwn   Permission = "orders.read_own"
	PermOrdersReadAll   Permission = "orders.read_all"
	PermAdminsManage    Permission = "admins.manage"
)

var rolePermissions = map[Role]map[Permission]bool{
	RoleUser: {
		PermOrdersCreate:  true,
		PermOrdersReadOwn: true,
	},
	RoleAdmin: {
		PermProductsWrite:   true,
		PermCategoriesWrite: true,
		PermProductsImport:  true,
		PermOrdersReadAll:   true,
	},
	RoleSuperAdmin: {
		PermProductsWrite:   true,
		PermCategoriesWrite: true,
		PermProductsImport:  true,
		PermOrdersReadAll:   true,
		PermAdminsManage:    true,
	},
}

func (r Role) Can(p Permission) bool {
	return rolePermissions[r][p]
}
