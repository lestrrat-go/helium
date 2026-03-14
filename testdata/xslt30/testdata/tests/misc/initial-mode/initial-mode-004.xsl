<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:my="http://my.net/" exclude-result-prefixes="my xs">

  <!-- PURPOSE test paramters in initial-mode invocations -->

   <xsl:template match="/" mode="flobble">
    <xsl:param name="a"/>
    <xsl:param name="my:b" tunnel="yes"/>
    <out>
      <xsl:sequence select="*,$a,$my:b"/>
    </out>
  </xsl:template>



</xsl:stylesheet>
