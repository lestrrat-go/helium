<?xml version="1.0"?>

<!-- test use of xsl:attribute with a namespace prefix not known until run-time -->

<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xsd="http://www.w3.org/2001/XMLSchema" xmlns:d7p1="http://schemas.xmlsoap.org/wsdl/">

  <xsl:template match="xsd:attribute">
    <xsl:copy>
      <xsl:for-each select="@*">
        <xsl:variable name="name" select="name()"/>
        <xsl:variable name="val" select="."/>
        <xsl:attribute name="{$name}">xxx</xsl:attribute>
      </xsl:for-each>
    </xsl:copy>
  </xsl:template>
  
  <xsl:template match="text()"/>

</xsl:stylesheet>
