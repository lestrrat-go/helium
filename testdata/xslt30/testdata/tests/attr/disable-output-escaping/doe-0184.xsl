<?xml version="1.0"?> 
<xsl:stylesheet 
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:f="http://local-functions/"
  exclude-result-prefixes="#all"
  version="3.0">
  
  <xsl:output method="xml" encoding="UTF-8" indent="no"/>

  <!-- Purpose: d-o-e is ignored in temporary output state -->
  
  <xsl:output omit-xml-declaration="yes"/>

<xsl:template match="/">
  <out>
    <xsl:copy-of select="f:format('&lt;')"/>
    <xsl:value-of select="'&amp;lt;'" disable-output-escaping='yes'/>
  </out>
</xsl:template>
  
  <xsl:function name="f:format" as="text()">
    <xsl:param name="in" as="xs:string"/>
    <xsl:value-of select="$in" disable-output-escaping="true"/>
  </xsl:function>

</xsl:stylesheet>
