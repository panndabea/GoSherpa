(() => {
  const translations = window.GoSherpaTranslations || {};

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

    saveLanguage(language);
  };

  document.querySelectorAll("[data-lang-option]").forEach((button) => {
    button.addEventListener("click", () => {
      translateElements(button.dataset.langOption);
    });
  });

  translateElements(getInitialLanguage());
})();
