package main

import (
	"bufio"
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"encoding/csv"
	"epdg_ue/pkg/ike/context"
	"epdg_ue/pkg/ike/handler"
	"epdg_ue/pkg/ike/message"
	"errors"
	"fmt"
	"hash"
	"log"
	"math/big"
	"math/rand"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

var (
	UeAddr     = "192.168.122.1"
	Ikev2PortI = ":44620"
	Ikev2PortR = ":500"
)

func main() {
	// Read Scan Result File
	file, err := os.Open("scan-results.json")
	checkError(err)
	defer file.Close()

	// Read file by lines and get IPv4 addresses
	var epdgs []net.IP
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		line = strings.ReplaceAll(line, "\"", "")
		line = strings.ReplaceAll(line, ": {", "")
		line = strings.ReplaceAll(line, " ", "")

		ip := net.ParseIP(line)
		if ip != nil && ip.To4() != nil {
			epdgs = append(epdgs, ip)
		}
	}

	fmt.Println("Got Ip Addresses of", len(epdgs), "EPDGs")

	var mapEpdgReply map[string]string
	mapEpdgReply = make(map[string]string)

	ueUDPAddr, err := net.ResolveUDPAddr("udp", UeAddr+Ikev2PortI)
	checkError(err)

	output, err := os.Create("output.csv")
	checkError(err)
	defer output.Close()

	writer := csv.NewWriter(output)

	for _, epdg := range epdgs {
		EpdgAddr := epdg.String()

		epdgUDPAddr, err := net.ResolveUDPAddr("udp", EpdgAddr+Ikev2PortR)
		checkError(err)

		udpConnection, err := net.ListenUDP("udp", ueUDPAddr)
		checkError(err)

		// Forge IKE_SA_INIT Initiator Request
		ikeInitiatorSPI := uint64(rand.Uint32())<<32 + uint64(rand.Uint32())
		ikeMessage := new(message.IKEMessage)
		ikeMessage.BuildIKEHeader(ikeInitiatorSPI, 0, message.IKE_SA_INIT, message.InitiatorBitCheck, 0)

		securityAssociation := ikeMessage.Payloads.BuildSecurityAssociation()
		proposal := securityAssociation.Proposals.BuildProposal(1, message.TypeIKE, nil)

		// Encryption
		var attributeType uint16 = message.AttributeTypeKeyLength
		var keyLength256 uint16 = 256
		var keyLength192 uint16 = 192
		var keyLength128 uint16 = 128
		proposal.EncryptionAlgorithm.BuildTransform(message.TypeEncryptionAlgorithm, message.ENCR_AES_CBC, &attributeType, &keyLength256, nil)
		proposal.EncryptionAlgorithm.BuildTransform(message.TypeEncryptionAlgorithm, message.ENCR_AES_CBC, &attributeType, &keyLength192, nil)
		proposal.EncryptionAlgorithm.BuildTransform(message.TypeEncryptionAlgorithm, message.ENCR_AES_CBC, &attributeType, &keyLength128, nil)
		proposal.EncryptionAlgorithm.BuildTransform(message.TypeEncryptionAlgorithm, message.ENCR_AES_CTR, &attributeType, &keyLength256, nil)
		proposal.EncryptionAlgorithm.BuildTransform(message.TypeEncryptionAlgorithm, message.ENCR_AES_CTR, &attributeType, &keyLength192, nil)
		proposal.EncryptionAlgorithm.BuildTransform(message.TypeEncryptionAlgorithm, message.ENCR_AES_CTR, &attributeType, &keyLength128, nil)
		proposal.EncryptionAlgorithm.BuildTransform(message.TypeEncryptionAlgorithm, message.ENCR_3DES, nil, nil, nil)
		proposal.EncryptionAlgorithm.BuildTransform(message.TypeEncryptionAlgorithm, message.ENCR_DES, nil, nil, nil)

		// Integrity
		proposal.IntegrityAlgorithm.BuildTransform(message.TypeIntegrityAlgorithm, message.AUTH_HMAC_SHA1_96, nil, nil, nil)
		proposal.IntegrityAlgorithm.BuildTransform(message.TypeIntegrityAlgorithm, message.AUTH_HMAC_MD5_96, nil, nil, nil)
		proposal.IntegrityAlgorithm.BuildTransform(message.TypeIntegrityAlgorithm, message.AUTH_HMAC_SHA2_256_128, nil, nil, nil)
		proposal.IntegrityAlgorithm.BuildTransform(message.TypeIntegrityAlgorithm, message.AUTH_HMAC_SHA2_512_256, nil, nil, nil)

		// PRF
		proposal.PseudorandomFunction.BuildTransform(message.TypePseudorandomFunction, message.PRF_HMAC_MD5, nil, nil, nil)
		proposal.PseudorandomFunction.BuildTransform(message.TypePseudorandomFunction, message.PRF_HMAC_SHA1, nil, nil, nil)
		proposal.PseudorandomFunction.BuildTransform(message.TypePseudorandomFunction, message.PRF_HMAC_SHA2_256, nil, nil, nil)
		proposal.PseudorandomFunction.BuildTransform(message.TypePseudorandomFunction, message.PRF_HMAC_SHA2_384, nil, nil, nil)
		proposal.PseudorandomFunction.BuildTransform(message.TypePseudorandomFunction, message.PRF_HMAC_SHA2_512, nil, nil, nil)

		// DH
		//proposal.DiffieHellmanGroup.BuildTransform(message.TypeDiffieHellmanGroup, message.DH_768_BIT_MODP, nil, nil, nil)
		//proposal.DiffieHellmanGroup.BuildTransform(message.TypeDiffieHellmanGroup, message.DH_1024_BIT_MODP, nil, nil, nil)
		//proposal.DiffieHellmanGroup.BuildTransform(message.TypeDiffieHellmanGroup, message.DH_1536_BIT_MODP, nil, nil, nil)
		//proposal.DiffieHellmanGroup.BuildTransform(message.TypeDiffieHellmanGroup, message.DH_2048_BIT_MODP, nil, nil, nil)
		//proposal.DiffieHellmanGroup.BuildTransform(message.TypeDiffieHellmanGroup, message.DH_3072_BIT_MODP, nil, nil, nil)
		//proposal.DiffieHellmanGroup.BuildTransform(message.TypeDiffieHellmanGroup, message.DH_4096_BIT_MODP, nil, nil, nil)
		//proposal.DiffieHellmanGroup.BuildTransform(message.TypeDiffieHellmanGroup, message.DH_6144_BIT_MODP, nil, nil, nil)
		proposal.DiffieHellmanGroup.BuildTransform(message.TypeDiffieHellmanGroup, message.DH_8192_BIT_MODP, nil, nil, nil)

		// Key exchange data
		generator := new(big.Int).SetUint64(handler.Group18Generator)
		factor, ok := new(big.Int).SetString(handler.Group18PrimeString, 16)
		if !ok {
			fmt.Errorf("Generate key exchange data failed")
		}
		secret := handler.GenerateRandomNumber()
		localPublicKeyExchangeValue := new(big.Int).Exp(generator, secret, factor).Bytes()
		prependZero := make([]byte, len(factor.Bytes())-len(localPublicKeyExchangeValue))
		localPublicKeyExchangeValue = append(prependZero, localPublicKeyExchangeValue...)
		ikeMessage.Payloads.BUildKeyExchange(message.DH_8192_BIT_MODP, localPublicKeyExchangeValue)

		// Nonce
		localNonce := handler.GenerateRandomNumber().Bytes()
		ikeMessage.Payloads.BuildNonce(localNonce)

		// Add NAT Detection Payloads (Source, Destination) for NAT-T
		spiI := make([]byte, 8)
		spiR := make([]byte, 8)
		ipi := net.ParseIP(UeAddr)
		porti := ueUDPAddr.Port

		// Source IP hash = SHA-1( SPIi|| SPIr || IPi || Porti)
		binary.BigEndian.PutUint64(spiI, ikeInitiatorSPI)
		localDetectionData := make([]byte, 22)
		copy(localDetectionData[0:8], spiI)
		copy(localDetectionData[8:16], spiR)
		copy(localDetectionData[16:20], ipi.To4())
		binary.BigEndian.PutUint16(localDetectionData[20:22], uint16(porti))

		sha1HashFunction := sha1.New()
		_, err = sha1HashFunction.Write(localDetectionData)
		if err != nil {
			panic(err)
		}
		notificationData := sha1HashFunction.Sum(nil)

		ikeMessage.Payloads.BuildNotification(0, message.NAT_DETECTION_SOURCE_IP, nil, notificationData)

		// Destination IP hash = SHA-1( SPIi || SPIr || IPr || Portr)
		ipr := epdgUDPAddr.IP
		portr := epdgUDPAddr.Port

		localDetectionData = make([]byte, 22)
		copy(localDetectionData[0:8], spiI)
		copy(localDetectionData[8:16], spiR)
		copy(localDetectionData[16:20], ipr.To4())
		binary.BigEndian.PutUint16(localDetectionData[20:22], uint16(portr))

		sha1HashFunction = sha1.New()
		_, err = sha1HashFunction.Write(localDetectionData)
		checkError(err)

		notificationData = sha1HashFunction.Sum(nil)

		ikeMessage.Payloads.BuildNotification(0, message.NAT_DETECTION_DESTINATION_IP, nil, notificationData)

		// Send IKE_SA_INIT to EPDG
		ikeMessageData, err := ikeMessage.Encode()
		checkError(err)
		_, err = udpConnection.WriteToUDP(ikeMessageData, epdgUDPAddr)
		checkError(err)

		log.Println("IKE_SA_INIT sent to EPDG at", epdgUDPAddr)

		// Receive IKE_SA_INIT reply from EPDG
		err = udpConnection.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		checkError(err)
		buffer := make([]byte, 65535)
		n, ikeSenderAddr, err := udpConnection.ReadFromUDP(buffer)
		if err != nil {
			fmt.Errorf("Read IKE Message fail: %+v", err)
		}
		ikeMessage.Payloads.Reset()
		err = ikeMessage.Decode(buffer[:n])
		if err != nil {
			fmt.Errorf("Decode IKE Message fail: %+v", err)
		}

		var sharedKeyExchangeData []byte
		var remoteNonce []byte
		var ueIsBehindNAT, epdgIsBehindNAT bool

		for _, ikePayload := range ikeMessage.Payloads {
			switch ikePayload.Type() {
			case message.TypeSA:
				fmt.Println("Received IKE SA Payload. IKE_SA_INIT Exchange done.")
				mapEpdgReply[epdgUDPAddr.String()] = "proposal-sa-accepted"
			case message.TypeKE:
				remotePublicKeyExchangeValue := ikePayload.(*message.KeyExchange).KeyExchangeData
				var i int = 0
				for {
					if remotePublicKeyExchangeValue[i] != 0 {
						break
					}
				}
				remotePublicKeyExchangeValue = remotePublicKeyExchangeValue[i:]
				remotePublicKeyExchangeValueBig := new(big.Int).SetBytes(remotePublicKeyExchangeValue)
				sharedKeyExchangeData = new(big.Int).Exp(remotePublicKeyExchangeValueBig, secret, factor).Bytes()
			case message.TypeNiNr:
				remoteNonce = ikePayload.(*message.Nonce).NonceData
			case message.TypeN:
				notification := ikePayload.(*message.Notification)
				fmt.Println("Received Notify Payload. Parsing...")
				if notification.NotifyMessageType == message.NAT_DETECTION_SOURCE_IP {
					fmt.Println("Received NAT-Detection Source IP Payload. Computing Detection Data.")
					// Check received Source hash = SHA-1( SPIi || SPIr || IPr || Portr)
					localDetectionData = make([]byte, 22)
					binary.BigEndian.PutUint64(localDetectionData[0:8], ikeMessage.InitiatorSPI)
					binary.BigEndian.PutUint64(localDetectionData[8:16], ikeMessage.ResponderSPI)
					copy(localDetectionData[16:20], ikeSenderAddr.IP.To4())
					binary.BigEndian.PutUint16(localDetectionData[20:22], uint16(ikeSenderAddr.Port))

					sha1HashFunction = sha1.New()
					_, err = sha1HashFunction.Write(localDetectionData)
					checkError(err)

					if !bytes.Equal(notification.NotificationData, sha1HashFunction.Sum(nil)) {
						fmt.Println("EPDG is Behind NAT. Switch to UDP Port 4500 for IKEv2.")
						epdgIsBehindNAT = true
					}

				} else if notification.NotifyMessageType == message.NAT_DETECTION_DESTINATION_IP {
					fmt.Println("Received NAT-Detection Destination IP Payload. Computing Detection Data.")
					// Check received Destination hash = SHA-1( SPIi || SPIr || IPi || Porti)
					localDetectionData = make([]byte, 22)
					binary.BigEndian.PutUint64(localDetectionData[0:8], ikeMessage.InitiatorSPI)
					binary.BigEndian.PutUint64(localDetectionData[8:16], ikeMessage.ResponderSPI)
					copy(localDetectionData[16:20], ipi.To4())
					binary.BigEndian.PutUint16(localDetectionData[20:22], uint16(porti))

					sha1HashFunction = sha1.New()
					_, err = sha1HashFunction.Write(localDetectionData)
					checkError(err)

					if !bytes.Equal(notification.NotificationData, sha1HashFunction.Sum(nil)) {
						fmt.Println("UE is Behind NAT. Switch to UDP Port 4500 for IKEv2.")
						ueIsBehindNAT = true
					}
				} else if notification.NotifyMessageType == message.COOKIE {
					fmt.Println("Received Cookie Payload.")
					mapEpdgReply[epdgUDPAddr.String()] = "cookie"
					break
				} else if notification.NotifyMessageType == message.NO_PROPOSAL_CHOSEN {
					fmt.Println("Received No Proposal Chosen. Change Proposal")
					mapEpdgReply[epdgUDPAddr.String()] = "no-proposal-chosen"
					break
				} else if notification.NotifyMessageType == message.INVALID_SYNTAX {
					fmt.Println("Received Invalid Syntax Payload.")
					mapEpdgReply[epdgUDPAddr.String()] = "invalid-syntax"
					break
				} else if notification.NotifyMessageType == message.INVALID_KE_PAYLOAD {
					fmt.Println("Received Invalid KE Payload. Try to change Proposal")
					mapEpdgReply[epdgUDPAddr.String()] = "invalid-ke-payload"
					break
				} else {
					fmt.Println("Received Notify Payload for Error Reporting. Value:", notification.NotifyMessageType)
					mapEpdgReply[epdgUDPAddr.String()] = "private-use-error-reporting-" + strconv.Itoa(int(notification.NotifyMessageType))
				}
			}
		}

		ikeSecurityAssociation := &context.IKESecurityAssociation{
			LocalSPI:               ikeInitiatorSPI,
			RemoteSPI:              ikeMessage.ResponderSPI,
			InitiatorMessageID:     0,
			ResponderMessageID:     0,
			EncryptionAlgorithm:    proposal.EncryptionAlgorithm[0],
			IntegrityAlgorithm:     proposal.IntegrityAlgorithm[0],
			PseudorandomFunction:   proposal.PseudorandomFunction[0],
			DiffieHellmanGroup:     proposal.DiffieHellmanGroup[0],
			ConcatenatedNonce:      append(localNonce, remoteNonce...),
			DiffieHellmanSharedKey: sharedKeyExchangeData,
			UEIsBehindNAT:          ueIsBehindNAT,
			EPDGIsBehindNAT:        epdgIsBehindNAT,
		}

		if err := generateKeyForIKESA(ikeSecurityAssociation); err != nil {
			fmt.Errorf("Generate key for IKE SA failed: %+v", err)
		}

		// Scrittura dei dati dalla mappa al file CSV
		err = writer.Write([]string{epdgUDPAddr.String(), mapEpdgReply[epdgUDPAddr.String()]})
		checkError(err)
		writer.Flush()

		udpConnection.Close()
	}
}

