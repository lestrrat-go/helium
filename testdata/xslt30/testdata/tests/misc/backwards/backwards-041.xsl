<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="1.0">

  <!-- PURPOSE: effect of BC on function call argument conversions -->

  <xsl:template match="/">
    <out><xsl:value-of select="ends-with(base-uri(//item), '_backwards-test-set.xml')"/></out>
  </xsl:template>


</xsl:stylesheet>
