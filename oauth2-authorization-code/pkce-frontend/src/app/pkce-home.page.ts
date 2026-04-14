import { ChangeDetectionStrategy, Component, OnInit, inject } from '@angular/core';
import { JsonPipe } from '@angular/common';

import { environment } from '../environments/environment';
import { PkceBrowserFlowService } from './pkce-auth.service';

@Component({
  selector: 'app-pkce-home-page',
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [JsonPipe],
  templateUrl: './pkce-home.page.html',
  styleUrl: './pkce-home.page.css',
})
export class PkceBrowserFlowPage implements OnInit {
  protected readonly browserFlow = inject(PkceBrowserFlowService);
  protected readonly providerBaseUrl = environment.providerBaseUrl;
  protected readonly clientId = environment.clientId;
  protected readonly redirectUri = environment.redirectUri;

  async ngOnInit(): Promise<void> {
    await this.browserFlow.initializeAuth();
  }

  login(user: 'alice' | 'bob'): void {
    void this.browserFlow.beginLogin(user);
  }

  logout(): void {
    void this.browserFlow.logout();
  }

  endProviderSession(): void {
    const logoutUrl = new URL(environment.providerLogoutUrl);
    logoutUrl.searchParams.set('post_logout_redirect_uri', window.location.origin);
    window.location.assign(logoutUrl.toString());
  }
}
