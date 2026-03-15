<?xml version="1.0"?> 
<xsl:stylesheet 
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:f="http://local-functions/"
  exclude-result-prefixes="#all"
  version="3.0">
  
  <xsl:output method="xml" encoding="UTF-8" indent="no"/>

  <!-- Purpose: d-o-e works within try/catch -->
  
  <xsl:output omit-xml-declaration="yes"/>
  
  <xsl:param name="p" select="1"/>

<xsl:template match="/">
  <out>
    <xsl:try>
      <xsl:value-of select="'&amp;lt;'" disable-output-escaping='yes'/>
      <xsl:value-of select="10 idiv $p"/>
      <xsl:catch errors="*">
        <error/>
      </xsl:catch>
    </xsl:try>
  </out>
</xsl:template>
  

</xsl:stylesheet>
