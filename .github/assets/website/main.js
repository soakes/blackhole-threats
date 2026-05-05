import './style.css';

// Fetch env variables injected by Vite via define plugin
const fallbackHighlights = [
  'Fetch feeds concurrently & normalize IPs',
  'Summarise networks to communities',
  'Export concise BGP route deltas',
];

const defaultMetadata = {
  siteUrl: process.env.PUBLIC_SITE_URL.replace(/\/?$/, '/'),
  releaseVersion: process.env.PUBLIC_RELEASE_VERSION,
  commit: process.env.PUBLIC_COMMIT,
  buildDate: process.env.PUBLIC_BUILD_DATE,
  aptFingerprint: process.env.PUBLIC_APT_FINGERPRINT,
  releaseHighlights: fallbackHighlights,
};

function formatBuildDate(value) {
  const parsedBuildDate = new Date(value);
  return Number.isNaN(parsedBuildDate.valueOf())
    ? value
    : `${new Intl.DateTimeFormat("en-GB", {
        dateStyle: "long",
        timeStyle: "short",
        timeZone: "UTC",
      }).format(parsedBuildDate)} UTC`;
}

function normalizeHighlights(value) {
  if (!Array.isArray(value)) {
    return fallbackHighlights;
  }

  const highlights = value
    .map((item) => `${item ?? ''}`.trim())
    .filter(Boolean)
    .slice(0, 3);

  return highlights.length > 0 ? highlights : fallbackHighlights;
}

function normalizeMetadata(value = {}) {
  const siteUrl = `${value.site_url || value.siteUrl || defaultMetadata.siteUrl}`.replace(/\/?$/, '/');
  const releaseVersion = `${value.release_version || value.releaseVersion || defaultMetadata.releaseVersion}`;
  const commit = `${value.commit || defaultMetadata.commit}`;
  const buildDate = `${value.build_date || value.buildDate || defaultMetadata.buildDate}`;
  const aptFingerprint = `${value.apt_fingerprint || value.aptFingerprint || defaultMetadata.aptFingerprint}`;

  return {
    siteUrl,
    releaseVersion,
    commit,
    shortCommit: commit.length > 10 ? commit.slice(0, 7) : commit,
    formattedBuildDate: formatBuildDate(buildDate),
    aptFingerprint,
    releaseHighlights: normalizeHighlights(value.release_highlights || value.releaseHighlights),
  };
}

function renderMetadata(metadata) {
  const mappings = {
    'ui-version': metadata.releaseVersion,
    'ui-commit': metadata.shortCommit,
    'ui-date': metadata.formattedBuildDate,
    'ui-fingerprint': metadata.aptFingerprint,
    'footer-version': metadata.releaseVersion,
    'footer-commit': metadata.shortCommit,
  };

  for (const [id, value] of Object.entries(mappings)) {
    const el = document.getElementById(id);
    if (el) {
      el.textContent = value;
    }
  }

  const siteUrlLink = document.getElementById('site-url-link');
  if (siteUrlLink) {
    siteUrlLink.href = metadata.siteUrl;
  }

  const aptLink = document.getElementById('nav-apt-link');
  if (aptLink) {
    aptLink.href = metadata.siteUrl;
  }

  const aptCodeContainer = document.getElementById('apt-code-container');
  if (aptCodeContainer) {
    aptCodeContainer.innerHTML = `
<pre><code>sudo install -d -m 0755 /etc/apt/keyrings
curl -fsSL ${metadata.siteUrl}blackhole-threats-archive-keyring.gpg \\
  | sudo tee /etc/apt/keyrings/blackhole-threats-archive-keyring.gpg >/dev/null

sudo tee /etc/apt/sources.list.d/blackhole-threats.sources >/dev/null &lt;&lt;'EOF'
Types: deb deb-src
URIs: ${metadata.siteUrl}
Suites: stable
Components: main
Signed-By: /etc/apt/keyrings/blackhole-threats-archive-keyring.gpg
EOF

sudo apt update && sudo apt install blackhole-threats</code></pre>`;
  }

  const highlightsList = document.getElementById('ui-highlights');
  if (highlightsList) {
    highlightsList.replaceChildren(
      ...metadata.releaseHighlights.map((highlight) => {
        const item = document.createElement('li');
        item.textContent = highlight;
        return item;
      }),
    );
  }
}

// Inject into DOM
document.addEventListener('DOMContentLoaded', () => {
  renderMetadata(normalizeMetadata(defaultMetadata));

  fetch('./website-metadata.json', { cache: 'no-store' })
    .then((response) => {
      if (!response.ok) {
        return null;
      }
      return response.json();
    })
    .then((metadata) => {
      if (metadata) {
        renderMetadata(normalizeMetadata(metadata));
      }
    })
    .catch(() => {
      // Keep the build-time defaults if the published metadata file is absent.
    });

  // Intersection Observer for scroll animations (optional extra flair if we had components lower down)
  const observerOptions = {
    root: null,
    rootMargin: '0px',
    threshold: 0.1
  };
  
  const observer = new IntersectionObserver((entries, observer) => {
    entries.forEach(entry => {
      if (entry.isIntersecting) {
        entry.target.classList.add('fade-in');
        observer.unobserve(entry.target);
      }
    });
  }, observerOptions);

  document.querySelectorAll('.glass-card').forEach(card => {
    // If they aren't already animated on load, observe them
    if(!card.closest('.stagger-in') && !card.closest('.fade-in')) {
      card.style.opacity = '0';
      observer.observe(card);
    }
  });
});
