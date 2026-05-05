workspace "Order System" "Example for import tests" {

  model {
    user = person "User" "A customer using the system"
    orderSystem = softwareSystem "Order System" "Handles order processing" {
      webApp = container "Web App" "React SPA" "TypeScript"
      api    = container "API"     "REST backend" "Go"
      db     = container "Database" "" "PostgreSQL"
    }
    externalPayment = softwareSystem "Payment Provider" "External payment gateway"

    user       -> webApp         "Uses" "HTTPS"
    webApp     -> api            "Calls" "HTTP/JSON"
    api        -> db             "Reads/Writes"
    api        -> externalPayment "Processes payment" "HTTPS"
  }

  views {
    systemContext orderSystem "Context" {
      include *
      autoLayout
    }
    container orderSystem "Containers" {
      include *
    }
  }
}
