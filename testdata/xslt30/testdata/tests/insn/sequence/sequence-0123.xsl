<?xml version="1.0"?>

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:s="http://s.com/" exclude-result-prefixes="s">

<!-- PURPOSE:   Test xsl:sequence with a fallback child -->

<xsl:template match="doc">
  <zzz>
  <xsl:sequence select="1,2,3">
    <xsl:fallback>
      <xsl:value-of select="'1 2 3 4'"/>
    </xsl:fallback>
  </xsl:sequence>       
  </zzz>  
</xsl:template>



</xsl:stylesheet>
