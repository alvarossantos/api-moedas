document.addEventListener('DOMContentLoaded', () => {
    const fromSelect = document.getElementById('from');
    const toSelect = document.getElementById('to');
    const baseCurrencySelect = document.getElementById('base-currency');
    const convertBtn = document.getElementById('convert-btn');
    const swapBtn = document.getElementById('swap-btn');
    const amountInput = document.getElementById('amount');
    const amountWrapper = document.getElementById('amount-wrapper'); 
    const feedbackMsg = document.getElementById('input-feedback');   
    const resultDiv = document.getElementById('result');
    const convertedValueP = document.getElementById('converted-value');
    const errorDiv = document.getElementById('error');
    const ratesTableBody = document.querySelector('#rates-table tbody');
    const ratesErrorDiv = document.getElementById('rates-error');

    const API_URL = ''; 
    const popularCurrencies = ['USD', 'EUR', 'BRL', 'JPY', 'GBP'];

    // --- Helper para limpar numero Brasileiro ---
    // Transforma "1.500,99" em "1500.99" (formato aceito pelo Go/JS)
    function parseBrazilianNumber(value) {
        if (!value) return NaN;
        // 1. Remove todos os pontos (separador de milhar)
        // 2. Troca a vírgula por ponto (decimal)
        const cleanValue = value.replace(/\./g, '').replace(',', '.');
        return parseFloat(cleanValue);
    }

    // --- 1. Validação ---
    function validateInput() {
        const value = amountInput.value;
        
        feedbackMsg.textContent = '';
        feedbackMsg.classList.remove('error-text');
        amountWrapper.classList.remove('invalid');
        convertBtn.disabled = false;
        hideMessages();

        if (value === '') return;

        // Aceita números, pontos e vírgulas
        const hasInvalidChars = /[^0-9.,]/.test(value);

        // Usa nossa função helper para checar o valor real
        const numberValue = parseBrazilianNumber(value);
        const isNegative = numberValue < 0;

        if (hasInvalidChars) {
            showInputError("⚠️ Apenas números são permitidos.");
        } else if (isNegative) {
            showInputError("⚠️ Não são permitidos valores negativos.");
        }
    }

    function showInputError(msg) {
        feedbackMsg.textContent = msg;
        feedbackMsg.classList.add('error-text');
        amountWrapper.classList.add('invalid');
        convertBtn.disabled = true;
    }

    // --- 2. Popula Moedas ---
    async function fetchAndPopulateCurrencies() {
        try {
            const response = await fetch(`${API_URL}/rates?base=USD`);
            if (!response.ok) throw new Error('Não foi possível carregar as moedas.');
            
            const data = await response.json();
            if (!data.rates) throw new Error('Formato de resposta inválido');
            
            const currencies = Object.keys(data.rates).sort();
            const selects = [fromSelect, toSelect, baseCurrencySelect];
            
            selects.forEach(select => {
                if (!select) return;
                select.innerHTML = ''; 

                const defaultOption = new Option('Selecione...', '', true, true);
                defaultOption.disabled = true;
                select.add(defaultOption);

                popularCurrencies.forEach(currency => {
                    if (currencies.includes(currency)) select.add(new Option(currency, currency));
                });

                select.add(new Option('──────────', '', false, false));
                select.options[select.options.length - 1].disabled = true;

                currencies.forEach(currency => {
                    if (!popularCurrencies.includes(currency)) select.add(new Option(currency, currency));
                });
            });

        } catch (error) {
            console.error(error);
            showError("Erro ao conectar com o Backend Go.", errorDiv);
        }
    }

    // --- 3. Conversão ---
    async function convertCurrency() {
        const rawAmount = amountInput.value; 
        const from = fromSelect.value;
        const to = toSelect.value;

        // CORREÇÃO AQUI: Usa a função que remove os pontos de milhar antes de enviar
        const amountForServer = parseBrazilianNumber(rawAmount);

        if (isNaN(amountForServer) || amountForServer <= 0) {
            validateInput(); // Mostra erro se não for numero válido
            return;
        }

        convertBtn.disabled = true;
        convertBtn.innerHTML = '<i class="fas fa-circle-notch fa-spin"></i> Processando...';

        try {
            // Envia "1500.99" limpo para o servidor
            const response = await fetch(`${API_URL}/convert?from=${from}&to=${to}&amount=${amountForServer}`);
            const data = await response.json();

            if (!response.ok) throw new Error(data.error || 'Erro na conversão.');

            const inputFormatted = amountForServer.toLocaleString('pt-BR', { minimumFractionDigits: 2 });

            const resultFormatted = data.result.toLocaleString('pt-BR', { minimumFractionDigits: 2, maximumFractionDigits: 4 });
            
            const impliedRate = data.result / amountForServer;

            const rateFormatted = impliedRate.toLocaleString('pt-BR', { minimumFractionDigits: 2, maximumFractionDigits: 6});
            
            convertedValueP.innerHTML = `
                <div style="margin-bottom: 5px;">
                    <span style="color: #94a3b8; font-size: 0.8em;">${inputFormatted} ${from} =</span> 
                    ${resultFormatted} <small>${to}</small>
                </div>
                <div style="font-size: 0.75rem; color: #6366f1; font-weight: 500; opacity: 0.8;">
                    <i class="fas fa-info-circle"></i> Cotação: 1 ${from} = ${rateFormatted} ${to}
                </div>
            `;
            
            resultDiv.classList.remove('hidden');

        } catch (error) {
            showError(error.message, errorDiv);
        } finally {
            convertBtn.disabled = false;
            convertBtn.innerHTML = 'Converter Agora';
        }
    }

    // --- 4. Taxas ---
    async function fetchRates() {
        const base = baseCurrencySelect.value;
        if (!base) return;
        ratesTableBody.innerHTML = '<tr><td colspan="2" style="text-align:center">Carregando...</td></tr>';

        try {
            const response = await fetch(`${API_URL}/rates?base=${base}`);
            const data = await response.json();
            if (!response.ok) throw new Error('Erro ao buscar taxas.');

            ratesTableBody.innerHTML = '';
            const targetCurrencies = ['USD', 'EUR', 'BRL', 'GBP', 'JPY', 'CAD', 'AUD', 'ARS'];
            
            targetCurrencies.forEach(currency => {
                if (currency !== base && data.rates[currency]) {
                    const row = ratesTableBody.insertRow();
                    const rateFormatted = data.rates[currency].toFixed(4).replace('.', ',');
                    row.innerHTML = `<td>${currency}</td><td>${rateFormatted}</td>`;
                }
            });

        } catch (error) {
            ratesTableBody.innerHTML = '';
        }
    }

    // Helpers
    function swapCurrencies() {
        const temp = fromSelect.value;
        fromSelect.value = toSelect.value;
        toSelect.value = temp;
        resultDiv.classList.add('hidden');
    }
    function showError(msg, element) {
        element.textContent = msg;
        element.classList.remove('hidden');
    }
    function hideMessages() {
        errorDiv.classList.add('hidden');
        resultDiv.classList.add('hidden');
    }

    // Listeners
    convertBtn.addEventListener('click', convertCurrency);
    swapBtn.addEventListener('click', swapCurrencies);
    baseCurrencySelect.addEventListener('change', fetchRates);
    amountInput.addEventListener('input', validateInput);
    fromSelect.addEventListener('change', hideMessages);
    toSelect.addEventListener('change', hideMessages);

    fetchAndPopulateCurrencies();
});