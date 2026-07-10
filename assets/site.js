(() => {
  const translations = window.GoSherpaTranslations || {};
  const commandDemos = {
    explain: {
      command: "gosherpa explain ParseFile",
      output: `$ gosherpa explain ParseFile

EXPLAIN

TARGET
  ParseFile (function)

DEFINITION
  internal/sherpa/parse.go:15

RISK
  medium
  - References found: 7.
  - Related tests found: 4.

READING ORDER
  1. Definition - internal/sherpa/parse.go:15
     Start with the symbol declaration and nearby implementation.
  2. Caller: LoadRepository - internal/sherpa/repository.go:28
     See how repository loading depends on parsing.
  3. Test: TestParseFile - internal/sherpa/parse_test.go:11
     Check expected behavior and regression coverage.

SUGGESTED TESTS
  TestParseFile                       internal/sherpa/parse_test.go:11 (direct)

TEST PLAN
  go test ./internal/sherpa`
    },
    context: {
      command: "gosherpa context symbol ParseFile --max-references 20 --max-tests 10 --json",
      output: `$ gosherpa context symbol ParseFile --max-references 20 --max-tests 10 --json

{
  "schemaVersion": 1,
  "command": "context symbol",
  "target": "ParseFile",
  "data": {
    "identity": {
      "package": "./internal/sherpa",
      "symbol": "ParseFile",
      "kind": "function",
      "definition": {
        "file": "internal/sherpa/parse.go",
        "line": 15
      }
    },
    "confidence": "medium",
    "sourceContext": {
      "startLine": 12,
      "endLine": 35
    },
    "references": [
      {
        "file": "internal/sherpa/repository.go",
        "line": 28
      }
    ],
    "relatedTests": [
      {
        "name": "TestParseFile",
        "file": "internal/sherpa/parse_test.go",
        "line": 11
      }
    ]
  }
}`
    },
    impact: {
      command: "gosherpa impact diff --base HEAD",
      output: `$ gosherpa impact diff --base HEAD

IMPACT DIFF

CHANGED FILES
  internal/sherpa/parse.go

CHANGED PACKAGES
  ./internal/sherpa

AFFECTED PACKAGES
  ./internal/sherpa
  ./cmd/gosherpa

AFFECTED SYMBOLS
  ./internal/sherpa.ParseFile

TEST PLAN
  go test ./internal/sherpa
  go test ./cmd/gosherpa`
    },
    tests: {
      command: "gosherpa tests affected --base HEAD",
      output: `$ gosherpa tests affected --base HEAD

AFFECTED TESTS

  TestParseFile                       internal/sherpa/parse_test.go:11 (direct)
  TestRepositoryLoad                  internal/sherpa/repository_test.go:19
  TestMainRunsSymbolCommand           cmd/gosherpa/main_test.go:2445

TEST PLAN
  Direct:
    go test ./internal/sherpa
      Direct tests reference changed symbol ParseFile.

  Fallback:
    go test ./cmd/gosherpa
      Compile impacted CLI package and cover command behavior.`
    }
  };
  const supportedLanguages = Object.keys(translations);
  const defaultLanguage = "de";
  if (!translations[defaultLanguage]) {
    return;
  }

  const readSavedLanguage = () => {
    try {
      return window.localStorage.getItem("gosherpa-language");
    } catch {
      return "";
    }
  };

  const saveLanguage = (language) => {
    try {
      window.localStorage.setItem("gosherpa-language", language);
    } catch {
      // Local storage can be unavailable in strict browser contexts.
    }
  };

  const getCurrentLanguage = () => document.documentElement.lang || defaultLanguage;

  const getTranslation = (key, fallback) => {
    const language = getCurrentLanguage();
    const dictionary = translations[language] || translations[defaultLanguage];
    return dictionary[key] || fallback;
  };

  const restoreCopyLabel = (label, key = "demo.copy", fallback = "Copy") => {
    label.textContent = getTranslation(key, fallback);
  };

  const copyText = async (text) => {
    if (navigator.clipboard && window.isSecureContext) {
      await navigator.clipboard.writeText(text);
      return;
    }

    const textArea = document.createElement("textarea");
    textArea.value = text;
    textArea.setAttribute("readonly", "");
    textArea.style.position = "fixed";
    textArea.style.inset = "-9999px auto auto -9999px";
    document.body.appendChild(textArea);
    textArea.select();
    document.execCommand("copy");
    textArea.remove();
  };

  const setupCommandDemo = () => {
    const demo = document.querySelector("[data-command-demo]");
    if (!demo) {
      return;
    }

    const tabs = Array.from(demo.querySelectorAll("[data-demo-command]"));
    const output = demo.querySelector("[data-demo-output]");
    const commandLine = demo.querySelector("[data-demo-command-line]");
    const copyButton = demo.querySelector("[data-copy-command]");
    const copyLabel = demo.querySelector("[data-copy-label]");
    const panel = demo.querySelector('[role="tabpanel"]');
    let activeKey = tabs[0]?.dataset.demoCommand || "";
    let restoreTimer;

    const activate = (key) => {
      const item = commandDemos[key];
      if (!item) {
        return;
      }

      activeKey = key;
      if (output) {
        output.textContent = item.output;
      }
      if (commandLine) {
        commandLine.textContent = item.command;
      }

      tabs.forEach((tab) => {
        const isActive = tab.dataset.demoCommand === key;
        tab.classList.toggle("is-active", isActive);
        tab.setAttribute("aria-selected", String(isActive));
        tab.setAttribute("tabindex", isActive ? "0" : "-1");
        if (isActive && panel) {
          panel.setAttribute("aria-labelledby", tab.id);
        }
      });
    };

    tabs.forEach((tab, index) => {
      tab.addEventListener("click", () => {
        activate(tab.dataset.demoCommand);
      });

      tab.addEventListener("keydown", (event) => {
        const keys = ["ArrowRight", "ArrowDown", "ArrowLeft", "ArrowUp", "Home", "End"];
        if (!keys.includes(event.key)) {
          return;
        }

        event.preventDefault();
        let nextIndex = index;
        if (event.key === "ArrowRight" || event.key === "ArrowDown") {
          nextIndex = (index + 1) % tabs.length;
        } else if (event.key === "ArrowLeft" || event.key === "ArrowUp") {
          nextIndex = (index - 1 + tabs.length) % tabs.length;
        } else if (event.key === "Home") {
          nextIndex = 0;
        } else if (event.key === "End") {
          nextIndex = tabs.length - 1;
        }

        tabs[nextIndex].focus();
        activate(tabs[nextIndex].dataset.demoCommand);
      });
    });

    if (copyButton && copyLabel) {
      copyButton.addEventListener("click", async () => {
        const item = commandDemos[activeKey];
        if (!item) {
          return;
        }

        await copyText(item.command);
        window.clearTimeout(restoreTimer);
        copyLabel.textContent = getTranslation("code.copied", "Copied");
        restoreTimer = window.setTimeout(() => restoreCopyLabel(copyLabel), 1400);
      });
    }

    activate(activeKey);
  };

  const setupCopyableCodeBlocks = () => {
    document.querySelectorAll("[data-copy-block-button]").forEach((button) => {
      const block = button.closest("[data-copyable-code]");
      const source = block?.querySelector("[data-copy-content]");
      const label = button.querySelector("[data-copy-block-label]");
      let restoreTimer;

      if (!source || !label) {
        return;
      }

      button.addEventListener("click", async () => {
        const text = source.textContent.trim();
        if (!text) {
          return;
        }

        await copyText(text);
        window.clearTimeout(restoreTimer);
        label.textContent = getTranslation("code.copied", "Copied");
        restoreTimer = window.setTimeout(() => restoreCopyLabel(label, "code.copy", "Copy"), 1400);
      });
    });
  };

  const setupMobileMenu = () => {
    const toggle = document.querySelector("[data-menu-toggle]");
    const menu = document.querySelector("[data-mobile-menu]");
    if (!toggle || !menu) {
      return;
    }

    const setOpen = (isOpen) => {
      toggle.classList.toggle("is-open", isOpen);
      toggle.setAttribute("aria-expanded", String(isOpen));
      toggle.setAttribute("aria-label", getTranslation(isOpen ? "menu.close" : "menu.open", isOpen ? "Close menu" : "Open menu"));
      menu.hidden = !isOpen;
    };

    toggle.addEventListener("click", () => {
      setOpen(toggle.getAttribute("aria-expanded") !== "true");
    });

    menu.querySelectorAll("a").forEach((link) => {
      link.addEventListener("click", () => {
        setOpen(false);
      });
    });

    document.addEventListener("keydown", (event) => {
      if (event.key === "Escape" && toggle.getAttribute("aria-expanded") === "true") {
        setOpen(false);
        toggle.focus();
      }
    });

    const desktopQuery = window.matchMedia("(min-width: 921px)");
    const closeOnDesktop = (event) => {
      if (event.matches) {
        setOpen(false);
      }
    };

    if (desktopQuery.addEventListener) {
      desktopQuery.addEventListener("change", closeOnDesktop);
    }
  };

  const getInitialLanguage = () => {
    const requestedLanguage = new URLSearchParams(window.location.search).get("lang");
    if (supportedLanguages.includes(requestedLanguage)) {
      return requestedLanguage;
    }

    const savedLanguage = readSavedLanguage();
    if (supportedLanguages.includes(savedLanguage)) {
      return savedLanguage;
    }

    const browserLanguage = navigator.language.slice(0, 2).toLowerCase();
    return supportedLanguages.includes(browserLanguage) ? browserLanguage : defaultLanguage;
  };

  const setMetaContent = (selector, content) => {
    const element = document.querySelector(selector);
    if (element) {
      element.setAttribute("content", content);
    }
  };

  const translateElements = (language) => {
    const dictionary = translations[language] || translations[defaultLanguage];

    document.documentElement.lang = language;
    document.title = dictionary["meta.title"];
    setMetaContent('meta[name="description"]', dictionary["meta.description"]);
    setMetaContent('meta[property="og:description"]', dictionary["meta.ogDescription"]);
    setMetaContent('meta[property="og:locale"]', dictionary["meta.locale"]);

    document.querySelectorAll("[data-i18n]").forEach((element) => {
      const value = dictionary[element.dataset.i18n];
      if (value !== undefined) {
        element.textContent = value;
      }
    });

    document.querySelectorAll("[data-i18n-label]").forEach((element) => {
      const value = dictionary[element.dataset.i18nLabel];
      if (value !== undefined) {
        element.setAttribute("aria-label", value);
      }
    });

    document.querySelectorAll("[data-i18n-alt]").forEach((element) => {
      const value = dictionary[element.dataset.i18nAlt];
      if (value !== undefined) {
        element.setAttribute("alt", value);
      }
    });

    document.querySelectorAll("[data-lang-option]").forEach((button) => {
      const isActive = button.dataset.langOption === language;
      button.classList.toggle("is-active", isActive);
      button.setAttribute("aria-pressed", String(isActive));
    });

    document.querySelectorAll("[data-menu-toggle]").forEach((button) => {
      const key = button.getAttribute("aria-expanded") === "true" ? "menu.close" : "menu.open";
      const value = dictionary[key];
      if (value !== undefined) {
        button.setAttribute("aria-label", value);
      }
    });

    saveLanguage(language);
  };

  document.querySelectorAll("[data-lang-option]").forEach((button) => {
    button.addEventListener("click", () => {
      translateElements(button.dataset.langOption);
    });
  });

  setupMobileMenu();
  setupCommandDemo();
  setupCopyableCodeBlocks();
  translateElements(getInitialLanguage());
})();
