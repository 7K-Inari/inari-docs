"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.default = Home;
var clsx_1 = require("clsx");
var Link_1 = require("@docusaurus/Link");
var useDocusaurusContext_1 = require("@docusaurus/useDocusaurusContext");
var Layout_1 = require("@theme/Layout");
var Heading_1 = require("@theme/Heading");
var index_module_css_1 = require("./index.module.css");
var sections = [
    {
        title: 'Architecture',
        to: '/docs/architecture/inari-platform-plan',
        description: 'The canonical architecture & development plan for the Inari platform.',
    },
    {
        title: 'User Guide',
        to: '/docs/user-guide/',
        description: 'For developers: catalogs, deployments, and your resources.',
    },
    {
        title: 'Operator Guide',
        to: '/docs/operator-guide/',
        description: 'For platform engineers: bootstrap, operate, and govern the fleet.',
    },
    {
        title: 'Extension Authors',
        to: '/docs/extension-authors/',
        description: 'Build backend plugins and UI extensions on the Inari SDKs.',
    },
    {
        title: 'Tutorials',
        to: '/docs/tutorials/',
        description: 'Hands-on walkthroughs for common Inari workflows.',
    },
    {
        title: 'ADRs',
        to: '/docs/category/adrs',
        description: 'Architecture Decision Records — what we chose, and why.',
    },
];
function Home() {
    var siteConfig = (0, useDocusaurusContext_1.default)().siteConfig;
    return (<Layout_1.default title="Home" description={siteConfig.tagline}>
      <header className={(0, clsx_1.default)('hero hero--primary', index_module_css_1.default.heroBanner)}>
        <div className="container">
          <Heading_1.default as="h1" className="hero__title">
            {siteConfig.title}
          </Heading_1.default>
          <p className="hero__subtitle">{siteConfig.tagline}</p>
          <div className={index_module_css_1.default.buttons}>
            <Link_1.default className="button button--secondary button--lg" to="/docs/architecture/inari-platform-plan">
              Read the platform plan
            </Link_1.default>
          </div>
        </div>
      </header>
      <main>
        <section className={index_module_css_1.default.sections}>
          <div className="container">
            <div className="row">
              {sections.map(function (s) { return (<div key={s.title} className="col col--4">
                  <div className={index_module_css_1.default.card}>
                    <Heading_1.default as="h3">
                      <Link_1.default to={s.to}>{s.title}</Link_1.default>
                    </Heading_1.default>
                    <p>{s.description}</p>
                  </div>
                </div>); })}
            </div>
          </div>
        </section>
      </main>
    </Layout_1.default>);
}
