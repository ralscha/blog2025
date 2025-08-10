import { Component, effect, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import {
  translateObjectSignal,
  translateSignal,
  TranslocoDirective,
  TranslocoEvents,
  TranslocoPipe,
  TranslocoService
} from '@jsverse/transloco';
import {
  TranslocoCurrencyPipe,
  TranslocoDatePipe,
  TranslocoDecimalPipe,
  TranslocoLocaleService
} from '@jsverse/transloco-locale';
import { TranslocoMarkupComponent } from 'ngx-transloco-markup';

@Component({
  selector: 'app-home',
  imports: [
    CommonModule,
    TranslocoDatePipe,
    TranslocoCurrencyPipe,
    TranslocoDecimalPipe,
    TranslocoDirective,
    TranslocoMarkupComponent,
    TranslocoPipe
  ],
  templateUrl: './home.html',
  styleUrl: './home.css'
})
export class Home {
  private readonly translocoService = inject(TranslocoService);
  private readonly translocoLocaleService = inject(TranslocoLocaleService);
  name = 'Alex';
  gender: 'male' | 'female' | 'other' = 'male';
  unreadCount = 1;
  itemCount = 2;
  today = new Date();
  price = 1234.56;
  percent = 0.42;
  items: Array<string> = [];
  activeLang = this.translocoService.getActiveLang() || 'en';

  constructor() {
    this.translocoService
      .selectTranslate('homeTitle')
      .subscribe(translation => console.log('Home title:', translation));

    this.translocoService
      .selectTranslate('greeting', { name: this.name })
      .subscribe(translation => console.log('Greeting:', translation));

    this.translocoService
      .selectTranslateObject('fruits')
      .subscribe(translations => {
        this.items = [
          translations.apples,
          translations.bananas,
          translations.cherries
        ];
      });

    this.translocoService.langChanges$.subscribe(lang => {
      this.activeLang = lang;
      this.syncLocale(lang);
    });

    const home = translateSignal('homeTitle');
    const greeting = translateSignal('greeting', { name: this.name });
    const fruits = translateObjectSignal('fruits');

    effect(() => {
      console.log('Home title:', home());
      console.log('Greeting:', greeting());
      console.log('Fruits:', fruits()?.['apples']);
    });

    const ht = this.translocoService.translate('homeTitle');
    const g = this.translocoService.translate('greeting', { name: this.name });
    const f = this.translocoService.translateObject('fruits');
    console.log('Sync Home title:', ht);
    console.log('Sync Greeting:', g);
    console.log('Sync Fruits:', f?.['apples']);

    this.translocoService.events$.subscribe((event: TranslocoEvents) => {
      console.log('Event received:', event);
      switch (event.type) {
        case 'langChanged':
          console.log('Language changed to:', event.payload);
          break;
        case 'translationLoadFailure':
          console.log('Translation changed for:', event.payload);
          break;
        case 'translationLoadSuccess':
          console.log('Translation loaded for:', event.payload);
          const ht2 = this.translocoService.translate('homeTitle');
          console.log('Sync Home title after load:', ht2);
          break;
        default:
          console.warn('Unhandled event:', event);
      }
    });

    const availableLangs = this.translocoService.getAvailableLangs();
    console.log(availableLangs);

    this.syncLocale(this.activeLang);
  }

  setLang(lang: 'en' | 'de') {
    if (lang === this.activeLang) {
      return;
    }
    this.translocoService.setActiveLang(lang);
  }

  private syncLocale(lang: string) {
    const locale = lang === 'de' ? 'de-DE' : 'en-US';
    this.translocoLocaleService.setLocale(locale);
  }
}
