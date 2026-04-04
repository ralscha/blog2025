import { DOCUMENT } from '@angular/common';
import { DestroyRef, Injectable, inject, signal } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { SwUpdate, type VersionEvent } from '@angular/service-worker';
import { interval, startWith } from 'rxjs';

@Injectable({ providedIn: 'root' })
export class AppUpdateService {
  readonly #destroyRef = inject(DestroyRef);
  readonly #document = inject(DOCUMENT);
  readonly #swUpdate = inject(SwUpdate);
  readonly #updateAvailable = signal(false);
  readonly #isChecking = signal(false);
  readonly #isActivating = signal(false);
  readonly #statusMessage = signal('Waiting for the next update check.');

  readonly updateAvailable = this.#updateAvailable.asReadonly();
  readonly isChecking = this.#isChecking.asReadonly();
  readonly isActivating = this.#isActivating.asReadonly();
  readonly statusMessage = this.#statusMessage.asReadonly();

  constructor() {
    if (!this.#swUpdate.isEnabled) {
      this.#statusMessage.set(
        'Service worker updates are disabled in development mode. Use a production build to test them locally.',
      );
      return;
    }

    this.#swUpdate.versionUpdates.subscribe((event) => {
      this.#handleVersionEvent(event);
    });

    this.#registerUpdateCheckTriggers();

    interval(60_000)
      .pipe(startWith(0), takeUntilDestroyed(this.#destroyRef))
      .subscribe(() => this.#checkForUpdates());
  }

  async #checkForUpdates(): Promise<void> {
    if (!this.#swUpdate.isEnabled || this.#isChecking() || this.#updateAvailable()) {
      return;
    }

    this.#isChecking.set(true);
    this.#statusMessage.set('Checking for a newer app version...');

    try {
      const updateFound = await this.#swUpdate.checkForUpdate();

      if (!updateFound) {
        this.#statusMessage.set('No new version is available right now.');
      }
    } catch (error) {
      console.error('Failed to check for updates', error);
      this.#statusMessage.set('The app could not check for updates. See the console for details.');
    } finally {
      this.#isChecking.set(false);
    }
  }

  async activateUpdate(): Promise<void> {
    if (!this.#swUpdate.isEnabled || this.#isActivating()) {
      return;
    }

    this.#isActivating.set(true);
    this.#statusMessage.set('Applying the latest version and reloading the app...');

    try {
      await this.#swUpdate.activateUpdate();
      document.location.reload();
    } catch (error) {
      console.error('Could not activate the latest version', error);
      this.#isActivating.set(false);
      this.#statusMessage.set(
        'Could not apply the latest version. Refresh the page and try again.',
      );
    }
  }

  #registerUpdateCheckTriggers(): void {
    const windowRef = this.#document.defaultView;

    if (!windowRef) {
      return;
    }

    windowRef.addEventListener('online', this.#handleOnline);
    this.#document.addEventListener('visibilitychange', this.#handleVisibilityChange);

    this.#destroyRef.onDestroy(() => {
      windowRef.removeEventListener('online', this.#handleOnline);
      this.#document.removeEventListener('visibilitychange', this.#handleVisibilityChange);
    });
  }

  readonly #handleOnline = (): void => {
    void this.#checkForUpdates();
  };

  readonly #handleVisibilityChange = (): void => {
    if (this.#document.visibilityState !== 'visible') {
      return;
    }

    void this.#checkForUpdates();
  };

  #handleVersionEvent(event: VersionEvent): void {
    switch (event.type) {
      case 'VERSION_DETECTED':
        console.info(`Downloading new app version: ${event.version.hash}`);
        this.#statusMessage.set('A new version was detected and is being downloaded.');
        break;
      case 'VERSION_READY':
        console.info(`Current app version: ${event.currentVersion.hash}`);
        console.info(`New app version ready for use: ${event.latestVersion.hash}`);
        this.#updateAvailable.set(true);
        this.#statusMessage.set('A new version is ready. Reload the app to apply it.');
        break;
      case 'VERSION_INSTALLATION_FAILED':
        console.error(`Failed to install app version '${event.version.hash}': ${event.error}`);
        this.#statusMessage.set('A new version was found, but the download failed.');
        break;
      case 'NO_NEW_VERSION_DETECTED':
        console.info('No new app version detected');
        this.#statusMessage.set('No new version is available right now.');
        break;
    }
  }
}
