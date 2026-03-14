<?xml version='1.0'?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<xsl:output method="xml"/>

<xsl:template match="one-tag">
		 <xsl:value-of select="self::node()"/>
</xsl:template>

</xsl:stylesheet>