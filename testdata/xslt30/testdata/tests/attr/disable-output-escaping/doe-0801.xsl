<?xml version="1.0"?> 
<xsl:stylesheet 
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:f="http://local-functions/"
  exclude-result-prefixes="#all"
  version="3.0">
  
  <xsl:output method="xml" encoding="UTF-8" indent="no"  omit-xml-declaration="yes"/>

  <!-- Purpose: d-o-e works within streamed xsl:fork -->
  
  <xsl:mode streamable="true"/>

<xsl:template match="doc">
  <out>
    <xsl:fork>
      <xsl:sequence>
        <xsl:value-of select="a" disable-output-escaping='yes'/>
      </xsl:sequence>
      <xsl:sequence>
        <xsl:value-of select="c" disable-output-escaping='yes'/>
      </xsl:sequence>
      <xsl:sequence>
        <xsl:value-of select="e" disable-output-escaping='yes'/>
      </xsl:sequence>
      <xsl:sequence>
        <xsl:value-of select="i" disable-output-escaping='yes'/>
      </xsl:sequence>
      <xsl:sequence>
        <xsl:value-of select="b" disable-output-escaping='yes'/>
      </xsl:sequence>
      <xsl:sequence>
        <xsl:value-of select="h" disable-output-escaping='yes'/>
      </xsl:sequence>
      <xsl:sequence>
        <xsl:value-of select="f" disable-output-escaping='yes'/>
      </xsl:sequence>
      <xsl:sequence>
        <xsl:value-of select="d" disable-output-escaping='yes'/>
      </xsl:sequence>
      <xsl:sequence>
        <xsl:value-of select="g" disable-output-escaping='yes'/>
      </xsl:sequence>
 
    </xsl:fork>
  </out>
</xsl:template>
  

</xsl:stylesheet>
