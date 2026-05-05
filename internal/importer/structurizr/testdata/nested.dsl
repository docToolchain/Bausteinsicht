workspace {
  model {
    // Inline relationships inside element block
    customer = person "Customer" {
      -> frontend "Uses"
    }
    mySystem = softwareSystem "My System" {
      frontend = container "Frontend" "SPA" "React" {
        searchComp = component "Search" "Full-text search" "TypeScript"
      }
      backend = container "Backend" "REST API" "Go"
    }

    frontend -> backend "API calls" "HTTP"
  }

  views {
    systemLandscape {
      include *
    }
  }
}
