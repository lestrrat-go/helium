<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="#all"
  version="3.0">
  
  <xsl:output method="xml"/>
  
  <xsl:variable name="p" select="()"/>
  
  <xsl:template match="/" name="xsl:initial-template">
    <xsl:iterate select="$p">
      <xsl:param name="p1" as="xs:string" select="'test'"/>
      <xsl:on-completion>
        <completion>
          <xsl:copy-of select="$p1"/>
        </completion>
      </xsl:on-completion>
    </xsl:iterate>
  </xsl:template>
  
</xsl:stylesheet>