<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">


<xsl:template match="tag">
  <xsl:param name="p1" select="'default'"/>
  <imp-t><xsl:value-of select="$p1"/></imp-t>
</xsl:template>

<xsl:template match="bag">
  <xsl:param name="p1" select="'default'"/>
  <imp-b><xsl:value-of select="$p1"/></imp-b>
</xsl:template>

</xsl:stylesheet>