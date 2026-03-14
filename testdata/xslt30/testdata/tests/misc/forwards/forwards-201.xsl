<?xml version="1.0"?> 
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

  <xsl:template name="main">
    <xsl:future version="4.0">
      <out>wrong</out>
      <xsl:fallback><out>ok</out></xsl:fallback>
      <out>wrong</out>
    </xsl:future>
  </xsl:template>
   
</xsl:transform>
