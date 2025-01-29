Feature: Solon
  In order to validate the workings of odysseia-greeks backend infra
  As the chief developer
  We need to be able to validate the functioning of Solon the gatekeeper for vault

  @solon
  Scenario: Creating credentials while not having the proper annotations should return an error
    Given solon returns a healthy response
    When a request is made to register the running pod with incorrect role and access annotations
    Then a validation error is returned that the pod annotations do not match the requested role and access

  @solon
  Scenario: Creating credentials with the correct annotations should be allowed
    Given solon returns a healthy response
    When a request is made to register the running pod with correct role and access annotations
    Then a successful register should be made

  @solon
  Scenario: Creating credentials with an impersonated podname will still lead to a secret for the calling pod
    Given solon returns a healthy response
    When a request is made to register the running pod with correct role and access annotations but a mismatched podname
    When a request is made for a one time token
    And a request is made for a second one time token
    Then the token from the mismatched podname is not valid
    And the token from the actual podname is valid
    And the tokens are not usable twice

  @solon
  Scenario: Creating credentials with a valid podname will lead to a secret being retrievable
    Given solon returns a healthy response
    When a request is made to register the running pod with correct role and access annotations
    When a request is made for a one time token
    Then a one time token is returned
    And the token from the actual podname is valid