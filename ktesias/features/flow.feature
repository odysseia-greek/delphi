Feature: Flow
  In order to validate the workings of odysseia-greeks backend infra
  As the chief developer
  We need to be able to validate the functioning of the entire infra flow

@flow
Scenario: a pod register is made and checked in elastic
  Given solon returns a healthy response
  And a request is made to register the running pod with correct role and access annotations
  And a request is made for a one time token
  And an elastic client is created with the one time token that was created
  When a call is made to the correct index with the correct action
  Then a 200 should be returned

  @flow
  Scenario: the side car ptolemaios is used to query data
    Given ptolemaios is asked for the current config
    And an elastic client is created with the vault data
    When a call is made to the correct index with the correct action
    Then a 200 should be returned

  @flow
  Scenario: cross access to an index is blocked by the roles created by drakon
    Given ptolemaios is asked for the current config
    And an elastic client is created with the vault data
    When a call is made to an index not part of the annotations
    Then a 403 should be returned