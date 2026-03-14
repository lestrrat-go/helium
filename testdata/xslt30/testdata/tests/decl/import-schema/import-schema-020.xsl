<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
        xmlns:xs="http://www.w3.org/2001/XMLSchema"
        exclude-result-prefixes="xs">

<!-- PURPOSE: Test use of a built-in list type: NMTOKENS. -->

  

  <xsl:template match="/">
    <out>
      <xsl:variable name="t" as="attribute(*,xs:NMTOKENS)">
        <xsl:attribute name="a" type="xs:NMTOKENS" select="'red green blue'"/>
      </xsl:variable>
      <a><xsl:value-of select="string-join($t, ',')"/></a>
      <b><xsl:value-of select="$t[1] = 'green'"/></b>
      <c><xsl:value-of select="data($t) instance of xs:NMTOKEN*"/></c>
        
    </out>
  </xsl:template>
  
</xsl:stylesheet>