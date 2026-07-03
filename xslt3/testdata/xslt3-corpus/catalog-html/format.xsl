<?xml version="1.0" encoding="UTF-8"?>
<!-- Included helper module: a namespaced xsl:function used by the main module.
     Loaded via xsl:include, so the compile-time URIResolver must resolve
     "format.xsl" against the base URI. -->
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:fmt="urn:helium:corpus:format"
    exclude-result-prefixes="xs fmt">

  <xsl:function name="fmt:money" as="xs:string">
    <xsl:param name="amount" as="xs:decimal"/>
    <xsl:param name="currency" as="xs:string"/>
    <xsl:sequence select="concat($currency, ' ', format-number($amount, '#,##0.00'))"/>
  </xsl:function>

</xsl:stylesheet>
