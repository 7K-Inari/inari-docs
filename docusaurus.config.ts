import {themes as prismThemes} from 'prism-react-renderer';
import type {Config} from '@docusaurus/types';
import type * as Preset from '@docusaurus/preset-classic';

const config: Config = {
  title: 'Inari Docs',
  tagline: 'Documentation for the Inari multi-tenant Internal Developer Platform',

  url: 'https://7k-inari.github.io',
  baseUrl: '/inari-docs/',

  organizationName: '7K-Inari',
  projectName: 'inari-docs',

  trailingSlash: true,

  // M0: the canonical plan doc references images (diagrams/) that do not exist
  // yet. Downgrade to 'warn' for now; tighten to 'throw' once content stabilizes.
  onBrokenLinks: 'warn',
  markdown: {
    mermaid: true,
    hooks: {
      onBrokenMarkdownLinks: 'warn',
      onBrokenMarkdownImages: 'warn',
    },
  },

  themes: ['@docusaurus/theme-mermaid'],

  i18n: {
    defaultLocale: 'en',
    locales: ['en'],
  },

  presets: [
    [
      'classic',
      {
        docs: {
          // All content lives in the repo-root docs/ folder — single source of
          // truth, including the canonical architecture plan and ADRs.
          path: 'docs',
          routeBasePath: 'docs',
          sidebarPath: './sidebars.ts',
          editUrl: 'https://github.com/7K-Inari/inari-docs/tree/main/',
        },
        blog: false,
        theme: {
          customCss: './src/css/custom.css',
        },
      } satisfies Preset.Options,
    ],
  ],

  themeConfig: {
    navbar: {
      title: 'Inari Docs',
      items: [
        {type: 'docSidebar', sidebarId: 'docs', position: 'left', label: 'Docs'},
        {to: '/docs/architecture/inari-platform-plan', label: 'Architecture', position: 'left'},
        {to: '/docs/category/adrs', label: 'ADRs', position: 'left'},
        {href: 'https://github.com/7K-Inari/inari-docs', label: 'GitHub', position: 'right'},
      ],
    },
    footer: {
      style: 'dark',
      links: [
        {
          title: 'Docs',
          items: [
            {label: 'Architecture', to: '/docs/architecture/inari-platform-plan'},
            {label: 'User Guide', to: '/docs/user-guide/'},
            {label: 'Operator Guide', to: '/docs/operator-guide/'},
          ],
        },
        {
          title: 'More',
          items: [
            {label: 'ADRs', to: '/docs/category/adrs'},
            {label: 'GitHub', href: 'https://github.com/7K-Inari/inari-docs'},
          ],
        },
      ],
      copyright: `Copyright © ${new Date().getFullYear()} Inari contributors. Apache-2.0.`,
    },
    prism: {
      theme: prismThemes.github,
      darkTheme: prismThemes.dracula,
    },
  } satisfies Preset.ThemeConfig,
};

export default config;
