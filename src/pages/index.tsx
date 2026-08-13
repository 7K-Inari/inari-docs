import clsx from 'clsx';
import Link from '@docusaurus/Link';
import useDocusaurusContext from '@docusaurus/useDocusaurusContext';
import Layout from '@theme/Layout';
import Heading from '@theme/Heading';

import styles from './index.module.css';

const sections = [
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

export default function Home(): React.ReactNode {
  const {siteConfig} = useDocusaurusContext();
  return (
    <Layout title="Home" description={siteConfig.tagline}>
      <header className={clsx('hero hero--primary', styles.heroBanner)}>
        <div className="container">
          <Heading as="h1" className="hero__title">
            {siteConfig.title}
          </Heading>
          <p className="hero__subtitle">{siteConfig.tagline}</p>
          <div className={styles.buttons}>
            <Link className="button button--secondary button--lg" to="/docs/architecture/inari-platform-plan">
              Read the platform plan
            </Link>
          </div>
        </div>
      </header>
      <main>
        <section className={styles.sections}>
          <div className="container">
            <div className="row">
              {sections.map((s) => (
                <div key={s.title} className="col col--4">
                  <div className={styles.card}>
                    <Heading as="h3">
                      <Link to={s.to}>{s.title}</Link>
                    </Heading>
                    <p>{s.description}</p>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </section>
      </main>
    </Layout>
  );
}
