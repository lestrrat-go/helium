<?xml version='1.0'?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<xsl:output method="xml"/>
<xsl:strip-space elements="*"/>

<xsl:template match="one-tag">
 <a>From Imported stylesheet: <xsl:value-of select="self::node()"/></a>
</xsl:template>

</xsl:stylesheet>