func checkError(err error) {
	// Panics if err is not nil
	if err != nil {
		panic(err)
	}
}

func generateKeyForIKESA(ikeSecurityAssociation *context.IKESecurityAssociation) error {
	// Transforms
	transformPseudorandomFunction := ikeSecurityAssociation.PseudorandomFunction

	// Get key length of SK_d, SK_ai, SK_ar, SK_ei, SK_er, SK_pi, SK_pr
	var length_SK_d, length_SK_ai, length_SK_ar, length_SK_ei, length_SK_er, length_SK_pi, length_SK_pr, totalKeyLength int
	var ok bool

	length_SK_d = 20
	length_SK_ai = 20
	length_SK_ar = length_SK_ai
	length_SK_ei = 32
	length_SK_er = length_SK_ei
	length_SK_pi, length_SK_pr = length_SK_d, length_SK_d
	totalKeyLength = length_SK_d + length_SK_ai + length_SK_ar + length_SK_ei + length_SK_er + length_SK_pi + length_SK_pr

	// Generate IKE SA key as defined in RFC7296 Section 1.3 and Section 1.4
	var pseudorandomFunction hash.Hash

	if pseudorandomFunction, ok = handler.NewPseudorandomFunction(ikeSecurityAssociation.ConcatenatedNonce, transformPseudorandomFunction.TransformID); !ok {
		return errors.New("New pseudorandom function failed")
	}

	if _, err := pseudorandomFunction.Write(ikeSecurityAssociation.DiffieHellmanSharedKey); err != nil {
		return errors.New("Pseudorandom function write failed")
	}

	SKEYSEED := pseudorandomFunction.Sum(nil)

	seed := concatenateNonceAndSPI(ikeSecurityAssociation.ConcatenatedNonce, ikeSecurityAssociation.LocalSPI, ikeSecurityAssociation.RemoteSPI)

	var keyStream, generatedKeyBlock []byte
	var index byte
	for index = 1; len(keyStream) < totalKeyLength; index++ {
		if pseudorandomFunction, ok = handler.NewPseudorandomFunction(SKEYSEED, transformPseudorandomFunction.TransformID); !ok {
			return errors.New("New pseudorandom function failed")
		}
		if _, err := pseudorandomFunction.Write(append(append(generatedKeyBlock, seed...), index)); err != nil {
			return errors.New("Pseudorandom function write failed")
		}
		generatedKeyBlock = pseudorandomFunction.Sum(nil)
		keyStream = append(keyStream, generatedKeyBlock...)
	}

	// Assign keys into context
	ikeSecurityAssociation.SK_d = keyStream[:length_SK_d]
	keyStream = keyStream[length_SK_d:]
	ikeSecurityAssociation.SK_ai = keyStream[:length_SK_ai]
	keyStream = keyStream[length_SK_ai:]
	ikeSecurityAssociation.SK_ar = keyStream[:length_SK_ar]
	keyStream = keyStream[length_SK_ar:]
	ikeSecurityAssociation.SK_ei = keyStream[:length_SK_ei]
	keyStream = keyStream[length_SK_ei:]
	ikeSecurityAssociation.SK_er = keyStream[:length_SK_er]
	keyStream = keyStream[length_SK_er:]
	ikeSecurityAssociation.SK_pi = keyStream[:length_SK_pi]
	keyStream = keyStream[length_SK_pi:]
	ikeSecurityAssociation.SK_pr = keyStream[:length_SK_pr]
	keyStream = keyStream[length_SK_pr:]

	return nil
}

func concatenateNonceAndSPI(nonce []byte, SPI_initiator uint64, SPI_responder uint64) []byte {
	spi := make([]byte, 8)

	binary.BigEndian.PutUint64(spi, SPI_initiator)
	newSlice := append(nonce, spi...)
	binary.BigEndian.PutUint64(spi, SPI_responder)
	newSlice = append(newSlice, spi...)

	return newSlice
}
