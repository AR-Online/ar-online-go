package aronline_test

import (
	"context"
	"net/http"
	"testing"
)

func TestSendingProofDecodificaOBase64EDeixaOCruAcessivel(t *testing.T) {
	fake := newFakeGateway(t)
	// "JVBERi0x" é "%PDF-1" — o começo de qualquer PDF de verdade.
	fake.answers(t, map[string]any{"content": "JVBERi0x"})

	proof, err := fake.client().Legacy.SendingProof(context.Background(), "f6cb58f2")
	if err != nil {
		t.Fatalf("não esperava erro: %v", err)
	}

	if string(proof.PDF) != "%PDF-1" {
		t.Errorf("PDF = %q, queria os bytes decodificados", proof.PDF)
	}

	if proof.ContentBase64 != "JVBERi0x" {
		t.Errorf("ContentBase64 = %q", proof.ContentBase64)
	}

	if proof.Message != "" {
		t.Errorf("Message = %q, queria vazia quando o comprovante veio", proof.Message)
	}

	if fake.received.Path != "/gw/sending-proof/f6cb58f2" {
		t.Errorf("caminho = %q", fake.received.Path)
	}
}

func TestSendingProofSemStatusDeEntregaVemMensagemENaoEhErro(t *testing.T) {
	fake := newFakeGateway(t)
	fake.answers(t, map[string]any{
		"message": "O comprovante para e-mail consultado ainda não possui o status de entrega",
	})

	proof, err := fake.client().Legacy.SendingProof(context.Background(), "f6cb58f2")
	// A frase do gateway não é falha: é "pergunte de novo mais tarde".
	if err != nil {
		t.Fatalf("a mensagem não é erro: %v", err)
	}

	if proof.PDF != nil || proof.ContentBase64 != "" {
		t.Errorf("veio PDF onde só havia mensagem: %+v", proof)
	}

	if !stringsContains(proof.Message, "ainda não possui o status") {
		t.Errorf("Message = %q", proof.Message)
	}
}

func TestSendingProofCorpoVazioResolveComTudoVazio(t *testing.T) {
	fake := newFakeGateway(t)
	fake.answers(t, map[string]any{})

	proof, err := fake.client().Legacy.SendingProof(context.Background(), "f6cb58f2")
	if err != nil {
		t.Fatalf("não esperava erro: %v", err)
	}

	if proof.PDF != nil || proof.ContentBase64 != "" || proof.Message != "" {
		t.Errorf("proof = %+v, queria os três vazios", proof)
	}
}

func TestSendingProofBase64IlegivelViraErroDoSDK(t *testing.T) {
	fake := newFakeGateway(t)
	fake.answers(t, map[string]any{"content": "%%%não-é-base64%%%"})

	_, err := fake.client().Legacy.SendingProof(context.Background(), "f6cb58f2")
	failure := legacyError(t, err)

	if !stringsContains(failure.Message, "base64") {
		t.Errorf("mensagem = %q", failure.Message)
	}

	// A exceção crua do decodificador continua embaixo, mas não vaza sozinha.
	if failure.Unwrap() == nil {
		t.Error("a causa tem de continuar alcançável")
	}
}

func TestSendingProofPropagaARecusa(t *testing.T) {
	fake := newFakeGateway(t)
	fake.refuses(t, http.StatusNotFound, map[string]any{"message": "E-mail não encontrado"})

	_, err := fake.client().Legacy.SendingProof(context.Background(), "sumido")

	if got := legacyError(t, err).Status; got != http.StatusNotFound {
		t.Errorf("status = %d, queria 404", got)
	}
}

func TestLaudoEntregaOsBytesDoPDFBinario(t *testing.T) {
	fake := newFakeGateway(t)
	// Esta rota é a única que responde o arquivo direto: nada de base64, nada
	// de JSON.
	fake.answersRaw(http.StatusOK, "application/pdf", "%PDF-1.4 laudo")

	bytes, err := fake.client().Legacy.Laudo(context.Background(), "f6cb58f2")
	if err != nil {
		t.Fatalf("não esperava erro: %v", err)
	}

	if string(bytes) != "%PDF-1.4 laudo" {
		t.Errorf("laudo = %q", bytes)
	}

	if fake.received.Path != "/gw/email/laudo/f6cb58f2" {
		t.Errorf("caminho = %q", fake.received.Path)
	}
}

func TestLaudoDeRegistroInexistenteResponde404EmJSON(t *testing.T) {
	fake := newFakeGateway(t)
	fake.refuses(t, http.StatusNotFound, map[string]any{
		"statusCode": 404, "message": "Registro não encontrado",
	})

	_, err := fake.client().Legacy.Laudo(context.Background(), "sumido")
	failure := legacyError(t, err)

	if failure.Status != http.StatusNotFound {
		t.Errorf("status = %d, queria 404", failure.Status)
	}

	if failure.Message != "Registro não encontrado" {
		t.Errorf("mensagem = %q", failure.Message)
	}
}

func TestFinalizarReguaEhGETComEfeitoColateral(t *testing.T) {
	fake := newFakeGateway(t)
	fake.answers(t, map[string]any{"message": "Regua de notificação finalizada com sucesso"})

	result, err := fake.client().Legacy.FinalizarRegua(context.Background(), "f6cb58f2")
	if err != nil {
		t.Fatalf("não esperava erro: %v", err)
	}

	if result.Message != "Regua de notificação finalizada com sucesso" {
		t.Errorf("mensagem = %q", result.Message)
	}

	// GET com efeito colateral é o contrato, e o SDK não "conserta" para POST:
	// consertar aqui seria uma chamada que funciona no curl e não aqui.
	if fake.received.Method != http.MethodGet {
		t.Errorf("método = %q, queria GET", fake.received.Method)
	}

	if fake.received.Path != "/regua-notificacao/finalizar/f6cb58f2" {
		t.Errorf("caminho = %q", fake.received.Path)
	}
}

func TestFinalizarReguaPropagaARecusa(t *testing.T) {
	fake := newFakeGateway(t)
	fake.refuses(t, http.StatusNotFound, map[string]any{"message": "E-mail não encontrado"})

	_, err := fake.client().Legacy.FinalizarRegua(context.Background(), "sumido")

	if got := legacyError(t, err).Status; got != http.StatusNotFound {
		t.Errorf("status = %d, queria 404", got)
	}
}
