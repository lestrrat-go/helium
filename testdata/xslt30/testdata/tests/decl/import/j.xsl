<?xml version='1.0'?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<xsl:output method="xml"/>

<xsl:template match="two-tag">
 <b>From Included stylesheet: <xsl:value-of select="self::node()"/></b>
</xsl:template>

</xsl:stylesheet>