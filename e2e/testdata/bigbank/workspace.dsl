/*
 * Big Bank plc — combined workspace (SystemLandscape + Internet Banking System)
 * Source: https://github.com/avisi-cloud/structurizr-site-generatr/blob/main/docs/example/workspace.dsl
 */
workspace "Big Bank plc" "Example workspace illustrating key Structurizr DSL features." {

    model {
        customer = person "Personal Banking Customer" "A customer of the bank, with personal bank accounts." "Customer"

        acquirer = softwareSystem "Acquirer" "Facilitates PIN transactions for merchants." "External System"

        supportStaff = person "Customer Service Staff" "Customer service staff within the bank." "Bank Staff"
        backoffice = person "Back Office Staff" "Administration and support staff within the bank." "Bank Staff"

        mainframe = softwareSystem "Mainframe Banking System" "Stores all of the core banking information about customers, accounts, transactions, etc." "Existing System"
        email = softwareSystem "E-mail System" "The internal Microsoft Exchange e-mail system." "Existing System"
        atm = softwareSystem "ATM" "Allows customers to withdraw cash." "Existing System"

        internetBankingSystem = softwareSystem "Internet Banking System" "Allows customers to view information about their bank accounts, and make payments." {
            singlePageApplication = container "Single-Page Application" "Provides all of the Internet banking functionality to customers via their web browser." "JavaScript and Angular" "Web Browser"
            mobileApp = container "Mobile App" "Provides a limited subset of the Internet banking functionality to customers via their mobile device." "Xamarin" "Mobile App"
            webApplication = container "Web Application" "Delivers the static content and the Internet banking single page application." "Java and Spring MVC"
            apiApplication = container "API Application" "Provides Internet banking functionality via a JSON/HTTPS API." "Java and Spring MVC" {
                signinController = component "Sign In Controller" "Allows users to sign in to the Internet Banking System." "Spring MVC Rest Controller"
                accountsSummaryController = component "Accounts Summary Controller" "Provides customers with a summary of their bank accounts." "Spring MVC Rest Controller"
                resetPasswordController = component "Reset Password Controller" "Allows users to reset their passwords with a single use URL." "Spring MVC Rest Controller"
                securityComponent = component "Security Component" "Provides functionality related to signing in, changing passwords, etc." "Spring Bean"
                mainframeBankingSystemFacade = component "Mainframe Banking System Facade" "A facade onto the mainframe banking system." "Spring Bean"
                emailComponent = component "E-mail Component" "Sends e-mails to users." "Spring Bean"
            }
            database = container "Database" "Stores user registration information, hashed authentication credentials, access logs, etc." "Oracle Database Schema" "Database"
        }

        customer -> internetBankingSystem "Views account balances, and makes payments using"
        internetBankingSystem -> mainframe "Gets account information from, and makes payments using"
        internetBankingSystem -> email "Sends e-mail using"
        email -> customer "Sends e-mails to"
        customer -> supportStaff "Asks questions to" "Telephone"
        supportStaff -> mainframe "Uses"
        customer -> atm "Withdraws cash using"
        atm -> mainframe "Uses"
        backoffice -> mainframe "Uses"
        acquirer -> mainframe "Performs clearing and settlement"

        customer -> webApplication "Visits bigbank.com/ib using" "HTTPS"
        customer -> singlePageApplication "Views account balances, and makes payments using"
        customer -> mobileApp "Views account balances, and makes payments using"
        webApplication -> singlePageApplication "Delivers to the customer's web browser"

        singlePageApplication -> signinController "Makes API calls to" "JSON/HTTPS"
        singlePageApplication -> accountsSummaryController "Makes API calls to" "JSON/HTTPS"
        singlePageApplication -> resetPasswordController "Makes API calls to" "JSON/HTTPS"
        mobileApp -> signinController "Makes API calls to" "JSON/HTTPS"
        mobileApp -> accountsSummaryController "Makes API calls to" "JSON/HTTPS"
        mobileApp -> resetPasswordController "Makes API calls to" "JSON/HTTPS"
        signinController -> securityComponent "Uses"
        accountsSummaryController -> mainframeBankingSystemFacade "Uses"
        resetPasswordController -> securityComponent "Uses"
        resetPasswordController -> emailComponent "Uses"
        securityComponent -> database "Reads from and writes to" "JDBC"
        mainframeBankingSystemFacade -> mainframe "Makes API calls to" "XML/HTTPS"
        emailComponent -> email "Sends e-mail using"
    }

    views {
        systemLandscape "SystemLandscape" {
            include *
            title "Big Bank plc — System Landscape"
        }

        systemContext internetBankingSystem "SystemContext" {
            include *
            title "System Context of Internet Banking System"
            description "Describes the overall system context."
        }

        container internetBankingSystem "Containers" {
            include *
            title "Internet Banking System — Containers"
        }

        component apiApplication "Components" {
            include *
            title "API Application — Components"
        }
    }
}
