<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

  <!-- Test of pattern with current() appearing inside predicate -->

<xsl:template match="text()"/>

<xsl:template match="doc">
	<xsl:apply-templates select="*"/>
</xsl:template>

<xsl:template match="text-elem[current()/@att eq '2']">
<t>
	<xsl:value-of select="string(.)"/>
</t>
</xsl:template>

</xsl:stylesheet>
