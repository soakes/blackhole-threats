import './style.css';

// Fetch env variables injected by Vite via define plugin
const siteUrl = process.env.PUBLIC_SITE_URL.replace(/\/?$/, '/');
const releaseVersion = process.env.PUBLIC_RELEASE_VERSION;
const commit = process.env.PUBLIC_COMMIT;
const buildDateStr = process.env.PUBLIC_BUILD_DATE;
const aptFingerprint = process.env.PUBLIC_APT_FINGERPRINT;

const shortCommit = commit.length > 10 ? commit.slice(0, 7) : commit;

// Format dates
const parsedBuildDate = new Date(buildDateStr);
const formattedBuildDate = Number.isNaN(parsedBuildDate.valueOf())
  ? buildDateStr
  : `${new Intl.DateTimeFormat("en-GB", {
      dateStyle: "long",
      timeStyle: "short",
      timeZone: "UTC",
    }).format(parsedBuildDate)} UTC`;

// Inject into DOM
document.addEventListener('DOMContentLoaded', () => {

  // Update Dynamic Text Fields
  const mappings = {
    'ui-version': releaseVersion,
    'ui-commit': shortCommit,
    'ui-date': formattedBuildDate,
    'ui-fingerprint': aptFingerprint,
    'footer-version': releaseVersion,
    'footer-commit': shortCommit
  };

  for (const [id, value] of Object.entries(mappings)) {
    const el = document.getElementById(id);
    if (el) el.textContent = value;
  }

  // Update Links dynamically
  const siteUrlLink = document.getElementById('site-url-link');
  if (siteUrlLink) siteUrlLink.href = siteUrl;

  const aptLink = document.getElementById('nav-apt-link');
  if (aptLink) aptLink.href = siteUrl;

  // Add the dynamic code snippet for APT setup
  const aptCodeContainer = document.getElementById('apt-code-container');
  if (aptCodeContainer) {
    aptCodeContainer.innerHTML = `
<pre><code>sudo install -d -m 0755 /etc/apt/keyrings
curl -fsSL ${siteUrl}blackhole-threats-archive-keyring.gpg \\
  | sudo tee /etc/apt/keyrings/blackhole-threats-archive-keyring.gpg >/dev/null

sudo tee /etc/apt/sources.list.d/blackhole-threats.sources >/dev/null &lt;&lt;'EOF'
Types: deb deb-src
URIs: ${siteUrl}
Suites: stable
Components: main
Signed-By: /etc/apt/keyrings/blackhole-threats-archive-keyring.gpg
EOF

sudo apt update && sudo apt install blackhole-threats</code></pre>`;
  }

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
