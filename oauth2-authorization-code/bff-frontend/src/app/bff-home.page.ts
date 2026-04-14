import { ChangeDetectionStrategy, Component, OnInit, inject } from '@angular/core';
import { JsonPipe } from '@angular/common';

import { environment } from '../environments/environment';
import { BffBrowserSessionService } from './bff-session.service';

@Component({
  selector: 'app-bff-home-page',
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [JsonPipe],
  templateUrl: './bff-home.page.html',
  styleUrl: './bff-home.page.css',
})
export class BffBrowserSessionPage implements OnInit {
  protected readonly browserSession = inject(BffBrowserSessionService);
  protected readonly backendBaseUrl = environment.backendBaseUrl;

  async ngOnInit(): Promise<void> {
    await this.browserSession.loadSession();
  }

  logout(): void {
    void this.browserSession.logout();
  }

  endProviderSession(): void {
    const logoutUrl = new URL(environment.providerLogoutUrl);
    logoutUrl.searchParams.set('post_logout_redirect_uri', window.location.origin);
    window.location.assign(logoutUrl.toString());
  }
}
