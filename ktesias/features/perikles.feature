Feature: Perikles
  In order to validate the workings of odysseia-greeks backend infra
  As the chief developer
  We need to be able to validate the functioning of Perikles the admissions webbook

  @perikles
  Scenario Outline: Creating a deployment with annotations will trigger a creation process
    Given a deployment is created with role "<role>", access "<access>", host "<hostname>" and being a client of "<client>"
    When the created resource "<hostname>" is checked after a wait
    Then a secret should be created for tls certs for host "<hostname>"
    And CiliumNetWorkPolicies should exist for role "<role>" from host "<hostname>"
    And a CiliumNetWorkPolicy should exist for access from the deployment "<hostname>" to the host "<client>"
    Examples:
    | role   | access    | hostname    | client |
    | api    | testindex | ktesias-bdd | solon  |

  @wip
  Scenario: a deployment with the correct annotations
    Given solon returns a healthy response
    And a request is made to register the running pod with correct role and access annotations
    And a request is made for a one time token
    And an elastic client is created with the vault data
    When a call is made to the correct index with the correct action
    Then a 200 should be returned