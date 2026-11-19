
Feature: Depositing cash
  As an authenticated customer
  I want to deposit cash into my account
  So that I can increase my available balance

  Rule: Deposits must be greater than zero

  Scenario: Successful cash deposit
    Given Alice has 100.00 in their account
    When Alice deposits 50.00
    Then the deposit is successful
    And the account balance is 150.00

  Scenario: Declined deposit with negative amount
    Given Alice has 100.00 in their account
    When Alice deposits -20.00
    Then the deposit is declined
    And the account balance remains 100.00

  Scenario: Declined deposit with zero amount
    Given Alice has 100.00 in their account
    When Alice deposits 0.00
    Then the deposit is declined
    And the account balance remains 100.00
