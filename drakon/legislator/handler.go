package legislator

import (
	"fmt"
	"github.com/odysseia-greek/agora/aristoteles"
	"github.com/odysseia-greek/agora/aristoteles/models"
	"github.com/odysseia-greek/agora/plato/logging"
)

type DrakonHandler struct {
	Namespace string
	PodName   string
	Elastic   aristoteles.Client
	Roles     []string
	Indexes   []string
}

const (
	CreatorElasticRole  = "creator"
	SeederElasticRole   = "seeder"
	HybridElasticRole   = "hybrid"
	ApiElasticRole      = "api"
	AliasElasticRole    = "alias"
	TracingElasticIndex = "tracing"
)

func (d *DrakonHandler) CreateRoles() (bool, error) {
	logging.Debug("creating elastic roles based on labels")

	var created bool
	for _, index := range d.Indexes {
		if index == TracingElasticIndex {
			for _, role := range d.Roles {
				if role == CreatorElasticRole {
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

					roleCreated, err := d.Elastic.Access().CreateRole(role, putRole)
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

		for _, role := range d.Roles {
			if role == CreatorElasticRole {
				continue
			}

			logging.Debug(fmt.Sprintf("creating a role for index %s with role %s", index, role))
			var privileges []string
			names := []string{index}

			switch role {
			case SeederElasticRole:
				privileges = append(privileges, "delete_index")
				privileges = append(privileges, "create_index")
				privileges = append(privileges, "create")
			case HybridElasticRole:
				privileges = append(privileges, "create")
				privileges = append(privileges, "read")
				privileges = append(privileges, "index")
				privileges = append(privileges, "create_index")
			case ApiElasticRole:
				privileges = append(privileges, "read")
			case AliasElasticRole:
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
			roleCreated, err := d.Elastic.Access().CreateRole(roleName, putRole)
			if err != nil {
				return false, err
			}

			logging.Info(fmt.Sprintf("role: %s - created: %v", roleName, roleCreated))

			created = roleCreated
		}
	}

	return created, nil
}
