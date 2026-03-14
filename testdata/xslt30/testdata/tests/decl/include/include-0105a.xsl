<?xml version='1.0'?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<xsl:template name="two">
		 <two><xsl:value-of select="$test"/></two>
</xsl:template>

<xsl:variable name="test" select="'ERROR'"/>

</xsl:stylesheet>