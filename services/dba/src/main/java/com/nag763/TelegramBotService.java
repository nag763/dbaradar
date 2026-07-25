package com.nag763;

import java.util.List;

import org.eclipse.microprofile.config.inject.ConfigProperty;

import com.pengrad.telegrambot.TelegramBot;
import com.pengrad.telegrambot.model.Update;
import com.pengrad.telegrambot.request.GetUpdates;

import jakarta.enterprise.context.ApplicationScoped;

@ApplicationScoped
public class TelegramBotService {
    
    @ConfigProperty(name = "telegram.bot.api.key")
    private String telegramToken;

    private TelegramBot bot;

    public TelegramBotService() {
        this.bot = new TelegramBot(telegramToken);
    }

    public List<Update> getUpdates() {
        GetUpdates getUpdates = new GetUpdates().limit(100).offset(0).timeout(0);
        var response = bot.execute(getUpdates);
        return response.updates();
    }

}
