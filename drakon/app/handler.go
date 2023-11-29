package app

import (
	"fmt"
	"github.com/odysseia-greek/agora/aristoteles/models"
	"github.com/odysseia-greek/agora/plato/config"
	"github.com/odysseia-greek/agora/plato/logging"
	configs "github.com/odysseia-greek/delphi/drakon/config"
)

type DrakonHandler struct {
	Config *configs.Config
}

func (d *DrakonHandler) CreateRoles() (bool, error) {
	logging.Debug("creating elastic roles based on labels")

	var created bool
	for _, index := range d.Config.Indexes {
		if index == config.TracingElasticIndex {
			for _, role := range d.Config.Roles {
				if role == config.CreatorElasticRole {
					logging.Debug(fmt.Sprintf("creating a role for index %s with role %s", index, role))
					var privileges []string

					privileges = append(privileges, "create")
					privileges = append(privileges, "index")

					names := []string{index}

					indices := []models.Indices{
						{
							Names:      names,
							Privileges: privileges,
							Query:      "",
						},
					}

					putRole := models.CreateRoleRequest{
						Cluster:      []string{"all"},
						Indices:      indices,
						Applications: []models.Application{},
						RunAs:        nil,
						Metadata:     models.Metadata{Version: 1},
					}

					roleCreated, err := d.Config.Elastic.Access().CreateRole(role, putRole)
					if err != nil {
						return false, err
					}

					logging.Info(fmt.Sprintf("role: %s - created: %v", role, roleCreated))
					created = roleCreated
				} else {
					continue
				}
			}
		}

		for _, role := range d.Config.Roles {
			if role == config.CreatorElasticRole {
				continue
			}

			logging.Debug(fmt.Sprintf("creating a role for index %s with role %s", index, role))
			var privileges []string
			names := []string{index}

			switch role {
			case config.SeederElasticRole:
				privileges = append(privileges, "delete_index")
				privileges = append(privileges, "create_index")
				privileges = append(privileges, "create")
			case config.HybridElasticRole:
				privileges = append(privileges, "create")
				privileges = append(privileges, "read")
				privileges = append(privileges, "index")
			case config.ApiElasticRole:
				privileges = append(privileges, "read")
			case config.AliasElasticRole:
				privileges = append(privileges, "delete_index")
				privileges = append(privileges, "create_index")
				privileges = append(privileges, "manage")
				privileges = append(privileges, "all")
				names = []string{fmt.Sprintf("%s*", index)}
			}

			indices := []models.Indices{
				{
					Names:      names,
					Privileges: privileges,
					Query:      "",
				},
			}

			putRole := models.CreateRoleRequest{
				Cluster:      []string{"all"},
				Indices:      indices,
				Applications: []models.Application{},
				RunAs:        nil,
				Metadata:     models.Metadata{Version: 1},
			}

			roleName := fmt.Sprintf("%s_%s", index, role)
			roleCreated, err := d.Config.Elastic.Access().CreateRole(roleName, putRole)
			if err != nil {
				return false, err
			}

			logging.Info(fmt.Sprintf("role: %s - created: %v", roleName, roleCreated))

			created = roleCreated
		}
	}

	return created, nil
}
