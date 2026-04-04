import {
  ChangeDetectionStrategy,
  Component,
  ElementRef,
  effect,
  inject,
  viewChild,
} from '@angular/core';
import { AppUpdateService } from './core/services/app-update.service';
import { BUILD_INFO } from './build-info';

@Component({
  selector: 'app-root',
  changeDetection: ChangeDetectionStrategy.OnPush,
  templateUrl: './app.html',
  styleUrl: './app.css',
})
export class App {
  protected readonly appUpdate = inject(AppUpdateService);
  protected readonly buildInfo = BUILD_INFO;
  protected readonly reloadDialog = viewChild<ElementRef<HTMLDialogElement>>('reloadDialog');

  constructor() {
    effect(() => {
      const dialog = this.reloadDialog()?.nativeElement;

      if (!dialog) {
        return;
      }

      if (this.appUpdate.updateAvailable()) {
        if (!dialog.open) {
          dialog.showModal();
        }

        return;
      }

      if (dialog.open) {
        dialog.close();
      }
    });
  }
}
