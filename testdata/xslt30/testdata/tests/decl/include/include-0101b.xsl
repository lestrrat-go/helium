<?xml version='1.0'?>
<!DOCTYPE xsl:stylesheet SYSTEM "x/xinc01b.dtd">
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

&child;
<xsl:output method="html"/>

<xsl:template match="two-tag">
		 <xsl:value-of select="self::node()"/>
</xsl:template>



</xsl:stylesheet>
