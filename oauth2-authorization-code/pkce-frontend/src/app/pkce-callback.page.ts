import { ChangeDetectionStrategy, Component, OnInit, inject, signal } from '@angular/core';
import { RouterLink } from '@angular/router';
import { Router } from '@angular/router';

import { PkceBrowserFlowService } from './pkce-auth.service';

@Component({
  selector: 'app-pkce-callback-page',
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [RouterLink],
  templateUrl: './pkce-callback.page.html',
  styleUrl: './pkce-callback.page.css',
})
export class PkceBrowserCallbackPage implements OnInit {
  protected readonly browserFlow = inject(PkceBrowserFlowService);
  private readonly router = inject(Router);
  protected readonly status = signal('Completing the OAuth code flow with angular-oauth2-oidc...');

  async ngOnInit(): Promise<void> {
    await this.browserFlow.initializeAuth();

    if (this.browserFlow.error()) {
      this.status.set('The callback failed. Inspect the error banner and browser storage state.');
      return;
    }

    this.status.set('Login complete. Redirecting back to the PKCE dashboard...');
    await this.router.navigateByUrl('/');
  }
}
