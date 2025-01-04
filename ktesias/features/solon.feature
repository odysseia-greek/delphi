Feature: Solon
  In order to validate the workings of odysseia-greeks backend infra
  As the chief developer
  We need to be able to validate the functioning of Solon the gatekeeper for vault

  @solon
#    I am guessing this should at some point fail because it implies anyone can just ask for a one time token to the vault
#    this logic is mostly internal for ptolemaios but should be refined at some point
  Scenario: Asking for a one time token without having the proper annotations
    Given solon returns a healthy response
    When a request is made for a one time token without annotations
    Then a one time token is returned

  @solon
  Scenario: Creating credentials while not having the proper annotations should return an error
    Given solon returns a healthy response
    When a request is made to register the running pod with incorrect role and access annotations
    Then a validation error is returned that the pod annotations do not match the requested role and access

  @solon
  # this test also means that could impersonate a different pod and still receive the registry
  Scenario: Creating credentials with the correct annotations should be allowed
    Given solon returns a healthy response
    When a request is made to register the running pod with correct role and access annotations
    Then a successful register should be made
