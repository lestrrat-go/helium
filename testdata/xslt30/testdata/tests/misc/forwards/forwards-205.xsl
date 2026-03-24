<?xml version="1.0"?> 
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
  
  <xsl:template name="main">
    <out>
      <xsl:future version="4.0">
        <xsl:transform/>
        <xsl:sort select="$donkey"/>
        <xsl:with-param no-name="x:y:z"/>
        <xsl:use-package name="urn:package:003"/>
        <xsl:fallback><in>ok1</in></xsl:fallback>
        <xsl:fallback><in>ok2</in></xsl:fallback>
        <xxx><xsl:fallback><in>wrong</in></xsl:fallback></xxx>
        <yyy/>
      </xsl:future>
    </out>
  </xsl:template>
  
</xsl:transform>
