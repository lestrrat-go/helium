<?xml version="1.0" encoding="utf-8" ?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform" xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/" xmlns:mqh="http://www.fake.com/mqh/xsd" xmlns:qri="http://www.fake.com/xsd/Request" exclude-result-prefixes="#all">
  <xsl:output method="xml" indent="yes" encoding="UTF-8"/>

  <!-- always start at the beginning -->
  <xsl:template match="/">
    <xsl:apply-templates select="soap:Envelope/soap:Body/mqh:QuoteRetrieval/*:Request"/>
  </xsl:template>

  <!-- root of output -->
  <xsl:template match="*:Request">
    <xsl:element name="Request" namespace="http://www.fake.com/xsd/Request">
      <xsl:copy-of select="*" copy-namespaces="no"/>
    </xsl:element>
  </xsl:template>

</xsl:stylesheet>
