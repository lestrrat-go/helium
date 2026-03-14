<xsl:stylesheet version="1.0"
		xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

<xsl:template match="apply" priority="1">
<out><xsl:value-of select="sum(bvar/degree)+count(bvar[not(degree)])"/></out>
</xsl:template>

</xsl:stylesheet>
