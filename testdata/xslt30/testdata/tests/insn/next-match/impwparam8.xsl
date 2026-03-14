<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- FileName: impwparam8 -->
<!-- Purpose: Used by override20_022. Has xsl:param to receive param, just in case. -->


<xsl:template match="tag" priority="2">
  <xsl:param name="p1" select="'default'"/>
  <imp1-t><xsl:value-of select="$p1"/></imp1-t>
  <xsl:text>&#10;</xsl:text>
  <xsl:next-match>
    	<xsl:with-param name="p1" select="'included template'"/>
    </xsl:next-match>
</xsl:template>

</xsl:stylesheet>