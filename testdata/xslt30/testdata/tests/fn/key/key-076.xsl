<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:my="http://my.uri/" exclude-result-prefixes="my">
  
  <?spec fo#func-id?>
  <!-- Test that xml:id is recognized as an ID attribute -->

  <xsl:template match="doc">
    <out><xsl:value-of select="id(' id3')/title"/></out>
  </xsl:template>
  

</xsl:stylesheet>
