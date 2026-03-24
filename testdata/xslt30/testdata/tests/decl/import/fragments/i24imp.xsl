<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">


<xsl:template match="tag">
  <pre>
    <xsl:value-of select="concat('border should be ',$color)"/>
  </pre>
</xsl:template>

</xsl:stylesheet>