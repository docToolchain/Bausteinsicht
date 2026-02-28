<div class="row flex-xl-nowrap">
    <main class="col-12 col-md-12 col-xl-12 pl-md-12" role="main">
        <!-- Hero Section -->
        <div class="p-5 rounded" style="background: linear-gradient(135deg, #1a365d 0%, #2d5a87 100%); color: white; margin-bottom: 2rem;">
            <h1 style="font-size: 2.5rem; font-weight: 700;">Bausteinsicht</h1>
            <p class="lead" style="font-size: 1.4rem; opacity: 0.95;">
                Architecture-as-Code meets draw.io
            </p>
            <p style="font-size: 1.1rem; opacity: 0.85; max-width: 700px;">
                Maintain your software architecture in a structured JSON model and visualize it in draw.io — with full bidirectional synchronization. No more outdated diagrams.
            </p>
            <p style="margin-top: 1.5rem;">
                <a href="https://github.com/rdmueller/Bauteinsicht" class="btn btn-light btn-lg" style="font-weight: 600;">
                    Get Started
                </a>
                <a href="arc42/chapters/01_introduction_and_goals.html" class="btn btn-outline-light btn-lg" style="margin-left: 0.5rem;">
                    Architecture Docs
                </a>
            </p>
        </div>

        <!-- Key Features -->
        <h2 style="text-align: center; margin-bottom: 2rem; color: #1a365d;">What makes Bausteinsicht different?</h2>

        <div class="row row-cols-1 row-cols-md-2 mb-4">
            <div class="col mb-4">
                <div class="card h-100 shadow-sm border-0" style="border-left: 4px solid #2d5a87 !important;">
                    <div class="card-body">
                        <h4 class="card-title" style="color: #1a365d;">
                            <span style="font-size: 1.5rem; margin-right: 0.5rem;">&#128260;</span>
                            Bidirectional Sync
                        </h4>
                        <p class="card-text">
                            Edit in draw.io or in the JSON model — changes sync both ways.
                            No more "which diagram is the latest version?" questions.
                            Your model stays in sync with your diagrams automatically.
                        </p>
                    </div>
                </div>
            </div>
            <div class="col mb-4">
                <div class="card h-100 shadow-sm border-0" style="border-left: 4px solid #2d5a87 !important;">
                    <div class="card-body">
                        <h4 class="card-title" style="color: #1a365d;">
                            <span style="font-size: 1.5rem; margin-right: 0.5rem;">&#127912;</span>
                            draw.io as Frontend
                        </h4>
                        <p class="card-text">
                            No proprietary viewer needed. Use the tool everyone already knows.
                            Full control over styling via draw.io templates.
                            Your diagrams look exactly the way you want them.
                        </p>
                    </div>
                </div>
            </div>
            <div class="col mb-4">
                <div class="card h-100 shadow-sm border-0" style="border-left: 4px solid #2d5a87 !important;">
                    <div class="card-body">
                        <h4 class="card-title" style="color: #1a365d;">
                            <span style="font-size: 1.5rem; margin-right: 0.5rem;">&#129302;</span>
                            LLM-Friendly
                        </h4>
                        <p class="card-text">
                            JSON model format that LLMs read and write natively.
                            CLI commands let AI agents modify your architecture.
                            IDE autocompletion via JSON Schema — no plugin needed.
                        </p>
                    </div>
                </div>
            </div>
            <div class="col mb-4">
                <div class="card h-100 shadow-sm border-0" style="border-left: 4px solid #2d5a87 !important;">
                    <div class="card-body">
                        <h4 class="card-title" style="color: #1a365d;">
                            <span style="font-size: 1.5rem; margin-right: 0.5rem;">&#128736;</span>
                            Flexible Hierarchy
                        </h4>
                        <p class="card-text">
                            Not limited to 4 C4 levels. Define your own element kinds.
                            Components of components? No problem.
                            Your architecture, your rules.
                        </p>
                    </div>
                </div>
            </div>
        </div>

        <!-- How it works -->
        <div class="bg-light p-4 rounded mb-4">
            <h3 style="color: #1a365d; margin-bottom: 1.5rem;">How it works</h3>
            <div class="row">
                <div class="col-md-6 mb-3">
                    <div class="card h-100">
                        <div class="card-header" style="background-color: #1a365d; color: white;">
                            <h5 class="mb-0">1. Define your model</h5>
                            <small>JSON with IDE support</small>
                        </div>
                        <div class="card-body">
                            <pre style="background: #f8f9fa; padding: 1rem; border-radius: 4px; font-size: 0.85rem;"><code>{
  "model": {
    "webshop": {
      "kind": "system",
      "title": "Webshop",
      "children": {
        "api": {
          "kind": "container",
          "title": "REST API",
          "technology": "Spring Boot"
        }
      }
    }
  }
}</code></pre>
                        </div>
                    </div>
                </div>
                <div class="col-md-6 mb-3">
                    <div class="card h-100">
                        <div class="card-header" style="background-color: #2d5a87; color: white;">
                            <h5 class="mb-0">2. Sync and visualize</h5>
                            <small>Single binary CLI</small>
                        </div>
                        <div class="card-body">
                            <pre style="background: #f8f9fa; padding: 1rem; border-radius: 4px; font-size: 0.85rem;"><code># Sync model to draw.io
bausteinsicht sync

# Watch for changes
bausteinsicht watch

# Let AI modify architecture
bausteinsicht add element \
  --id cache \
  --kind container \
  --title "Redis Cache"

# Validate model
bausteinsicht validate</code></pre>
                        </div>
                    </div>
                </div>
            </div>
        </div>

        <!-- Comparison -->
        <div class="text-center mb-4 p-4">
            <h4 style="color: #1a365d;">Inspired by the best, built for draw.io</h4>
            <p class="text-muted">
                Bausteinsicht takes the best ideas from
                <strong>Structurizr</strong> and <strong>LikeC4</strong>
                and combines them with the world's most popular free diagramming tool.
            </p>
        </div>

        <!-- Call to Action -->
        <div class="text-center p-4 rounded" style="background-color: #f0f7ff;">
            <h3 style="color: #1a365d;">Ready to structure your architecture?</h3>
            <p>
                <a href="https://github.com/rdmueller/Bauteinsicht" class="btn btn-primary btn-lg">View on GitHub</a>
                <a href="spec/01_use_cases.html" class="btn btn-outline-secondary btn-lg">Specification</a>
                <a href="PRD/PRD-001-Bausteinsicht.html" class="btn btn-outline-secondary btn-lg">Product Requirements</a>
            </p>
        </div>
    </main>
</div>
